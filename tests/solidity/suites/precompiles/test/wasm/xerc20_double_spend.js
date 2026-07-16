import hre from 'hardhat'
import { expect } from 'chai'
import {
  BANK_PRECOMPILE_ADDRESS,
  BECH32_PRECOMPILE_ADDRESS,
  LARGE_GAS_LIMIT,
  WASM_DELEGATE_PRECOMPILE_ADDRESS,
  WASM_PRECOMPILE_ADDRESS,
  findEvent,
} from '../common.js'

const { ethers } = await hre.network.connect()

const INCREMENT_MSG = '{"increment":{}}'
const GET_COUNTER_QUERY = '{"get_count":{}}'
const FORCED_OUTER_REVERT_MARKER = ethers.keccak256(
  ethers.toUtf8Bytes('wasm execution completed')
)
const LOW_CALLER_GAS_LIMIT = 1_500_000n
const GAS_BURN_ITERATIONS = 5_000
// A failed nested call must leave less than 1/32 of its pre-call gas budget.
const MAX_FAILED_CALL_GAS_REMAINDER_RATIO = 32n
// The counter WASM (revert_counter_submsg) is NOT a +1 accumulator: a successful
// `increment` runs increment -> no_op submsg -> reply and lands the counter at 3,
// then saturates (further executes keep it at 3). So a shared instance cannot tell
// a committed execute (3) apart from a rolled-back one (also 3 once saturated).
// Tests that need to observe rollback/commit therefore spin up a fresh instance
// (count 0) and assert the absolute value: 3 == committed, 0 == fully rolled back.
const COUNTER_WASM_CODE_ID = BigInt(process.env.COUNTER_WASM_CODE_ID ?? '1')

function findEvents(logs, iface, eventName) {
  const events = []
  for (const log of logs) {
    try {
      const parsed = iface.parseLog(log)
      if (parsed && parsed.name === eventName) {
        events.push(parsed)
      }
    } catch {
      // ignore logs that do not match the contract interface
    }
  }
  return events
}

function normalizeAddress(address) {
  return address.toLowerCase()
}

function expectNoReceiptLogsFrom(receipt, addresses, message) {
  const blocked = new Set(addresses.map(normalizeAddress))
  const matches = receipt.logs.filter((log) =>
    blocked.has(normalizeAddress(log.address))
  )
  expect(matches, message).to.have.length(0)
}

function expectReceiptLogsFrom(receipt, address, message) {
  const matches = receipt.logs.filter(
    (log) => normalizeAddress(log.address) === normalizeAddress(address)
  )
  expect(matches, message).not.to.have.length(0)
}

async function expectNoProviderLogsFrom(addresses, receipt) {
  for (const address of addresses) {
    const logs = await ethers.provider.getLogs({
      address,
      fromBlock: receipt.blockNumber,
      toBlock: receipt.blockNumber,
    })
    expect(logs, `eth_getLogs must not expose reverted logs for ${address}`).to
      .have.length(0)
  }
}

function expectTransfer(events, from, to, value) {
  const match = events.find(
    (event) =>
      normalizeAddress(event.args.from) === normalizeAddress(from) &&
      normalizeAddress(event.args.to) === normalizeAddress(to) &&
      event.args.value === value
  )
  expect(match, `missing Transfer(${from}, ${to}, ${value})`).to.exist
}

async function resolveWasmAddress(addr) {
  const converter = await ethers.getContractAt(
    'Bech32I',
    BECH32_PRECOMPILE_ADDRESS
  )

  if (addr.startsWith('xpla')) {
    return {
      bech32: addr,
      hex: await converter.bech32ToHex.staticCall(addr),
    }
  }

  return {
    bech32: await converter.hexToBech32.staticCall(addr, 'xpla'),
    hex: addr,
  }
}

describe('xerc20 wasm precompile accounting', function () {
  let bankSendWasm
  let bech32

  before(async function () {
    const counterAddr = process.env.COUNTER_WASM_ADDRESS
    const bankSendAddr = process.env.XERC20_BANK_SEND_WASM_ADDRESS
    if (!counterAddr || !bankSendAddr) {
      throw new Error(
        'COUNTER_WASM_ADDRESS and XERC20_BANK_SEND_WASM_ADDRESS are required'
      )
    }

    bankSendWasm = await resolveWasmAddress(bankSendAddr)
    bech32 = await ethers.getContractAt(
      'Bech32I',
      BECH32_PRECOMPILE_ADDRESS
    )
  })

  async function getCounter(target) {
    const wasm = await ethers.getContractAt(
      'IWasm',
      WASM_PRECOMPILE_ADDRESS
    )
    const data = await wasm.smartContractState.staticCall(
      target.hex,
      ethers.toUtf8Bytes(GET_COUNTER_QUERY)
    )
    const decoded = JSON.parse(ethers.toUtf8String(data))
    const counter = decoded.counter ?? decoded.count
    if (counter === undefined) {
      throw new Error('counter query response is missing a counter value')
    }
    return BigInt(counter)
  }

  async function deployAdversarialFixture() {
    const [deployer, callbackRecipient] = await ethers.getSigners()
    const initialSupply = ethers.parseEther('100')
    const transferAmount = ethers.parseEther('10')
    const callbackAmount = ethers.parseEther('1')

    const Token = await ethers.getContractFactory('AdversarialXerc20')
    const token = await Token.deploy(initialSupply)
    await token.waitForDeployment()

    const tokenAddress = await token.getAddress()
    const denom = `xerc20:${tokenAddress.toLowerCase()}`

    const Callback = await ethers.getContractFactory('Xerc20ReentryTarget')
    const callback = await Callback.deploy(tokenAddress, denom)
    await callback.waitForDeployment()

    const PoC = await ethers.getContractFactory('BankXerc20DoubleSpendPoC')
    const poc = await PoC.deploy(tokenAddress, denom)
    await poc.waitForDeployment()

    await (await token.transfer(await poc.getAddress(), transferAmount)).wait()
    await (await token.transfer(await callback.getAddress(), callbackAmount)).wait()

    return {
      deployer,
      callbackRecipient,
      token,
      callback,
      poc,
      transferAmount,
      callbackAmount,
    }
  }

  // Instantiate a brand-new counter instance (count 0) so its absolute value is a
  // reliable commit/rollback witness, instead of the shared instance saturated at 3.
  async function instantiateFreshCounter() {
    const [deployer] = await ethers.getSigners()
    const wasm = await ethers.getContractAt('IWasm', WASM_PRECOMPILE_ADDRESS)
    const tx = await wasm.instantiateContract(
      deployer.address,
      ethers.ZeroAddress,
      COUNTER_WASM_CODE_ID,
      'fresh_counter',
      ethers.toUtf8Bytes('{}'),
      [],
      { gasLimit: LARGE_GAS_LIMIT }
    )
    const receipt = await tx.wait()
    const event = findEvent(receipt.logs, wasm.interface, 'InstantiateContract')
    expect(event, 'InstantiateContract event must be emitted').to.exist
    return { hex: event.args.contractAddress }
  }

  it('rejects double-crediting via wasm xerc20 funds followed by direct transfer', async function () {
    const [deployer, directRecipient] = await ethers.getSigners()

    const amount = ethers.parseEther('10')
    const initialSupply = ethers.parseEther('100')

    const Token = await ethers.getContractFactory('PoCToken')
    const token = await Token.deploy(initialSupply)
    await token.waitForDeployment()

    const tokenAddress = await token.getAddress()
    const denom = `xerc20:${tokenAddress.toLowerCase()}`

    const PoC = await ethers.getContractFactory('BankXerc20DoubleSpendPoC')
    const poc = await PoC.deploy(tokenAddress, denom)
    await poc.waitForDeployment()

    const pocAddress = await poc.getAddress()
    await (await token.transfer(pocAddress, amount)).wait()
    const totalSupplyBefore = await token.totalSupply()
    const freshCounter = await instantiateFreshCounter()
    expect(await getCounter(freshCounter)).to.equal(0n)

    await expect(
      poc.exploitViaWasmFunds(
        freshCounter.hex,
        ethers.toUtf8Bytes(INCREMENT_MSG),
        directRecipient.address,
        amount,
        { gasLimit: LARGE_GAS_LIMIT }
      )
    ).to.revert(ethers)

    const deployerBalance = await token.balanceOf(deployer.address)
    const pocBalance = await token.balanceOf(pocAddress)
    const wasmBalance = await token.balanceOf(freshCounter.hex)
    const directRecipientBalance = await token.balanceOf(directRecipient.address)
    const totalSupplyAfter = await token.totalSupply()

    expect(pocBalance).to.equal(amount)
    expect(wasmBalance).to.equal(0n)
    expect(directRecipientBalance).to.equal(0n)
    expect(totalSupplyAfter).to.equal(totalSupplyBefore)
    expect(
      deployerBalance + pocBalance + wasmBalance + directRecipientBalance
    ).to.equal(totalSupplyAfter)
    expect(await getCounter(freshCounter)).to.equal(0n)
  })

  it('keeps native funds accounting visible after wasm sends xerc20 through BankMsg', async function () {
    const [deployer, xerc20Recipient] = await ethers.getSigners()

    const nativeAmount = ethers.parseEther('1')
    const xerc20Amount = ethers.parseEther('10')
    const initialSupply = ethers.parseEther('100')

    const Token = await ethers.getContractFactory('PoCToken')
    const token = await Token.deploy(initialSupply)
    await token.waitForDeployment()

    const tokenAddress = await token.getAddress()
    const denom = `xerc20:${tokenAddress.toLowerCase()}`

    const PoC = await ethers.getContractFactory('BankXerc20DoubleSpendPoC')
    const poc = await PoC.deploy(tokenAddress, denom)
    await poc.waitForDeployment()

    const pocAddress = await poc.getAddress()
    await (
      await deployer.sendTransaction({
        to: pocAddress,
        value: nativeAmount,
      })
    ).wait()
    await (await token.transfer(bankSendWasm.hex, xerc20Amount)).wait()

    const xerc20RecipientBech32 = await bech32.hexToBech32.staticCall(
      xerc20Recipient.address,
      'xpla'
    )
    const wasmMsg = JSON.stringify({
      send_xerc20: {
        token: tokenAddress.toLowerCase(),
        recipient: xerc20RecipientBech32,
        amount: xerc20Amount.toString(),
      },
    })

    const totalSupplyBefore = await token.totalSupply()
    const wasmNativeBefore = await ethers.provider.getBalance(bankSendWasm.hex)

    const tx = await poc.executeWasmNativeFundsAndAssertBalance(
      bankSendWasm.hex,
      ethers.toUtf8Bytes(wasmMsg),
      nativeAmount,
      { gasLimit: LARGE_GAS_LIMIT }
    )
    await tx.wait()

    expect(await ethers.provider.getBalance(pocAddress)).to.equal(0n)
    expect(await ethers.provider.getBalance(bankSendWasm.hex)).to.equal(
      wasmNativeBefore + nativeAmount
    )
    expect(await token.balanceOf(bankSendWasm.hex)).to.equal(0n)
    expect(await token.balanceOf(xerc20Recipient.address)).to.equal(
      xerc20Amount
    )
    expect(await token.totalSupply()).to.equal(totalSupplyBefore)
  })

  it('keeps xerc20 Funds atomic when the token re-enters the bank precompile', async function () {
    const {
      callbackRecipient,
      token,
      callback,
      poc,
      transferAmount,
      callbackAmount,
    } = await deployAdversarialFixture()
    const totalSupplyBefore = await token.totalSupply()
    const pocAddress = await poc.getAddress()
    const callbackAddress = await callback.getAddress()
    const freshCounter = await instantiateFreshCounter()
    expect(await getCounter(freshCounter)).to.equal(0n)

    await (
      await callback.configure(
        callbackRecipient.address,
        callbackAmount,
        false
      )
    ).wait()
    await (
      await token.configureCallback(
        await callback.getAddress(),
        callback.interface.encodeFunctionData('onXerc20Transfer'),
        false,
        0
      )
    ).wait()

    const tx = await poc.executeWasmFunds(
      freshCounter.hex,
      ethers.toUtf8Bytes(INCREMENT_MSG),
      transferAmount,
      { gasLimit: LARGE_GAS_LIMIT }
    )
    const receipt = await tx.wait()

    expectReceiptLogsFrom(
      receipt,
      BANK_PRECOMPILE_ADDRESS,
      'successful bank reentry must emit a bank precompile log'
    )
    expect(findEvent(receipt.logs, token.interface, 'TransferProbe')).to.exist
    expect(findEvent(receipt.logs, callback.interface, 'CallbackEntered')).to.exist
    const reentry = findEvent(
      receipt.logs,
      callback.interface,
      'ReentryCompleted'
    )
    expect(reentry).to.exist
    expect(reentry.args.success).to.equal(true)

    const callbackResult = findEvent(
      receipt.logs,
      token.interface,
      'CallbackResult'
    )
    expect(callbackResult).to.exist
    expect(callbackResult.args.success).to.equal(true)

    const transfers = findEvents(receipt.logs, token.interface, 'Transfer')
    expect(transfers).to.have.length(2)
    expectTransfer(
      transfers,
      callbackAddress,
      callbackRecipient.address,
      callbackAmount
    )
    expectTransfer(transfers, pocAddress, freshCounter.hex, transferAmount)

    expect(await token.balanceOf(pocAddress)).to.equal(0n)
    expect(await token.balanceOf(freshCounter.hex)).to.equal(transferAmount)
    expect(await token.balanceOf(callbackAddress)).to.equal(0n)
    expect(await token.balanceOf(callbackRecipient.address)).to.equal(
      callbackAmount
    )
    expect(await token.totalSupply()).to.equal(totalSupplyBefore)
    // Full success path: the wasm execute commits, so the fresh counter must
    // reach its success value of 3 (0 would mean the increment was lost/rolled back).
    expect(await getCounter(freshCounter)).to.equal(3n)
  })

  it('rolls back a bank reentry when the token catches the callback revert', async function () {
    const {
      callbackRecipient,
      token,
      callback,
      poc,
      transferAmount,
      callbackAmount,
    } = await deployAdversarialFixture()
    const totalSupplyBefore = await token.totalSupply()
    const pocAddress = await poc.getAddress()
    const callbackAddress = await callback.getAddress()
    const freshCounter = await instantiateFreshCounter()
    expect(await getCounter(freshCounter)).to.equal(0n)

    await (
      await callback.configure(
        callbackRecipient.address,
        callbackAmount,
        true
      )
    ).wait()
    await (
      await token.configureCallback(
        callbackAddress,
        callback.interface.encodeFunctionData('onXerc20Transfer'),
        true,
        0
      )
    ).wait()

    const tx = await poc.executeWasmFunds(
      freshCounter.hex,
      ethers.toUtf8Bytes(INCREMENT_MSG),
      transferAmount,
      { gasLimit: LARGE_GAS_LIMIT }
    )
    const receipt = await tx.wait()

    const callbackResult = findEvent(
      receipt.logs,
      token.interface,
      'CallbackResult'
    )
    expect(callbackResult).to.exist
    expect(callbackResult.args.success).to.equal(false)
    const callbackError = callback.interface.parseError(
      callbackResult.args.returnData
    )
    expect(callbackError.name).to.equal('AdversarialCallbackRevert')
    expect(findEvent(receipt.logs, token.interface, 'PoisonLog')).to.exist

    expectNoReceiptLogsFrom(
      receipt,
      [callbackAddress, BANK_PRECOMPILE_ADDRESS],
      'callback and bank logs emitted inside the reverted callback frame must be removed'
    )
    await expectNoProviderLogsFrom(
      [callbackAddress, BANK_PRECOMPILE_ADDRESS],
      receipt
    )

    const transfers = findEvents(receipt.logs, token.interface, 'Transfer')
    expect(transfers).to.have.length(1)
    expectTransfer(transfers, pocAddress, freshCounter.hex, transferAmount)

    expect(await token.balanceOf(pocAddress)).to.equal(0n)
    expect(await token.balanceOf(freshCounter.hex)).to.equal(transferAmount)
    expect(await token.balanceOf(callbackAddress)).to.equal(callbackAmount)
    expect(await token.balanceOf(callbackRecipient.address)).to.equal(0n)
    expect(await token.totalSupply()).to.equal(totalSupplyBefore)
    // Only the inner bank reentry rolls back; the outer wasm execute still
    // commits, so the fresh counter must reach its success value of 3.
    expect(await getCounter(freshCounter)).to.equal(3n)
  })

  it('removes callback-frame logs when the token catches the callback revert without reentry', async function () {
    const { token, callback, poc, transferAmount } =
      await deployAdversarialFixture()
    const freshCounter = await instantiateFreshCounter()
    expect(await getCounter(freshCounter)).to.equal(0n)

    await (await callback.configure(ethers.ZeroAddress, 0, true)).wait()
    await (
      await token.configureCallback(
        await callback.getAddress(),
        callback.interface.encodeFunctionData('onXerc20Transfer'),
        true,
        0
      )
    ).wait()

    const tx = await poc.executeWasmFunds(
      freshCounter.hex,
      ethers.toUtf8Bytes(INCREMENT_MSG),
      transferAmount,
      { gasLimit: LARGE_GAS_LIMIT }
    )
    const receipt = await tx.wait()

    const callbackResult = findEvent(
      receipt.logs,
      token.interface,
      'CallbackResult'
    )
    expect(callbackResult).to.exist
    expect(callbackResult.args.success).to.equal(false)
    expect(findEvent(receipt.logs, token.interface, 'PoisonLog')).to.exist

    const callbackAddress = (await callback.getAddress()).toLowerCase()
    expect(
      receipt.logs.some((log) => log.address.toLowerCase() === callbackAddress),
      'logs emitted inside the reverted callback frame must be removed'
    ).to.equal(false)

    expect(await token.balanceOf(await poc.getAddress())).to.equal(0n)
    expect(await token.balanceOf(freshCounter.hex)).to.equal(transferAmount)
    // Caught callback revert only drops the callback frame; the wasm execute
    // commits, so the fresh counter must reach its success value of 3.
    expect(await getCounter(freshCounter)).to.equal(3n)
  })

  it('removes ERC20 and callback logs when the wasm precompile revert is caught', async function () {
    const { token, callback, poc, transferAmount } =
      await deployAdversarialFixture()
    const freshCounter = await instantiateFreshCounter()
    expect(await getCounter(freshCounter)).to.equal(0n)

    await (await callback.configure(ethers.ZeroAddress, 0, true)).wait()
    await (
      await token.configureCallback(
        await callback.getAddress(),
        callback.interface.encodeFunctionData('onXerc20Transfer'),
        false,
        0
      )
    ).wait()

    const tx = await poc.tryExecuteWasmFunds(
      freshCounter.hex,
      ethers.toUtf8Bytes(INCREMENT_MSG),
      transferAmount,
      { gasLimit: LARGE_GAS_LIMIT }
    )
    const receipt = await tx.wait()

    const callResult = findEvent(
      receipt.logs,
      poc.interface,
      'WasmFundsCallResult'
    )
    expect(callResult).to.exist
    expect(callResult.args.success).to.equal(false)

    expect(findEvent(receipt.logs, token.interface, 'TransferProbe')).to.equal(
      null
    )
    expect(findEvent(receipt.logs, token.interface, 'PoisonLog')).to.equal(null)
    expect(
      findEvent(receipt.logs, callback.interface, 'CallbackEntered')
    ).to.equal(null)

    const tokenLogs = await ethers.provider.getLogs({
      address: await token.getAddress(),
      fromBlock: receipt.blockNumber,
      toBlock: receipt.blockNumber,
    })
    expect(tokenLogs).to.have.length(0)

    expect(await token.balanceOf(await poc.getAddress())).to.equal(
      transferAmount
    )
    expect(await token.balanceOf(freshCounter.hex)).to.equal(0n)
    expect(await getCounter(freshCounter)).to.equal(0n)
  })

  it('rolls back wasm, token, and callback artifacts when the caller frame reverts after wasm success', async function () {
    const {
      callbackRecipient,
      token,
      callback,
      poc,
      transferAmount,
      callbackAmount,
    } = await deployAdversarialFixture()
    const freshCounter = await instantiateFreshCounter()
    expect(await getCounter(freshCounter)).to.equal(0n)
    const totalSupplyBefore = await token.totalSupply()
    const pocAddress = await poc.getAddress()
    const callbackAddress = await callback.getAddress()

    await (
      await callback.configure(
        callbackRecipient.address,
        callbackAmount,
        false
      )
    ).wait()
    await (
      await token.configureCallback(
        callbackAddress,
        callback.interface.encodeFunctionData('onXerc20Transfer'),
        false,
        0
      )
    ).wait()

    const tx = await poc.tryExecuteWasmFundsThenRevert(
      freshCounter.hex,
      ethers.toUtf8Bytes(INCREMENT_MSG),
      transferAmount,
      { gasLimit: LARGE_GAS_LIMIT }
    )
    const receipt = await tx.wait()

    const callResult = findEvent(
      receipt.logs,
      poc.interface,
      'WasmFundsCallResult'
    )
    expect(callResult).to.exist
    expect(callResult.args.success).to.equal(false)
    const forcedRevert = poc.interface.parseError(callResult.args.returnData)
    expect(forcedRevert.name).to.equal('ForcedOuterRevertAfterWasm')
    expect(forcedRevert.args.marker).to.equal(FORCED_OUTER_REVERT_MARKER)

    expectNoReceiptLogsFrom(
      receipt,
      [
        await token.getAddress(),
        callbackAddress,
        BANK_PRECOMPILE_ADDRESS,
        WASM_PRECOMPILE_ADDRESS,
      ],
      'logs emitted inside the reverted caller frame must be removed'
    )
    await expectNoProviderLogsFrom(
      [
        await token.getAddress(),
        callbackAddress,
        BANK_PRECOMPILE_ADDRESS,
        WASM_PRECOMPILE_ADDRESS,
      ],
      receipt
    )

    expect(await token.balanceOf(pocAddress)).to.equal(transferAmount)
    expect(await token.balanceOf(freshCounter.hex)).to.equal(0n)
    expect(await token.balanceOf(callbackAddress)).to.equal(callbackAmount)
    expect(await token.balanceOf(callbackRecipient.address)).to.equal(0n)
    expect(await token.totalSupply()).to.equal(totalSupplyBefore)
    // The outer caller frame reverts after wasm success: a correct rollback
    // leaves the fresh counter at 0. A nonzero value means wasm state leaked
    // out of the reverted frame (state contamination).
    const counterAfter = await getCounter(freshCounter)
    expect(
      counterAfter,
      `wasm counter leaked across reverted caller frame: expected 0, got ${counterAfter}`
    ).to.equal(0n)
  })

  it('commits xerc20 funds through the delegate wasm precompile address', async function () {
    const { deployer, token, poc, transferAmount } =
      await deployAdversarialFixture()
    const freshCounter = await instantiateFreshCounter()
    expect(await getCounter(freshCounter)).to.equal(0n)

    const senderBalanceBefore = await token.balanceOf(deployer.address)
    const counterBalanceBefore = await token.balanceOf(freshCounter.hex)

    const tx = await poc.executeDelegateWasmFunds(
      freshCounter.hex,
      ethers.toUtf8Bytes(INCREMENT_MSG),
      transferAmount,
      { gasLimit: LARGE_GAS_LIMIT }
    )
    const receipt = await tx.wait()

    expect(await getCounter(freshCounter)).to.equal(3n)
    expect(await token.balanceOf(deployer.address)).to.equal(
      senderBalanceBefore - transferAmount
    )
    expect(await token.balanceOf(freshCounter.hex)).to.equal(
      counterBalanceBefore + transferAmount
    )

    const transfers = findEvents(receipt.logs, token.interface, 'Transfer')
    expectTransfer(
      transfers,
      deployer.address,
      freshCounter.hex,
      transferAmount
    )

    const wasm = await ethers.getContractAt('IWasm', WASM_PRECOMPILE_ADDRESS)
    const executeContractTopic = wasm.interface.getEvent(
      'ExecuteContract'
    ).topicHash
    const executeContractLog = receipt.logs.find(
      (log) => log.topics[0] === executeContractTopic
    )
    expect(executeContractLog, 'delegate wasm execute log must be emitted').to
      .exist
    expect(normalizeAddress(executeContractLog.address)).to.equal(
      normalizeAddress(WASM_PRECOMPILE_ADDRESS)
    )
  })

  it('rolls back xerc20 funds through the delegate wasm precompile address', async function () {
    const { deployer, token, poc, transferAmount } =
      await deployAdversarialFixture()
    const freshCounter = await instantiateFreshCounter()
    expect(await getCounter(freshCounter)).to.equal(0n)

    const pocAddress = await poc.getAddress()
    const totalSupplyBefore = await token.totalSupply()
    const senderTokenBalanceBefore = await token.balanceOf(deployer.address)
    const counterTokenBalanceBefore = await token.balanceOf(freshCounter.hex)
    const pocTokenBalanceBefore = await token.balanceOf(pocAddress)
    const counterNativeBalanceBefore = await ethers.provider.getBalance(
      freshCounter.hex
    )
    const pocNativeBalanceBefore = await ethers.provider.getBalance(pocAddress)

    const tx = await poc.tryExecuteDelegateWasmFundsThenRevert(
      freshCounter.hex,
      ethers.toUtf8Bytes(INCREMENT_MSG),
      transferAmount,
      { gasLimit: LARGE_GAS_LIMIT }
    )
    const receipt = await tx.wait()

    const callResult = findEvent(
      receipt.logs,
      poc.interface,
      'WasmFundsCallResult'
    )
    expect(callResult).to.exist
    expect(callResult.args.success).to.equal(false)
    const forcedRevert = poc.interface.parseError(callResult.args.returnData)
    expect(forcedRevert.name).to.equal('ForcedOuterRevertAfterWasm')
    expect(forcedRevert.args.marker).to.equal(FORCED_OUTER_REVERT_MARKER)

    const revertedLogAddresses = [
      await token.getAddress(),
      BANK_PRECOMPILE_ADDRESS,
      WASM_PRECOMPILE_ADDRESS,
      WASM_DELEGATE_PRECOMPILE_ADDRESS,
    ]
    expectNoReceiptLogsFrom(
      receipt,
      revertedLogAddresses,
      'delegate wasm caller revert must remove token, bank, and wasm logs'
    )
    await expectNoProviderLogsFrom(revertedLogAddresses, receipt)

    expect(await token.balanceOf(deployer.address)).to.equal(
      senderTokenBalanceBefore
    )
    expect(await token.balanceOf(freshCounter.hex)).to.equal(
      counterTokenBalanceBefore
    )
    expect(await token.balanceOf(pocAddress)).to.equal(pocTokenBalanceBefore)
    expect(await token.totalSupply()).to.equal(totalSupplyBefore)
    expect(await ethers.provider.getBalance(freshCounter.hex)).to.equal(
      counterNativeBalanceBefore
    )
    expect(await ethers.provider.getBalance(pocAddress)).to.equal(
      pocNativeBalanceBefore
    )
    expect(await getCounter(freshCounter)).to.equal(0n)
  })

  it('bounds the nested ERC20 call gas by the wasm precompile caller frame', async function () {
    const { token, poc, transferAmount } = await deployAdversarialFixture()
    const freshCounter = await instantiateFreshCounter()
    expect(await getCounter(freshCounter)).to.equal(0n)

    const tx = await poc.executeWasmFunds(
      freshCounter.hex,
      ethers.toUtf8Bytes(INCREMENT_MSG),
      transferAmount,
      { gasLimit: LOW_CALLER_GAS_LIMIT }
    )
    const receipt = await tx.wait()
    const probe = findEvent(receipt.logs, token.interface, 'TransferProbe')
    expect(probe).to.exist

    const gasProbe = findEvent(
      receipt.logs,
      poc.interface,
      'WasmFundsGasProbe'
    )
    expect(gasProbe).to.exist
    expect(
      gasProbe.args.gasBeforeCall < LOW_CALLER_GAS_LIMIT,
      `wasm call gas ${gasProbe.args.gasBeforeCall} must be below tx gas limit ${LOW_CALLER_GAS_LIMIT}`
    ).to.equal(true)

    expect(
      probe.args.gasAtEntry <= gasProbe.args.gasBeforeCall,
      `nested ERC20 gas ${probe.args.gasAtEntry} exceeds caller frame gas ${gasProbe.args.gasBeforeCall}`
    ).to.equal(true)
    expect(await getCounter(freshCounter)).to.equal(3n)
  })

  it('calibrates token gas burn and fails atomically when it exceeds the caller frame', async function () {
    const calibration = await deployAdversarialFixture()
    await (
      await calibration.token.configureCallback(
        ethers.ZeroAddress,
        '0x',
        false,
        GAS_BURN_ITERATIONS
      )
    ).wait()
    const directTransferGas = await calibration.token.transfer.estimateGas(
      calibration.callbackRecipient.address,
      1n
    )
    expect(
      directTransferGas > LOW_CALLER_GAS_LIMIT,
      `direct ERC20 transfer gas ${directTransferGas} must exceed the low caller limit ${LOW_CALLER_GAS_LIMIT}`
    ).to.equal(true)
    expect(
      directTransferGas < BigInt(LARGE_GAS_LIMIT),
      `direct ERC20 transfer gas ${directTransferGas} must fit within the high-gas positive control ${LARGE_GAS_LIMIT}`
    ).to.equal(true)

    const highGas = await deployAdversarialFixture()
    const highGasBurnBefore = await highGas.token.burnAccumulator()
    const highGasCounter = await instantiateFreshCounter()
    expect(await getCounter(highGasCounter)).to.equal(0n)

    await (
      await highGas.token.configureCallback(
        ethers.ZeroAddress,
        '0x',
        false,
        GAS_BURN_ITERATIONS
      )
    ).wait()

    const highGasTx = await highGas.poc.executeWasmFunds(
      highGasCounter.hex,
      ethers.toUtf8Bytes(INCREMENT_MSG),
      highGas.transferAmount,
      { gasLimit: LARGE_GAS_LIMIT }
    )
    const highGasReceipt = await highGasTx.wait()
    expect(findEvent(highGasReceipt.logs, highGas.token.interface, 'TransferProbe'))
      .to.exist
    expect(await highGas.token.burnAccumulator()).to.not.equal(
      highGasBurnBefore
    )
    expect(await highGas.token.balanceOf(await highGas.poc.getAddress())).to.equal(
      0n
    )
    expect(await highGas.token.balanceOf(highGasCounter.hex)).to.equal(
      highGas.transferAmount
    )
    // High-gas positive control: the execute commits, so counter must reach 3.
    expect(await getCounter(highGasCounter)).to.equal(3n)

    const { token, poc, transferAmount } = await deployAdversarialFixture()
    const burnAccumulatorBefore = await token.burnAccumulator()
    const lowGasCounter = await instantiateFreshCounter()
    expect(await getCounter(lowGasCounter)).to.equal(0n)

    await (
      await token.configureCallback(
        ethers.ZeroAddress,
        '0x',
        false,
        GAS_BURN_ITERATIONS
      )
    ).wait()

    const tx = await poc.tryExecuteWasmFunds(
      lowGasCounter.hex,
      ethers.toUtf8Bytes(INCREMENT_MSG),
      transferAmount,
      { gasLimit: LOW_CALLER_GAS_LIMIT }
    )
    const receipt = await tx.wait()

    const callResult = findEvent(
      receipt.logs,
      poc.interface,
      'WasmFundsCallResult'
    )
    expect(callResult).to.exist
    expect(callResult.args.success).to.equal(false)

    const gasProbe = findEvent(
      receipt.logs,
      poc.interface,
      'WasmFundsGasProbe'
    )
    expect(gasProbe).to.exist
    expect(
      gasProbe.args.gasBeforeCall < LOW_CALLER_GAS_LIMIT,
      `wasm call gas ${gasProbe.args.gasBeforeCall} must be below tx gas limit ${LOW_CALLER_GAS_LIMIT}`
    ).to.equal(true)

    expect(
      gasProbe.args.gasAfterCall * MAX_FAILED_CALL_GAS_REMAINDER_RATIO <
        gasProbe.args.gasBeforeCall,
      `failed wasm call did not charge the outer frame enough: gas before ${gasProbe.args.gasBeforeCall}, gas after ${gasProbe.args.gasAfterCall}`
    ).to.equal(true)

    expect(findEvent(receipt.logs, token.interface, 'TransferProbe')).to.equal(
      null
    )
    expect(await token.burnAccumulator()).to.equal(burnAccumulatorBefore)
    expect(await token.balanceOf(await poc.getAddress())).to.equal(
      transferAmount
    )
    expect(await token.balanceOf(lowGasCounter.hex)).to.equal(0n)
    // Low-gas execute reverts and is caught: the whole wasm execute rolls back,
    // so the fresh counter must remain at 0 (no partial/contaminated state).
    const counterAfter = await getCounter(lowGasCounter)
    expect(
      counterAfter,
      `wasm counter changed across the low-gas reverted call from 0 to ${counterAfter}`
    ).to.equal(0n)
  })
})

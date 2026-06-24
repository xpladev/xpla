import hre from 'hardhat'
import { expect } from 'chai'
import {
  BECH32_PRECOMPILE_ADDRESS,
  LARGE_GAS_LIMIT,
} from '../common.js'

const { ethers } = await hre.network.connect()

const INCREMENT_MSG = '{"increment":{}}'

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
  let counterWasm
  let bankSendWasm
  let bech32

  before(async function () {
    const counterAddr = process.env.COUNTER_WASM_ADDRESS
    const bankSendAddr = process.env.XERC20_BANK_SEND_WASM_ADDRESS
    if (!counterAddr || !bankSendAddr) {
      this.skip()
      return
    }

    counterWasm = await resolveWasmAddress(counterAddr)
    bankSendWasm = await resolveWasmAddress(bankSendAddr)
    bech32 = await ethers.getContractAt(
      'Bech32I',
      BECH32_PRECOMPILE_ADDRESS
    )
  })

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

    await expect(
      poc.exploitViaWasmFunds(
        counterWasm.hex,
        ethers.toUtf8Bytes(INCREMENT_MSG),
        directRecipient.address,
        amount,
        { gasLimit: LARGE_GAS_LIMIT }
      )
    ).to.revert(ethers)

    const deployerBalance = await token.balanceOf(deployer.address)
    const pocBalance = await token.balanceOf(pocAddress)
    const wasmBalance = await token.balanceOf(counterWasm.hex)
    const directRecipientBalance = await token.balanceOf(directRecipient.address)
    const totalSupplyAfter = await token.totalSupply()

    expect(pocBalance).to.equal(amount)
    expect(wasmBalance).to.equal(0n)
    expect(directRecipientBalance).to.equal(0n)
    expect(totalSupplyAfter).to.equal(totalSupplyBefore)
    expect(
      deployerBalance + pocBalance + wasmBalance + directRecipientBalance
    ).to.equal(totalSupplyAfter)
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
})

import hre from 'hardhat'
import { expect } from 'chai'
import { BANK_PRECOMPILE_ADDRESS, LARGE_GAS_LIMIT, findEvent, waitWithTimeout } from '../common.js'

const { ethers } = await hre.network.connect()

describe('xerc20 bank precompile double spend PoC', function () {
  it('rejects double-crediting via bank send followed by direct transfer', async function () {
    const [deployer, bankRecipient, directRecipient] = await ethers.getSigners()

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

    const tx = await poc.exploit(
      bankRecipient.address,
      directRecipient.address,
      amount,
      { gasLimit: LARGE_GAS_LIMIT }
    )
    const receipt = await waitWithTimeout(tx, 20000)
    expect(receipt.status).to.equal(0)

    const deployerBalance = await token.balanceOf(deployer.address)
    const pocBalance = await token.balanceOf(pocAddress)
    const bankRecipientBalance = await token.balanceOf(bankRecipient.address)
    const directRecipientBalance = await token.balanceOf(directRecipient.address)
    const totalSupplyAfter = await token.totalSupply()

    expect(pocBalance).to.equal(amount)
    expect(bankRecipientBalance).to.equal(0n)
    expect(directRecipientBalance).to.equal(0n)
    expect(totalSupplyAfter).to.equal(totalSupplyBefore)
    expect(
      deployerBalance +
        pocBalance +
        bankRecipientBalance +
        directRecipientBalance
    ).to.equal(totalSupplyAfter)
  })

  it('charges failed nested ERC20 gas and rolls back the registered bank call', async function () {
    const [deployer, recipient] = await ethers.getSigners()
    const amount = ethers.parseEther('10')
    const lowGasLimit = 1_500_000n
    const burnIterations = 5_000

    async function deployBurningToken() {
      const Token = await ethers.getContractFactory('AdversarialXerc20')
      const token = await Token.deploy(ethers.parseEther('100'))
      await token.waitForDeployment()
      const tokenAddress = await token.getAddress()
      const PoC = await ethers.getContractFactory('BankXerc20DoubleSpendPoC')
      const poc = await PoC.deploy(tokenAddress, `xerc20:${tokenAddress.toLowerCase()}`)
      await poc.waitForDeployment()
      await (await token.transfer(await poc.getAddress(), amount)).wait()
      await (await token.configureCallback(ethers.ZeroAddress, '0x', false, burnIterations)).wait()
      return { token, poc }
    }

    const high = await deployBurningToken()
    const directTransferGas = await high.token.transfer.estimateGas(recipient.address, amount)
    expect(directTransferGas > lowGasLimit, 'burn must exhaust the low caller frame').to.equal(true)
    expect(directTransferGas < BigInt(LARGE_GAS_LIMIT), 'burn must fit the positive control').to.equal(true)
    const highBurnBefore = await high.token.burnAccumulator()
    const highReceipt = await (await high.poc.tryBankSend(recipient.address, amount, {
      gasLimit: LARGE_GAS_LIMIT,
    })).wait()
    const highResult = findEvent(highReceipt.logs, high.poc.interface, 'BankSendCallResult')
    expect(highReceipt.status).to.equal(1)
    expect(highResult.args.success).to.equal(true)
    expect(ethers.AbiCoder.defaultAbiCoder().decode(['bool'], highResult.args.returnData)[0]).to.equal(true)
    const highProbe = findEvent(highReceipt.logs, high.poc.interface, 'BankSendGasProbe')
    const tokenProbe = findEvent(highReceipt.logs, high.token.interface, 'TransferProbe')
    expect(tokenProbe).to.exist
    expect(tokenProbe.args.gasAtEntry <= highProbe.args.gasBeforeCall).to.equal(true)
    expect(await high.token.burnAccumulator()).to.not.equal(highBurnBefore)
    expect(await high.token.balanceOf(await high.poc.getAddress())).to.equal(0n)
    expect(await high.token.balanceOf(recipient.address)).to.equal(amount)

    const { token, poc } = await deployBurningToken()
    const pocAddress = await poc.getAddress()
    const tokenAddress = await token.getAddress()
    const senderBefore = await token.balanceOf(deployer.address)
    const supplyBefore = await token.totalSupply()
    const burnBefore = await token.burnAccumulator()
    const receipt = await (await poc.tryBankSend(recipient.address, amount, {
      gasLimit: lowGasLimit,
    })).wait()
    const result = findEvent(receipt.logs, poc.interface, 'BankSendCallResult')
    const probe = findEvent(receipt.logs, poc.interface, 'BankSendGasProbe')
    expect(receipt.status).to.equal(1) // the outer caller catches the failed precompile
    expect(result.args.success).to.equal(false)
    expect(result.args.returnData.slice(0, 10)).to.equal('0x08c379a0')
    const [failureReason] = ethers.AbiCoder.defaultAbiCoder().decode(
      ['string'], `0x${result.args.returnData.slice(10)}`
    )
    expect(failureReason).to.include('out of gas')
    expect(probe).to.exist

    // Check rollback before the regression assertion, so RED still proves atomicity.
    expect(await token.balanceOf(pocAddress)).to.equal(amount)
    expect(await token.balanceOf(recipient.address)).to.equal(0n)
    expect(await token.balanceOf(deployer.address)).to.equal(senderBefore)
    expect(await token.totalSupply()).to.equal(supplyBefore)
    expect(await token.burnAccumulator()).to.equal(burnBefore)
    for (const address of [tokenAddress, BANK_PRECOMPILE_ADDRESS]) {
      expect(receipt.logs.filter(log => log.address.toLowerCase() === address.toLowerCase())).to.have.length(0)
      expect(await ethers.provider.getLogs({
        address, fromBlock: receipt.blockNumber, toBlock: receipt.blockNumber,
      })).to.have.length(0)
    }

    expect(probe.args.gasBeforeCall < lowGasLimit).to.equal(true)
    // EIP-150 reserves gas in the outer frame; do not require the tx limit to be exhausted.
    expect(
      probe.args.gasAfterCall * 32n < probe.args.gasBeforeCall,
      `failed bank call undercharged: before=${probe.args.gasBeforeCall} after=${probe.args.gasAfterCall}`
    ).to.equal(true)
    expect(receipt.gasUsed > lowGasLimit * 9n / 10n, 'receipt must reflect the failed frame consumption').to.equal(true)
    expect(receipt.gasUsed < lowGasLimit, 'outer frame retains EIP-150 gas').to.equal(true)
  })

})

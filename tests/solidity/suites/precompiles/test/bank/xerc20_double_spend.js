import hre from 'hardhat'
import { expect } from 'chai'
import { LARGE_GAS_LIMIT } from '../common.js'

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

    await expect(
      poc.exploit(
        bankRecipient.address,
        directRecipient.address,
        amount,
        { gasLimit: LARGE_GAS_LIMIT }
      )
    ).to.revert(ethers)

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
})

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
    return await converter.bech32ToHex.staticCall(addr)
  }

  return addr
}

describe('xerc20 wasm precompile double spend PoC', function () {
  let counterWasm

  before(async function () {
    const counterAddr = process.env.COUNTER_WASM_ADDRESS
    if (!counterAddr) {
      this.skip()
      return
    }

    counterWasm = await resolveWasmAddress(counterAddr)
  })

  it('rejects double-crediting via wasm funds followed by direct transfer', async function () {
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
        counterWasm,
        ethers.toUtf8Bytes(INCREMENT_MSG),
        directRecipient.address,
        amount,
        { gasLimit: LARGE_GAS_LIMIT }
      )
    ).to.revert(ethers)

    const deployerBalance = await token.balanceOf(deployer.address)
    const pocBalance = await token.balanceOf(pocAddress)
    const wasmBalance = await token.balanceOf(counterWasm)
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
})

/**
 * WASM counter (counter_submsg) – normal execution only.
 * Verifies that the counter WASM contract behaves as intended when execute
 * completes without revert: increment → no_op submsg → reply → counter 3.
 *
 * Set COUNTER_WASM_ADDRESS (Bech32 or EVM hex) and run with get-contracts.
 */
import { expect } from 'chai';
import hre from 'hardhat';
import {
    WASM_PRECOMPILE_ADDRESS,
    BECH32_PRECOMPILE_ADDRESS,
    LARGE_GAS_LIMIT,
} from '../common.js';

const { ethers } = await hre.network.connect();

const GET_COUNTER_QUERY = '{"get_count":{}}';
const INCREMENT_MSG = '{"increment":{}}'; // full flow: increment → no_op submsg → reply (counter 3)

describe('WASM Counter (normal)', function () {
    let wasm;
    let bech32;
    let signer;
    let counterWasmAddress;

    before(async function () {
        const addr = process.env.COUNTER_WASM_ADDRESS;
        if (!addr || addr === '') {
            this.skip();
            return;
        }

        [signer] = await ethers.getSigners();
        wasm = await ethers.getContractAt('IWasm', WASM_PRECOMPILE_ADDRESS);
        bech32 = await ethers.getContractAt('Bech32I', BECH32_PRECOMPILE_ADDRESS);

        if (addr.startsWith('xpla')) {
            counterWasmAddress = await bech32.bech32ToHex.staticCall(addr);
        } else {
            counterWasmAddress = addr;
        }
    });

    async function getCounter() {
        const data = await wasm.smartContractState.staticCall(
            counterWasmAddress,
            ethers.toUtf8Bytes(GET_COUNTER_QUERY)
        );
        const decoded = ethers.toUtf8String(data);
        const obj = JSON.parse(decoded);
        return Number(obj.counter ?? obj.count ?? 0);
    }

    it('execute increment (full flow) then counter is 3', async function () {
        if (!counterWasmAddress) return this.skip();

        const funds = [];
        const tx = await wasm
            .connect(signer)
            .executeContract(
                signer.address,
                counterWasmAddress,
                ethers.toUtf8Bytes(INCREMENT_MSG),
                funds,
                { gasLimit: LARGE_GAS_LIMIT }
            );
        await tx.wait();

        const counter = await getCounter();
        expect(counter).to.equal(3);
    });
});

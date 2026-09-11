import { expect } from 'chai';
import hre from 'hardhat';
import {
    WASM_PRECOMPILE_ADDRESS,
    BECH32_PRECOMPILE_ADDRESS,
    LARGE_GAS_LIMIT,
} from '../common.js';
import {
    analyzeFailedTransaction,
    verifyTransactionRevert,
} from '../test_helper.js';

const { ethers } = await hre.network.connect();

const GET_COUNTER_QUERY = '{"get_count":{}}';
const INCREMENT_MSG = '{"increment":{}}'; // increment -> submsg no_op -> reply (counter 1->2->3)

// Measured success gas after StateDB CacheContext Commit:
//   Case 1 (EOA executeContract) ≈ 259678
//   Case 2 (WasmSnapshotRevertCaller) ≈ 286983
// Keep these just below that so WASM increment starts then OOGs and the snapshot rolls back.
const CASE1_OOG_GAS_LIMIT = 259_500
const CASE2_OOG_GAS_LIMIT = 286_500

describe('WASM Snapshot Revert', function () {
    let caller;
    let wasm;
    let bech32;
    let signer;
    let counterWasmAddress; // EVM-style address for precompile calls
    let analysis;

    before(async function () {
        const addr = process.env.REVERT_COUNTER_WASM_ADDRESS;
        if (!addr || addr === '') {
            this.skip();
            return;
        }

        [signer] = await ethers.getSigners();
        wasm = await ethers.getContractAt('IWasm', WASM_PRECOMPILE_ADDRESS);
        bech32 = await ethers.getContractAt('Bech32I', BECH32_PRECOMPILE_ADDRESS);

        // If Bech32, convert to EVM address for precompile
        if (addr.startsWith('xpla')) {
            counterWasmAddress = await bech32.bech32ToHex.staticCall(addr);
        } else {
            counterWasmAddress = addr;
        }

        const CallerFactory = await ethers.getContractFactory('WasmSnapshotRevertCaller');
        caller = await CallerFactory.deploy({ gasLimit: LARGE_GAS_LIMIT });
        await caller.waitForDeployment();

        analysis = null;
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

    it('Case 1: normal execution (sufficient gas) – counter should be 0', async function () {
        if (!counterWasmAddress) return this.skip();

        // Precompile requires caller == sender (ValidateSigner). So we call from EOA, not via WasmSnapshotRevertCaller.
        const funds = [];
        const tx = await wasm
            .connect(signer)
            .executeContract(
                signer.address,
                counterWasmAddress,
                ethers.toUtf8Bytes(INCREMENT_MSG),
                funds,
                { gasLimit: CASE1_OOG_GAS_LIMIT }
            );
        try {
            await tx.wait();
            expect.fail('Transaction should have failed with revert');
        } catch (error) {
            analysis = await analyzeFailedTransaction(error.receipt.hash, ethers);
        }

        verifyTransactionRevert(analysis, "out of gas");

        const counter = await getCounter();
        expect(counter).to.equal(0);
    });

    it('Case 2: via WasmSnapshotRevertCaller – on revert local counter rolls back to 0', async function () {
        if (!counterWasmAddress) return this.skip();

        // Sanity: local counter starts at 0
        const beforeLocal = await caller.counter();
        expect(beforeLocal).to.equal(0n);

        // Call via contract. The contract sets local counter=1, then calls WASM precompile with sender=address(this).
        // With limited gas, we expect revert (status=0), and local counter must rollback to 0.
        const tx = await caller.callExecuteWithLocalCounter(
            counterWasmAddress,
            ethers.toUtf8Bytes(INCREMENT_MSG),
            { gasLimit: CASE2_OOG_GAS_LIMIT }
        );

        // tx.wait() may throw (revert) or timeout; verify via JSON-RPC receipt.
        try {
            await tx.wait();
            expect.fail('Transaction should have failed with revert');
        } catch (error) {
            analysis = await analyzeFailedTransaction(error.receipt.hash, ethers);
        }

        // Inner precompile OOG is wrapped by the 63/64 call rule, so the outer
        // tx reverts with "execution reverted" rather than a top-level OOG.
        verifyTransactionRevert(analysis, "execution reverted");

        // Key point: on revert, local counter must rollback to 0 (not 1).
        const afterLocal = await caller.counter();
        expect(afterLocal, 'local counter should be 0 after revert').to.equal(0n);

        const counter = await getCounter();
        expect(counter).to.equal(0);
    });

    it('Case 3: via WasmSnapshotRevertCaller – on success local counter stays 1', async function () {
        if (!counterWasmAddress) return this.skip();

        // Sanity: local counter is 0 (from previous test or fresh state)
        const beforeLocal = await caller.counter();
        expect(beforeLocal).to.equal(0n);

        // Call with sufficient gas so WASM execute succeeds. Contract sets counter=1 before the call.
        const tx = await caller.callExecuteWithLocalCounter(
            counterWasmAddress,
            ethers.toUtf8Bytes(INCREMENT_MSG),
            { gasLimit: LARGE_GAS_LIMIT }
        );
        await tx.wait();

        // Key point: on success, local counter must remain 1 (no rollback).
        const afterLocal = await caller.counter();
        expect(afterLocal, 'local counter should be 1 after success').to.equal(1n);

        const counter = await getCounter();
        expect(counter, 'WASM counter should have incremented').to.be.greaterThan(0);
    });
});

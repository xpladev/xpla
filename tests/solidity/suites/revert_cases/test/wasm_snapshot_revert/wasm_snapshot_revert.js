const { expect } = require('chai');
const hre = require('hardhat');
const {
    WASM_PRECOMPILE_ADDRESS,
    BECH32_PRECOMPILE_ADDRESS,
    LARGE_GAS_LIMIT,
} = require('../common');

const {
    analyzeFailedTransaction,
    verifyTransactionRevert,
} = require('../test_helper')

const GET_COUNTER_QUERY = '{"get_count":{}}';
const INCREMENT_MSG = '{"increment":{}}'; // increment -> submsg no_op -> reply (counter 1->2->3)

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

        [signer] = await hre.ethers.getSigners();
        wasm = await hre.ethers.getContractAt('IWasm', WASM_PRECOMPILE_ADDRESS);
        bech32 = await hre.ethers.getContractAt('Bech32I', BECH32_PRECOMPILE_ADDRESS);

        // If Bech32, convert to EVM address for precompile
        if (addr.startsWith('xpla')) {
            counterWasmAddress = await bech32.bech32ToHex.staticCall(addr);
        } else {
            counterWasmAddress = addr;
        }

        const CallerFactory = await hre.ethers.getContractFactory('WasmSnapshotRevertCaller');
        caller = await CallerFactory.deploy({ gasLimit: LARGE_GAS_LIMIT });
        await caller.waitForDeployment();

        analysis = null;
    });

    async function getCounter() {
        const data = await wasm.smartContractState.staticCall(
            counterWasmAddress,
            hre.ethers.toUtf8Bytes(GET_COUNTER_QUERY)
        );
        const decoded = hre.ethers.toUtf8String(data);
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
                hre.ethers.toUtf8Bytes(INCREMENT_MSG),
                funds,
                { gasLimit: 259_670 }
            );
        try {
            await tx.wait();
            expect.fail('Transaction should have failed with revert');
        } catch (error) {
            analysis = await analyzeFailedTransaction(error.receipt.hash);
        }

        verifyTransactionRevert(analysis, "out of gas");

        const counter = await getCounter();
        expect(counter).to.equal(0);
    });

    it('Case 2: via WasmSnapshotRevertCaller local counter rolls back on revert', async function () {
        if (!counterWasmAddress) return this.skip();

        // Sanity: local counter starts at 0
        const beforeLocal = await caller.counter();
        expect(beforeLocal).to.equal(0n);

        // Call via contract. The contract sets local counter=1, then calls WASM precompile with sender=address(this).
        // With limited gas, we expect revert (status=0), and local counter must rollback to 0.
        const tx = await caller.callExecuteWithLocalCounter(
            counterWasmAddress,
            hre.ethers.toUtf8Bytes(INCREMENT_MSG),
            { gasLimit: 286980 }
        );

        // tx.wait() may throw (revert) or timeout; verify via JSON-RPC receipt.
        try {
            await tx.wait();
            expect.fail('Transaction should have failed with revert');
        } catch (error) {
            analysis = await analyzeFailedTransaction(error.receipt.hash);
        }

        verifyTransactionRevert(analysis, "out of gas");

        const afterLocal = await caller.counter();
        expect(afterLocal, 'local counter should rollback to 0').to.equal(0n);

        const counter = await getCounter();
        expect(counter).to.equal(0);
    });
});

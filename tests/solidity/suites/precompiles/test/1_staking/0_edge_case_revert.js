const { expect } = require('chai');
const hre = require('hardhat');
const {
    STAKING_PRECOMPILE_ADDRESS,
    LARGE_GAS_LIMIT,
    waitWithTimeout, RETRY_DELAY_FUNC
} = require('../common');

describe('Staking – edge case revert test', function () {
    const GAS_LIMIT = LARGE_GAS_LIMIT;

    let stakingReverter, staking, signer;
    let validatorAddress;

    before(async function () {
        [signer] = await hre.ethers.getSigners();
        
        // Get staking precompile interface
        staking = await hre.ethers.getContractAt('StakingI', STAKING_PRECOMPILE_ADDRESS);
        
        // Get actual nonce from chain (may not be 0 if other transactions were sent)
        const actualNonce = await hre.ethers.provider.getTransactionCount(signer.address);
        
        // Calculate contract address using actual nonce
        // Expected address: 0x3D641a2791533B4A0000345eA8d509d01E1ec301 (Bech32: xpla184jp5fu32va55qqqx30234gf6q0pascpg7nq6k)
        // This address should be funded with axpla in genesis (see local_node.sh)
        const predictedAddress = hre.ethers.getCreateAddress({
            from: signer.address,
            nonce: actualNonce
        });
        console.log('Predicted contract address (using actual nonce):', predictedAddress);
        
        // Deploy StakingReverter contract (axpla is funded in genesis)
        const StakingReverterFactory = await hre.ethers.getContractFactory('StakingReverter');
        stakingReverter = await StakingReverterFactory.deploy({
            gasLimit: GAS_LIMIT
        });
        await waitWithTimeout(stakingReverter.deploymentTransaction(), 20000, RETRY_DELAY_FUNC)

        const deployedAddress = await stakingReverter.getAddress();
        console.log('StakingReverter deployed at:', deployedAddress);
        
        // Verify deployed address matches predicted address
        expect(deployedAddress.toLowerCase()).to.equal(predictedAddress.toLowerCase(), 'Deployed address should match predicted address (based on actual nonce)');

        validatorAddress = 'xplavaloper10jmp6sgh4cc6zt3e8gw05wavvejgr5pwlgtsgz';
        
        console.log('Using validator address:', validatorAddress);
    });

    describe('Edge case: callPrecompileBeforeAndAfterRevert with numTimes=1', function () {
        it('should execute exactly two delegate operations', async function () {
            const contractAddress = await stakingReverter.getAddress();
            
            // Get initial delegation before test
            let initialShares, initialBalance;
            [initialShares, initialBalance] = await staking.delegation(contractAddress, validatorAddress);
            console.log('Initial delegation shares:', initialShares.toString());
            console.log('Initial delegation balance:', initialBalance.amount.toString());
            console.log('Initial delegation balance denom:', initialBalance.denom);

            // Call the edge case method with numTimes = 1
            console.log('Calling callPrecompileBeforeAndAfterRevert with numTimes=1');
            
            const tx = await stakingReverter.callPrecompileBeforeAndAfterRevert(1, validatorAddress, {
                gasLimit: GAS_LIMIT
            });
            await waitWithTimeout(tx, 20000, RETRY_DELAY_FUNC);
            const receipt = await tx.wait();
            
            console.log('Transaction hash:', receipt.hash);
            console.log('Gas used:', receipt.gasUsed.toString());
            
            // Verify transaction succeeded
            expect(receipt.status).to.equal(1);
            expect(receipt.gasUsed).to.be.greaterThan(0);
            
            // Check final delegation state
            const [finalShares, finalBalance] = await staking.delegation(contractAddress, validatorAddress);
            console.log('Final delegation shares:', finalShares.toString());
            console.log('Final delegation balance:', finalBalance.amount.toString());
            console.log('Final delegation balance denom:', finalBalance.denom);
            
            // Calculate expected final amount (initial + 2 delegate operations of 10 wei each)
            const expectedFinalAmount = BigInt(initialBalance.amount) + (2n * 10n);
            
            console.log('Expected final amount:', expectedFinalAmount.toString());
            console.log('Actual final amount:', finalBalance.amount.toString());
            
            // Verify exactly two delegate operations were executed
            // According to the pattern: one before the loop + one after the loop = 2 total
            expect(finalBalance.amount).to.equal(expectedFinalAmount);
            
            // Verify shares increased appropriately
            expect(finalShares).to.be.greaterThan(initialShares);
            
            // Verify that delegation uses axpla (bond denom)
            expect(finalBalance.denom).to.equal('axpla', 'Delegation should use axpla as bond denom');
            
            console.log('✓ Edge case test passed: exactly two delegate operations executed');
        });
    });
});
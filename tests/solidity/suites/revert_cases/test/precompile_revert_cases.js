import { expect } from 'chai';
import hre from 'hardhat';
import { LARGE_GAS_LIMIT, LOW_GAS_LIMIT } from './common.js';
import {
    decodeRevertReason,
    analyzeFailedTransaction,
    verifyTransactionRevert,
    verifyOutOfGasError
} from './test_helper.js';

const { ethers } = await hre.network.connect();

describe('Precompile Revert Cases E2E Tests', function () {
    let revertTestContract, precompileWrapper;
    let signer;
    let validValidatorAddress, invalidValidatorAddress;
    let analysis, decodedReason;

    before(async function () {
        [signer] = await ethers.getSigners();
        
        // Deploy RevertTestContract
        const RevertTestContractFactory = await ethers.getContractFactory('RevertTestContract');
        revertTestContract = await RevertTestContractFactory.deploy({
            value: ethers.parseEther('1.0'), // Fund with 1 ETH
            gasLimit: LARGE_GAS_LIMIT
        });
        await revertTestContract.waitForDeployment();
        
        // Deploy PrecompileWrapper
        const PrecompileWrapperFactory = await ethers.getContractFactory('PrecompileWrapper');
        precompileWrapper = await PrecompileWrapperFactory.deploy({
            value: ethers.parseEther('1.0'), // Fund with 1 ETH
            gasLimit: LARGE_GAS_LIMIT
        });
        await precompileWrapper.waitForDeployment();
        
        // Use a known validator for valid cases and invalid one for error cases
        validValidatorAddress = 'xplavaloper10jmp6sgh4cc6zt3e8gw05wavvejgr5pwlgtsgz';
        invalidValidatorAddress = 'xplavaloper10jmp6sgh4cc6zt3e8gw05wavvejgr5pinvalid';
        
        console.log('RevertTestContract deployed at:', await revertTestContract.getAddress());
        console.log('PrecompileWrapper deployed at:', await precompileWrapper.getAddress());

        analysis = null;
        decodedReason = null;
    });

    describe('Direct Precompile Call Reverts', function () {
        it('should handle direct staking precompile revert', async function () {
            try {
                const tx = await revertTestContract.directStakingRevert(invalidValidatorAddress, { gasLimit: LARGE_GAS_LIMIT });
                await tx.wait();
                expect.fail('Transaction should have reverted');
            } catch (error) {
                analysis = await analyzeFailedTransaction(error.receipt.hash, ethers)
            }
            verifyTransactionRevert(analysis, "invalid validator address")
        });

        it('should handle direct distribution precompile revert', async function () {            
            try {
                const tx = await revertTestContract.directDistributionRevert(invalidValidatorAddress, { gasLimit: LARGE_GAS_LIMIT });
                await tx.wait();
                expect.fail('Transaction should have reverted');
            } catch (error) {
                analysis = await analyzeFailedTransaction(error.receipt.hash, ethers)
            }
            verifyTransactionRevert(analysis, "invalid validator address")
        });

        it('should handle direct bank precompile revert', async function () {
            // directBankRevert is a view function, so it should revert immediately
            try {
                await revertTestContract.directBankRevert();
                expect.fail('Call should have reverted');
            } catch (error) {
                decodedReason = decodeRevertReason(error.data)
            }
            expect(decodedReason).to.include("intended revert")
        });

        it('should capture precompile revert reason through transaction receipt', async function () {
            try {
                const tx = await revertTestContract.directStakingRevert(invalidValidatorAddress, { gasLimit: LARGE_GAS_LIMIT });
                await tx.wait();
                expect.fail('Transaction should have reverted');
            } catch (error) {
                analysis = await analyzeFailedTransaction(error.receipt.hash, ethers)
            }
            verifyTransactionRevert(analysis, "invalid validator address")
        });
    });

    describe('Precompile Call Via Contract Reverts', function () {
        it('should handle precompile call via contract revert', async function () {            
            try {
                const tx = await revertTestContract.precompileViaContractRevert(invalidValidatorAddress, { gasLimit: LARGE_GAS_LIMIT });
                await tx.wait();
                expect.fail('Transaction should have reverted');
            } catch (error) {
                analysis = await analyzeFailedTransaction(error.receipt.hash, ethers)
            }
            verifyTransactionRevert(analysis, "invalid validator address")
        });

        it('should handle wrapper contract precompile revert', async function () {
            try {
                const tx = await precompileWrapper.wrappedStakingCall(invalidValidatorAddress, 1, { gasLimit: LARGE_GAS_LIMIT });
                await tx.wait();
                expect.fail('Transaction should have reverted');
            } catch (error) {
                analysis = await analyzeFailedTransaction(error.receipt.hash, ethers)
            }
            verifyTransactionRevert(analysis, "invalid validator address")
        });

        it('should capture wrapper revert reason via transaction receipt', async function () {
            try {
                const tx = await precompileWrapper.wrappedDistributionCall(invalidValidatorAddress, { gasLimit: LARGE_GAS_LIMIT });
                await tx.wait();
                expect.fail('Transaction should have reverted');
            } catch (error) {
                analysis = await analyzeFailedTransaction(error.receipt.hash, ethers)
            }
            verifyTransactionRevert(analysis, "invalid validator address")
        });
    });

    describe('Precompile OutOfGas Error Cases', function () {
        it('should handle direct precompile OutOfGas', async function () {
            // Use a very low gas limit to trigger OutOfGas on precompile calls            
            try {
                const tx = await revertTestContract.directStakingOutOfGas(validValidatorAddress, { gasLimit: LOW_GAS_LIMIT });
                await tx.wait();
                expect.fail('Transaction should have failed with OutOfGas');
            } catch (error) {
                analysis = await analyzeFailedTransaction(error.receipt.hash, ethers)
            }
            verifyOutOfGasError(analysis)
        });

        it('should handle precompile via contract OutOfGas', async function () {            
            try {
                const tx = await revertTestContract.precompileViaContractOutOfGas(validValidatorAddress, { gasLimit: LOW_GAS_LIMIT });
                await tx.wait();
                expect.fail('Transaction should have failed with OutOfGas');
            } catch (error) {
                analysis = await analyzeFailedTransaction(error.receipt.hash, ethers)
            }
            verifyOutOfGasError(analysis)
        });

        it('should handle wrapper precompile OutOfGas', async function () {
            try {
                const tx = await precompileWrapper.wrappedOutOfGasCall(validValidatorAddress, { gasLimit: LOW_GAS_LIMIT });
                await tx.wait();
                expect.fail('Transaction should have failed with OutOfGas');
            } catch (error) {
                analysis = await analyzeFailedTransaction(error.receipt.hash, ethers)
            }
            verifyOutOfGasError(analysis)
        });

        it('should analyze precompile OutOfGas error through transaction receipt', async function () {
            try {
                const tx = await revertTestContract.directStakingOutOfGas(validValidatorAddress, { gasLimit: LOW_GAS_LIMIT });
                await tx.wait();
                expect.fail('Transaction should have failed with OutOfGas');
            } catch (error) {
                analysis = await analyzeFailedTransaction(error.receipt.hash, ethers);
            }
            verifyOutOfGasError(analysis)
        });
    });

    describe('Comprehensive Precompile Error Analysis', function () {
        it('should properly decode various precompile error types from transaction receipts', async function () {
            const testCases = [
                {
                    name: 'Staking Precompile Revert',
                    call: () => revertTestContract.directStakingRevert(invalidValidatorAddress, { gasLimit: LARGE_GAS_LIMIT }),
                    expectedReason: "invalid validator address"
                },
                {
                    name: 'Distribution Precompile Revert',
                    call: () => revertTestContract.directDistributionRevert(invalidValidatorAddress, { gasLimit: LARGE_GAS_LIMIT }),
                    expectedReason: "invalid validator address"
                }
            ];

            for (const testCase of testCases) {
                try {
                    const tx = await testCase.call();
                    await tx.wait()
                    expect.fail(`${testCase.name} should have reverted`);
                } catch (error) {
                    analysis = await analyzeFailedTransaction(error.receipt.hash, ethers);
                }
                verifyTransactionRevert(analysis, testCase.expectedReason)
            }
        });

        it('should verify precompile error data is properly hex-encoded in receipts', async function () {
            try {
                const tx = await revertTestContract.directStakingRevert(invalidValidatorAddress, { gasLimit: LARGE_GAS_LIMIT });
                await tx.wait();
                expect.fail('Transaction should have reverted');
            } catch (error) {
                if (error.receipt) {
                    // Simulate the call to get error data
                    try {
                        const contractAddress = await revertTestContract.getAddress();
                        await ethers.provider.call({
                            to: contractAddress,
                            data: revertTestContract.interface.encodeFunctionData('directStakingRevert', [invalidValidatorAddress]),
                            gasLimit: LARGE_GAS_LIMIT
                        });
                    } catch (callError) {
                        expect(callError.data).to.match(/^0x/); // Should be hex-encoded
                        console.log('Precompile error data (hex):', callError.data);
                        
                        const decoded = decodeRevertReason(callError.data);
                        expect(decoded).to.include("invalid validator address");
                        console.log('Decoded precompile reason:', decoded);
                    }
                }
            }
        });
    });
});
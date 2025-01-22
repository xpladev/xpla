// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

address constant DISTRIBUTION_PRECOMPILE_ADDRESS = 0x1000000000000000000000000000000000000003;

IDistribution constant DISTRIBUTION_CONTRACT = IDistribution(
    DISTRIBUTION_PRECOMPILE_ADDRESS
);

interface IDistribution {
    // Transactions
    function withdrawDelegatorReward(
        address delegatorAddress,
        address validatorAddress
    ) external returns (uint256 amount);
}

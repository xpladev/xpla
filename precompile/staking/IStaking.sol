// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

address constant STAKING_PRECOMPILE_ADDRESS = 0x1000000000000000000000000000000000000002;

IStaking constant STAKING_CONTRACT = IStaking(
    STAKING_PRECOMPILE_ADDRESS
);

interface IStaking {
    // Transactions
    function delegate(
        address delegatorAddress,
        address validatorAddress,
        string memory denom,
        uint256 amount
    ) external returns (bool success);

    function beginRedelegate(
        address delegatorAddress,
        address validatorSrcAddress,
        address validatorDstAddress,
        string memory denom,
        uint256 amount
    ) external returns (uint256 completionTime);

    function undelegate(
        address delegatorAddress,
        address validatorAddress,
        string memory denom,
        uint256 amount
    ) external returns (uint256 completionTime);
}

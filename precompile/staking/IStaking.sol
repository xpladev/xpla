// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

address constant STAKING_PRECOMPILE_ADDRESS = 0x1000000000000000000000000000000000000002;

IStaking constant STAKING_CONTRACT = IStaking(
    STAKING_PRECOMPILE_ADDRESS
);

struct Coin {
    string denom;
    uint256 amount;
}

interface IStaking {
    // Transactions
    function delegate(
        address delegatorAddress,
        address validatorAddress,
        Coin calldata coin
    ) external returns (bool success);

    function beginRedelegate(
        address delegatorAddress,
        address validatorSrcAddress,
        address validatorDstAddress,
        Coin calldata coin
    ) external returns (uint256 completionTime);

    function undelegate(
        address delegatorAddress,
        address validatorAddress,
        Coin calldata coin
    ) external returns (uint256 completionTime);
}

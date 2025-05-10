// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {Coin} from "../util/Structs.sol";

address constant STAKING_PRECOMPILE_ADDRESS = 0x1000000000000000000000000000000000000002;

IStaking constant STAKING_CONTRACT = IStaking(
    STAKING_PRECOMPILE_ADDRESS
);

interface IStaking {
    // Transactions
    function delegate(
        address delegatorAddress,
        address validatorAddress,
        Coin calldata amount
    ) external returns (bool success);

    function beginRedelegate(
        address delegatorAddress,
        address validatorSrcAddress,
        address validatorDstAddress,
        Coin calldata amount
    ) external returns (uint256 completionTime);

    function undelegate(
        address delegatorAddress,
        address validatorAddress,
        Coin calldata amount
    ) external returns (uint256 completionTime);
}

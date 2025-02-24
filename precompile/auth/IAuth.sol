// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

address constant AUTH_PRECOMPILE_ADDRESS = 0x1000000000000000000000000000000000000005;

IAuth constant AUTH_CONTRACT = IAuth(
    AUTH_PRECOMPILE_ADDRESS
);

interface IAuth {
    function associatedAddress(address evmAddress) external view returns (bytes calldata addr);
}

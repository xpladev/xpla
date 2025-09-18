// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

address constant AUTH_PRECOMPILE_ADDRESS = 0x1000000000000000000000000000000000000005;

IAuth constant AUTH_CONTRACT = IAuth(
    AUTH_PRECOMPILE_ADDRESS
);

interface IAuth {
    // function accountAddressByID(uint accountId) external view returns (string calldata stringAddress);
    // function accounts(address evmAddress) external view returns (string[] calldata stringAddress);
    function account(address evmAddress) external view returns (string calldata stringAddress);
    // function params() external view returns (...);
    // function moduleAccounts() external view returns (string[] calldata stringAddresses);
    function moduleAccountByName(string calldata name) external view returns (string calldata stringAddress);
    function bech32Prefix() external view returns (string calldata prefix);
    function addressBytesToString(address evmAddress) external view returns (string calldata stringAddress);
    function addressStringToBytes(string calldata stringAddress) external view returns (address byteAddress);
    // function accountInfo(address evmAddress) external view returns (...);
}

// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

address constant WASM_PRECOMPILE_ADDRESS = 0x1000000000000000000000000000000000000004;

IWasm constant WASM_CONTRACT = IWasm(
    WASM_PRECOMPILE_ADDRESS
);

struct Coin {
    string denom;
    uint256 amount;
}

interface IWasm {
    // Transactions
    function instantiateContract(
        address sender,
        address admin,
        uint256 codeId,
        string calldata label,
        bytes calldata msg,
        Coin[] memory coins
    ) external returns (address contractAddress, bytes calldata data);
    function instantiateContract2(
        address sender,
        address admin,
        uint256 codeId,
        string calldata label,
        bytes calldata msg,
        Coin[] memory coins,
        bytes calldata salt,
        bool fixMsg
    ) external returns (address contractAddress, bytes calldata data);
    function executeContract(
        address sender,
        address contractAddress,
        bytes calldata msg,
        Coin[] memory coins
    ) external returns (bytes calldata data);
    function migrateContract(
        address sender,
        address contractAddress,
        uint256 codeId,
        bytes calldata msg
    ) external returns (bytes calldata data);
    
    // Queries
    function smartContractState(
        address contractAddress,
        bytes calldata queryData
    ) external view returns (bytes calldata data);
}

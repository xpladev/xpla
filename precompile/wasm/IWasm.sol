// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {Coin} from "cosmos-evm-contracts/precompiles/common/Types.sol";

address constant WASM_PRECOMPILE_ADDRESS = 0x1000000000000000000000000000000000000004;

IWasm constant WASM_CONTRACT = IWasm(
    WASM_PRECOMPILE_ADDRESS
);

interface IWasm {
    // Events
    /**
     * @dev InstantiateContract defines an event emitted when a wasm contract is successfully
     * instantiated via instantiateContract or instantiateContract2
     * @param sender the address of the sender
     * @param contractAddress the address of the instantiated contract
     * @param codeId the id of contract
     * @param admin the address of the contract admin
     * @param label the optional metadata to be stored with a contract instance
     * @param msg the message to be passed to the contract on instantiation
     * @param funds the coins that are transferred to the contract on instantiation
     * @param data the bytes to returned from the contract
     */
    event InstantiateContract(
        address indexed sender,
        address indexed contractAddress,
        uint64 indexed codeId,
        address admin,
        string label,
        bytes msg,
        Coin[] funds,
        bytes data
    );

    /**
     * @dev ExecuteContract defines an event emitted when a wasm contract is successfully
     * executed via executeContract
     * @param sender the address of the sender
     * @param contractAddress the address of executed contract
     * @param msg the message to be passed to the contract
     * @param funds the coins that are transferred to the contract on execution
     * @param data the bytes to returned from the contract
     */
    event ExecuteContract(
        address indexed sender,
        address indexed contractAddress,
        bytes msg,
        Coin[] funds,
        bytes data
    );

    /**
     * @dev MigrateContract defines an event emitted when a wasm contract is successfully
     * migrated via migrateContract
     * @param sender the address of the sender
     * @param contractAddress the address of migrated contract
     * @param codeId changed code id
     * @param msg the message to be passed to the contract on migration
     * @param data the bytes to returned from the contract
     */
    event MigrateContract(
        address indexed sender,
        address indexed contractAddress,
        uint64 indexed codeId,
        bytes msg,
        bytes data
    );

    // Transactions
    function instantiateContract(
        address sender,
        address admin,
        uint64 codeId,
        string calldata label,
        bytes calldata msg,
        Coin[] memory funds
    ) external returns (address contractAddress, bytes calldata data);
    function instantiateContract2(
        address sender,
        address admin,
        uint64 codeId,
        string calldata label,
        bytes calldata msg,
        Coin[] memory funds,
        bytes calldata salt,
        bool fixMsg
    ) external returns (address contractAddress, bytes calldata data);
    function executeContract(
        address sender,
        address contractAddress,
        bytes calldata msg,
        Coin[] memory funds
    ) external returns (bytes calldata data);
    function migrateContract(
        address sender,
        address contractAddress,
        uint64 codeId,
        bytes calldata msg
    ) external returns (bytes calldata data);
    
    // Queries
    /**
     * @dev Returns the native Cosmos bank balance of a Wasm contract, resolving
     * registered 20-byte aliases to the full contract address.
     * Reverts for missing contracts, invalid denoms, or xerc20/xcw20 denoms.
     * @param contractAddress the Wasm contract address or registered alias
     * @param denom a Cosmos bank denom, including axpla or an IBC denom
     * @return amount the balance in the denom's smallest unit, or zero if unheld
     */
    function balance(
        address contractAddress,
        string calldata denom
    ) external view returns (uint256 amount);
    function smartContractState(
        address contractAddress,
        bytes calldata queryData
    ) external view returns (bytes calldata data);
}

// SPDX-License-Identifier: MIT
pragma solidity ^0.8.18;

import {Coin} from "cosmos-evm-contracts/precompiles/common/Types.sol";
import {IWasm, WASM_PRECOMPILE_ADDRESS} from "../xpla/wasm/IWasm.sol";

// Normal query/funds fixture: every balance read executes an EVM STATICCALL.
contract WasmBalanceReader {
    IWasm private constant WASM = IWasm(WASM_PRECOMPILE_ADDRESS);

    function instantiate(uint64 codeId) external returns (address contractAddress) {
        Coin[] memory funds = new Coin[](0);
        (contractAddress,) = WASM.instantiateContract(
            address(this), address(this), codeId, "balance reader", bytes("{}"), funds
        );
    }

    function read(address contractAddress, string memory denom)
        public view returns (uint256 amount, uint256 suffixBalance)
    {
        (bool success, bytes memory result) = WASM_PRECOMPILE_ADDRESS.staticcall(
            abi.encodeCall(IWasm.balance, (contractAddress, denom))
        );
        require(success, "balance STATICCALL failed");
        amount = abi.decode(result, (uint256));
        suffixBalance = contractAddress.balance;
    }

    function fundAndRead(address contractAddress, uint256 amount, bytes calldata executeMsg, bool revertAfter)
        external returns (uint256 beforeAmount, uint256 afterAmount, uint256 suffixBalance)
    {
        (beforeAmount,) = read(contractAddress, "axpla");
        Coin[] memory funds = new Coin[](1);
        funds[0] = Coin("axpla", amount);
        WASM.executeContract(
            address(this), contractAddress, executeMsg, funds
        );
        (afterAmount, suffixBalance) = read(contractAddress, "axpla");
        require(afterAmount == beforeAmount + amount, "latest balance missing");
        require(!revertAfter, "requested outer revert");
    }
}

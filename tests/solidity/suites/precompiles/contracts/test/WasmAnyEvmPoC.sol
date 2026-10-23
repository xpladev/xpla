// SPDX-License-Identifier: MIT
pragma solidity ^0.8.18;

import {Coin} from "cosmos-evm-contracts/precompiles/common/Types.sol";
import {IWasm, WASM_PRECOMPILE_ADDRESS} from "../xpla/wasm/IWasm.sol";

contract InnerEvmEmitter {
    uint256 public counter;

    event InnerEvmLog(
        bytes32 indexed marker,
        address indexed caller,
        uint256 value,
        uint256 gasLeft
    );

    function emitInner(bytes32 marker, uint256 value) external {
        counter += value;
        emit InnerEvmLog(marker, msg.sender, value, gasleft());
    }
}

contract WasmAnyEvmPoC {
    IWasm private constant WASM = IWasm(WASM_PRECOMPILE_ADDRESS);

    error ForcedOuterRevertAfterAny(bytes32 marker);

    bytes32 public constant ANY_EXECUTION_COMPLETED_MARKER =
        keccak256("any execution completed");

    event AnyDispatchResult(bool success, bytes returnData);

    function executeAny(
        address wasmContract,
        bytes calldata wasmMsg
    ) external returns (bytes memory) {
        Coin[] memory funds = new Coin[](0);
        return WASM.executeContract(
            address(this),
            wasmContract,
            wasmMsg,
            funds
        );
    }

    function tryExecuteAny(
        address wasmContract,
        bytes calldata wasmMsg
    ) external returns (bool success, bytes memory returnData) {
        Coin[] memory funds = new Coin[](0);
        (success, returnData) = WASM_PRECOMPILE_ADDRESS.call(
            abi.encodeCall(
                IWasm.executeContract,
                (address(this), wasmContract, wasmMsg, funds)
            )
        );
        emit AnyDispatchResult(success, returnData);
    }

    function executeAnyThenRevert(
        address wasmContract,
        bytes calldata wasmMsg
    ) external returns (bytes memory) {
        require(msg.sender == address(this), "only self");

        Coin[] memory funds = new Coin[](0);
        WASM.executeContract(address(this), wasmContract, wasmMsg, funds);
        revert ForcedOuterRevertAfterAny(ANY_EXECUTION_COMPLETED_MARKER);
    }

    function tryExecuteAnyThenRevert(
        address wasmContract,
        bytes calldata wasmMsg
    ) external returns (bool success, bytes memory returnData) {
        (success, returnData) = address(this).call(
            abi.encodeCall(
                this.executeAnyThenRevert,
                (wasmContract, wasmMsg)
            )
        );
        emit AnyDispatchResult(success, returnData);
    }
}

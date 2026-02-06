// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

import "./xpla/wasm/IWasm.sol";
import "./xpla/util/Types.sol" as xplaTypes;

/// @title WasmSnapshotRevertCaller
/// @dev Calls WASM precompile executeContract for snapshot/revert tests.
/// Gas is controlled by the transaction gas limit when calling callExecute.
contract WasmSnapshotRevertCaller {
    /// @dev EVM-side state used to validate rollback behavior.
    /// Starts at 0. Set to 1 before calling the WASM precompile in Case 2 tests.
    uint256 public counter;

    /// @dev Case 2 helper: Mutate local EVM state, then call WASM precompile.
    /// IMPORTANT: WASM precompile validates `args[0] == caller` (ValidateSigner),
    /// so we must pass `address(this)` as the sender when calling via a contract.
    /// If the WASM execution reverts (e.g. OOG), this function's state update
    /// must rollback as well (counter remains 0).
    function callExecuteWithLocalCounter(
        address contractAddress,
        bytes calldata executeMsg
    ) external returns (bytes memory) {
        counter = 1;
        xplaTypes.Coin[] memory funds = new xplaTypes.Coin[](0);
        return IWasm(WASM_PRECOMPILE_ADDRESS).executeContract(
            address(this),
            contractAddress,
            executeMsg,
            funds
        );
    }
}

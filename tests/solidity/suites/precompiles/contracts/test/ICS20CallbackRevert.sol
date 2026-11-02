// SPDX-License-Identifier: MIT
pragma solidity ^0.8.18;

/// Exercises an ICS20 precompile call in a child frame that is deliberately
/// reverted while the outer frame succeeds.
contract ICS20CallbackRevert {
    address private constant ICS20 = 0x0000000000000000000000000000000000000802;
    bytes32 public constant ICS20_COMPLETED_MARKER =
        keccak256("ics20 precompile completed");

    error ChildCallUnexpectedlySucceeded();
    error ICS20CallFailed(bytes reason);
    error DeliberateChildRevert(bytes32 marker, uint64 packetSequence);

    event ChildICS20Succeeded(
        bytes32 indexed marker,
        uint64 packetSequence
    );
    event ControlICS20Succeeded(
        bytes32 indexed marker,
        uint64 packetSequence
    );

    /// Retains msg.value in the outer frame and invokes the supplied ICS20
    /// calldata through this contract's fallback in a value-free child frame.
    function attack(bytes calldata precompileCallData) external payable {
        (bool success, bytes memory result) = address(this).call(precompileCallData);
        if (success) revert ChildCallUnexpectedlySucceeded();

        bytes4 selector;
        bytes32 marker;
        uint256 packetSequence;
        if (result.length == 68) {
            assembly {
                selector := mload(add(result, 32))
                marker := mload(add(result, 36))
                packetSequence := mload(add(result, 68))
            }
        }
        if (
            selector != DeliberateChildRevert.selector ||
            marker != ICS20_COMPLETED_MARKER ||
            packetSequence == 0 ||
            packetSequence > type(uint64).max
        ) revert ICS20CallFailed(result);

        emit ChildICS20Succeeded(marker, uint64(packetSequence));
    }

    /// Commits the same native precompile path without a child-frame revert.
    /// The multichain test uses this as a control proving that a disabled or
    /// short-circuited precompile cannot make the rollback invariant pass.
    function controlTransfer(
        bytes calldata precompileCallData
    ) external returns (uint64 packetSequence) {
        packetSequence = _callICS20(precompileCallData);
        emit ControlICS20Succeeded(
            ICS20_COMPLETED_MARKER,
            packetSequence
        );
    }

    fallback() external {
        uint64 packetSequence = _callICS20(msg.data);
        revert DeliberateChildRevert(
            ICS20_COMPLETED_MARKER,
            packetSequence
        );
    }

    function _callICS20(
        bytes memory precompileCallData
    ) private returns (uint64 packetSequence) {
        (bool success, bytes memory result) = ICS20.call(precompileCallData);
        if (!success) {
            assembly {
                revert(add(result, 32), mload(result))
            }
        }
        if (result.length != 32) revert ICS20CallFailed(result);
        packetSequence = abi.decode(result, (uint64));
        if (packetSequence == 0) revert ICS20CallFailed(result);
    }
}

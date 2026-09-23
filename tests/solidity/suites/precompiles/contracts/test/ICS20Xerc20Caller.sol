// SPDX-License-Identifier: MIT
pragma solidity ^0.8.18;

import "@openzeppelin/contracts/token/ERC20/IERC20.sol";

interface IICS20Xerc20 {
    struct Height { uint64 revisionNumber; uint64 revisionHeight; }

    function transfer(
        string calldata port, string calldata channel, string calldata denom,
        uint256 amount, address sender, string calldata receiver,
        Height calldata timeoutHeight, uint64 timeoutTimestamp, string calldata memo
    ) external returns (uint64);
}

/// Exercises the registered ICS20 entry point without test-only StateDB injection.
contract ICS20Xerc20Caller {
    IICS20Xerc20 private constant ICS20 = IICS20Xerc20(0x0000000000000000000000000000000000000802);
    IERC20 public immutable token;
    string public denom;

    error ForcedOuterRevert();
    event SendResult(uint64 sequence, uint256 balanceAfter, bool reuseSuccess, bytes reuseReturnData);
    event ChildResult(bool success, bytes returnData);

    constructor(IERC20 token_, string memory denom_) {
        token = token_;
        denom = denom_;
    }

    receive() external payable {}

    function send(
        string memory channel, string memory receiver, uint64 timeoutTimestamp,
        address recipient, uint256 directAmount, uint256 ibcAmount
    ) public returns (uint64 sequence, uint256 balanceAfter, bool reuseSuccess, bytes memory reuseReturnData) {
        if (directAmount != 0) require(token.transfer(recipient, directAmount), "direct transfer failed");
        sequence = _send(channel, receiver, timeoutTimestamp, ibcAmount);
        balanceAfter = token.balanceOf(address(this));
        (reuseSuccess, reuseReturnData) = address(token).call(abi.encodeCall(IERC20.transfer, (recipient, ibcAmount)));
        emit SendResult(sequence, balanceAfter, reuseSuccess, reuseReturnData);
    }

    function sendThenRevert(
        string memory channel, string memory receiver, uint64 timeoutTimestamp,
        address recipient, uint256 directAmount, uint256 ibcAmount
    ) external {
        send(channel, receiver, timeoutTimestamp, recipient, directAmount, ibcAmount);
        revert ForcedOuterRevert();
    }

    function catchChildRevert(
        string memory channel, string memory receiver, uint64 timeoutTimestamp,
        address recipient, uint256 directAmount, uint256 ibcAmount, uint256 parentAmount
    ) external {
        (bool success, bytes memory result) = address(this).call(abi.encodeCall(
            this.sendThenRevert, (channel, receiver, timeoutTimestamp, recipient, directAmount, ibcAmount)
        ));
        emit ChildResult(success, result);
        require(token.transfer(recipient, parentAmount), "parent transfer failed");
    }

    function sendRepeated(
        string memory channel, string memory receiver, uint64 timeoutTimestamp,
        address recipient, uint256 directAmount, uint256 firstAmount, uint256 secondAmount
    ) external returns (uint64 first, uint64 second) {
        if (directAmount != 0) require(token.transfer(recipient, directAmount), "direct transfer failed");
        first = _send(channel, receiver, timeoutTimestamp, firstAmount);
        second = _send(channel, receiver, timeoutTimestamp, secondAmount);
    }

    function _send(string memory channel, string memory receiver, uint64 timeoutTimestamp, uint256 amount)
        private returns (uint64)
    {
        return ICS20.transfer("transfer", channel, denom, amount, address(this), receiver,
            IICS20Xerc20.Height(0, 0), timeoutTimestamp, "");
    }
}

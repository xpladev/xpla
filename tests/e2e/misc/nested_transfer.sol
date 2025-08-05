// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import {WASM_PRECOMPILE_ADDRESS, IWasm} from "../../../precompile/wasm/IWasm.sol";
import {AUTH_PRECOMPILE_ADDRESS, IAuth} from "../../../precompile/auth/IAuth.sol";
import {BANK_PRECOMPILE_ADDRESS, IBank} from "../../../precompile/bank/IBank.sol";
import {Coin} from "../../../precompile/util/Types.sol";
import "@openzeppelin/contracts/utils/Strings.sol";

contract NetstedTransfer {
    IAuth authContract = IAuth(AUTH_PRECOMPILE_ADDRESS);
    IWasm wasmContract = IWasm(WASM_PRECOMPILE_ADDRESS);
    IBank bankContract = IBank(BANK_PRECOMPILE_ADDRESS);

    string erc20Denom;
    string cw20ContractAddress;
    address evmAddressCw20Contract;

    constructor(address _token, string memory _cw20ContractAddress) {
        erc20Denom = string.concat("erc20/", Strings.toHexString(_token));
        cw20ContractAddress = _cw20ContractAddress;
        evmAddressCw20Contract = authContract.addressStringToBytes(cw20ContractAddress);
    }

    function executeTransfer(address to, uint112 value) external {
        Coin[] memory fund = new Coin[](1);
        fund[0] = Coin({denom: erc20Denom, amount: 1});

        string memory recipient = authContract.addressBytesToString(to);
        string memory stringValue = Strings.toString(value);

        // dummy CW20 transfer
        bytes memory contractMsg = bytes.concat(
            '{"transfer": {"recipient": "',
            bytes(recipient),
            '","amount": "',
            bytes(stringValue),
            '"}}'
        );

        wasmContract.executeContract(
            msg.sender,
            evmAddressCw20Contract,
            contractMsg,
            fund // this is the purpose
        );
    }

    function executeBankTransfer(address to, uint112 value) external {
        Coin[] memory fund = new Coin[](1);
        fund[0] = Coin({denom: erc20Denom, amount: value});
        bankContract.send(msg.sender, to, fund);
    }
}

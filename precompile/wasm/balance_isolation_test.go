package wasm_test

import (
	"bytes"
	"context"
	"errors"
	"math/big"
	"testing"

	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	cmn "github.com/cosmos/evm/precompiles/common"
	"github.com/cosmos/evm/x/vm/statedb"
	vmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	pwasm "github.com/xpladev/xpla/precompile/wasm"
	"github.com/xpladev/xpla/tests/integration/testutil"
	xplatypes "github.com/xpladev/xpla/types"
)

func TestWasmEntryPointPreservesExactBankBalanceOwnership(t *testing.T) {
	callModes := []struct {
		name     string
		delegate bool
	}{
		{name: "regular"},
		{name: "delegate", delegate: true},
	}
	tests := []struct {
		name                  string
		actionErr             error
		malformedBalanceEvent bool
		outerRevert           bool
		wantError             bool
		wantSenderBalance     int64
		wantContractBalance   int64
		wantRecipientBalance  int64
	}{
		{
			name:                 "success",
			wantSenderBalance:    993,
			wantContractBalance:  34,
			wantRecipientBalance: 8,
		},
		{
			name:                 "native error",
			actionErr:            errors.New("native action failed"),
			wantError:            true,
			wantSenderBalance:    1000,
			wantContractBalance:  30,
			wantRecipientBalance: 5,
		},
		{
			name:                  "balance event error",
			malformedBalanceEvent: true,
			wantError:             true,
			wantSenderBalance:     1000,
			wantContractBalance:   30,
			wantRecipientBalance:  5,
		},
		{
			name:                 "outer revert",
			outerRevert:          true,
			wantSenderBalance:    1000,
			wantContractBalance:  30,
			wantRecipientBalance: 5,
		},
	}

	for _, mode := range callModes {
		for _, tc := range tests {
			t.Run(mode.name+"/"+tc.name, func(t *testing.T) {
				input := testutil.CreateTestInput(t)
				sender := common.HexToAddress("0x1234")
				alias := common.HexToAddress("0x5678")
				senderAddr := sdk.AccAddress(sender.Bytes())
				aliasAddr := sdk.AccAddress(alias.Bytes())
				contractAddr := sdk.AccAddress(append(bytes.Repeat([]byte{1}, 12), alias.Bytes()...))
				recipientAddr := sdk.AccAddress(common.HexToAddress("0x9876").Bytes())
				coins := func(amount int64) sdk.Coins {
					return sdk.NewCoins(sdk.NewCoin(xplatypes.DefaultDenom, sdkmath.NewInt(amount)))
				}
				for _, account := range []struct {
					address sdk.AccAddress
					amount  int64
				}{{senderAddr, 1000}, {aliasAddr, 10}, {contractAddr, 30}, {recipientAddr, 5}} {
					require.NoError(t, input.InitAccountWithCoins(account.address, coins(account.amount)))
				}
				require.NoError(t, input.App.WasmKeeper.RegisterWasmAlias(input.Ctx, aliasAddr, contractAddr))
				setTestContractMetadata(t, input, contractAddr)
				require.False(t, input.BankKeeper.BlockedAddr(contractAddr), "the native bank keeper must remain unchanged")
				supply := input.BankKeeper.GetSupply(input.Ctx, xplatypes.DefaultDenom)

				const budget = uint64(1_000_000)
				ctx := input.Ctx.WithGasMeter(storetypes.NewGasMeter(budget)).WithEventManager(sdk.NewEventManager())
				db := statedb.New(ctx, input.App.EvmKeeper, statedb.NewEmptyTxConfig())
				require.Equal(t, uint64(1000), db.GetBalance(sender).Uint64())
				require.Equal(t, uint64(10), db.GetBalance(alias).Uint64())
				require.Equal(t, uint64(5), db.GetBalance(common.BytesToAddress(recipientAddr)).Uint64())
				evm := vm.NewEVM(vm.BlockContext{BlockNumber: big.NewInt(1)}, db, vmtypes.GetEthChainConfig(), vm.Config{})
				evm.Origin = sender
				server := gasRecordingWasmServer{execute: func(goCtx context.Context, msg *wasmtypes.MsgExecuteContract) (*wasmtypes.MsgExecuteContractResponse, error) {
					actionCtx := sdk.UnwrapSDKContext(goCtx)
					require.Equal(t, contractAddr.String(), msg.Contract)
					require.NoError(t, input.BankKeeper.SendCoins(actionCtx, senderAddr, contractAddr, coins(7)))
					require.NoError(t, input.BankKeeper.SendCoins(actionCtx, contractAddr, recipientAddr, coins(3)))
					if tc.actionErr != nil {
						return nil, tc.actionErr
					}
					if tc.malformedBalanceEvent {
						actionCtx.EventManager().EmitEvent(sdk.NewEvent(banktypes.EventTypeCoinReceived,
							sdk.NewAttribute(banktypes.AttributeKeyReceiver, recipientAddr.String()),
							sdk.NewAttribute(sdk.AttributeKeyAmount, "invalid amount")))
					}
					return &wasmtypes.MsgExecuteContractResponse{}, nil
				}}
				p := pwasm.NewPrecompiledWasm(server,
					input.App.WasmKeeper, input.BankKeeper)
				contract := vm.NewContract(sender, pwasm.Address, uint256.NewInt(0), budget, nil)
				var err error
				contract.Input, err = pwasm.ABI.Pack("executeContract", sender, alias, []byte(`{}`), cmn.NewCoinsResponse(nil))
				require.NoError(t, err)
				snapshot := db.Snapshot()
				if mode.delegate {
					_, err = p.RunDelegate(evm, contract, false)
				} else {
					_, err = p.Run(evm, contract, false)
				}
				cacheCtx, cacheErr := db.GetCacheContext()
				require.NoError(t, cacheErr)
				if tc.wantError {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
					require.Equal(t, uint64(993), db.GetBalance(sender).Uint64())
					require.Equal(t, uint64(10), db.GetBalance(alias).Uint64())
					require.Equal(t, uint64(8), db.GetBalance(common.BytesToAddress(recipientAddr)).Uint64())
					require.Contains(t, cacheCtx.EventManager().Events(), banktypes.NewCoinReceivedEvent(contractAddr, coins(7)))
					require.Contains(t, cacheCtx.EventManager().Events(), banktypes.NewCoinSpentEvent(contractAddr, coins(3)))
				}

				if tc.wantError || tc.outerRevert {
					// The EVM frame owns rollback for late handler errors and outer reverts.
					db.RevertToSnapshot(snapshot)
					require.Empty(t, cacheCtx.EventManager().Events())
					require.Empty(t, db.Logs())
				}
				require.NoError(t, db.Commit())
				require.Equal(t, coins(tc.wantSenderBalance), input.BankKeeper.GetAllBalances(ctx, senderAddr))
				require.Equal(t, coins(10), input.BankKeeper.GetAllBalances(ctx, aliasAddr))
				require.Equal(t, coins(tc.wantContractBalance), input.BankKeeper.GetAllBalances(ctx, contractAddr))
				require.Equal(t, coins(tc.wantRecipientBalance), input.BankKeeper.GetAllBalances(ctx, recipientAddr))
				require.Equal(t, supply, input.BankKeeper.GetSupply(ctx, xplatypes.DefaultDenom))
			})
		}
	}
}

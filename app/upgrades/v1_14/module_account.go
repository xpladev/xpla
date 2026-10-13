package v1_14

import (
	"fmt"
	"slices"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	xplaauthkeeper "github.com/xpladev/xpla/x/auth/keeper"
	dynamicdeflationtypes "github.com/xpladev/xpla/x/dynamicdeflation/types"
)

func ensureDynamicDeflationPoolAccount(ctx sdk.Context, accountKeeper xplaauthkeeper.AccountKeeper) error {
	const moduleName = dynamicdeflationtypes.PoolName
	expectedPermissions := []string{authtypes.Burner}

	moduleAddress, configuredPermissions := accountKeeper.GetModuleAddressAndPermissions(moduleName)
	if moduleAddress == nil {
		return fmt.Errorf("%s module address is not configured", moduleName)
	}
	if !moduleAddress.Equals(authtypes.NewModuleAddress(moduleName)) {
		return fmt.Errorf("%s module address is invalid: %s", moduleName, moduleAddress)
	}
	if !slices.Equal(configuredPermissions, expectedPermissions) {
		return fmt.Errorf("%s module permissions are %v, expected %v", moduleName, configuredPermissions, expectedPermissions)
	}

	account := accountKeeper.GetAccount(ctx, moduleAddress)
	var baseAccount *authtypes.BaseAccount
	switch account := account.(type) {
	case nil:
		if accountKeeper.GetModuleAccount(ctx, moduleName) == nil {
			return fmt.Errorf("create %s module account", moduleName)
		}
		return nil
	case *authtypes.BaseAccount:
		if account == nil {
			return fmt.Errorf("%s base account is nil", moduleName)
		}
		baseAccount = account
	case *authtypes.ModuleAccount:
		if account == nil || account.BaseAccount == nil {
			return fmt.Errorf("%s module account is not initialized", moduleName)
		}
		baseAccount = account.BaseAccount
	default:
		return fmt.Errorf("%s address contains unsupported account type %T", moduleName, account)
	}

	moduleAccount := authtypes.NewModuleAccount(baseAccount, moduleName, expectedPermissions...)
	if err := moduleAccount.Validate(); err != nil {
		return fmt.Errorf("validate %s module account: %w", moduleName, err)
	}
	accountKeeper.SetModuleAccount(ctx, moduleAccount)

	return nil
}

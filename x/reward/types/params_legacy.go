package types

import (
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

// Parameter keys
var (
	ParamStoreKeyFeePoolRate             = []byte("feepoolrate")
	ParamStoreKeyCommunityPoolRate       = []byte("communitypoolrate")
	ParamStoreKeyReserveRate             = []byte("reserverate")
	ParamStoreKeyReserveAccount          = []byte("reserveaccount")
	ParamStoreKeyRewardDistributeAccount = []byte("rewarddistributeaccount")
)

// ParamKeyTable - Key declaration for parameters
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// ParamSetPairs returns the parameter set pairs.
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(ParamStoreKeyFeePoolRate, &p.FeePoolRate, validateFeePoolRate),
		paramtypes.NewParamSetPair(ParamStoreKeyCommunityPoolRate, &p.CommunityPoolRate, validateCommunityPoolRate),
		paramtypes.NewParamSetPair(ParamStoreKeyReserveRate, &p.ReserveRate, validateReserveRate),
		paramtypes.NewParamSetPair(ParamStoreKeyReserveAccount, &p.ReserveAccount, validateAccount),
		paramtypes.NewParamSetPair(ParamStoreKeyRewardDistributeAccount, &p.RewardDistributeAccount, validateAccount),
	}
}

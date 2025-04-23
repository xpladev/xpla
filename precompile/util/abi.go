package util

import (
	"bytes"
	"embed"
	"errors"
	"math/big"
	"reflect"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	argsOffset = 4
)

type Coin struct {
	Denom  string   `json:"denom"`
	Amount *big.Int `json:"amount"`
}

func LoadABI(fs embed.FS, fileName string) (abi.ABI, error) {
	abiBz, err := fs.ReadFile(fileName)
	if err != nil {
		return abi.ABI{}, err
	}

	resAbi, err := abi.JSON(bytes.NewReader(abiBz))
	if err != nil {
		return abi.ABI{}, err
	}

	return resAbi, nil
}

func SplitInput(input []byte) (method []byte, args []byte) {
	return input[:argsOffset], input[argsOffset:]
}

func GetBool(src interface{}) (bool, error) {
	res, ok := src.(bool)
	if !ok {
		return false, errors.New("invalid bool")
	}
	return res, nil
}

func GetAccAddress(src interface{}) (sdk.AccAddress, error) {
	res, ok := src.(common.Address)
	if !ok {
		return nil, errors.New("invalid addr")
	}
	return sdk.AccAddress(res.Bytes()), nil
}

func GetBigInt(src interface{}) (sdkmath.Int, error) {
	res, ok := src.(*big.Int)
	if !ok {
		return sdkmath.ZeroInt(), errors.New("invalid big int")
	}
	return sdkmath.NewIntFromBigInt(res), nil
}

func GetString(src interface{}) (string, error) {
	res, ok := src.(string)
	if !ok {
		return "", errors.New("invalid string")
	}
	return res, nil
}

func GetByteArray(src interface{}) ([]byte, error) {
	res, ok := src.([]byte)
	if !ok {
		return []byte{}, errors.New("invalid byte array")
	}
	return res, nil
}

func GetCoin(src interface{}) (sdk.Coin, error) {
	val := reflect.ValueOf(src)
	if val.Kind() != reflect.Struct {
		return sdk.Coin{}, errors.New("invalid coin / struct")
	}

	denomField := val.FieldByName("Denom")
	amountField := val.FieldByName("Amount")

	if !denomField.IsValid() || !amountField.IsValid() {
		return sdk.Coin{}, errors.New("invalid coin / field")
	}

	denom, ok := denomField.Interface().(string)
	if !ok {
		return sdk.Coin{}, errors.New("invalid coin / denom")
	}

	amount, ok := amountField.Interface().(*big.Int)
	if !ok {
		return sdk.Coin{}, errors.New("invalid coin / amount")
	}

	return sdk.NewCoin(denom, sdkmath.NewIntFromBigInt(amount)), nil
}

func GetCoins(src interface{}) (sdk.Coins, error) {
	val := reflect.ValueOf(src)
	inputType := reflect.TypeOf(src)

	if inputType.Kind() != reflect.Slice && inputType.Kind() != reflect.Array {
		return nil, errors.New("invalid coins / not array-like input")
	}

	coins := make([]sdk.Coin, 0, val.Len())

	for i := 0; i < val.Len(); i++ {
		item := val.Index(i)

		coin, err := GetCoin(item.Interface())
		if err != nil {
			return nil, err
		}

		coins = append(coins, coin)
	}

	return sdk.NewCoins(coins...), nil
}

func ValidateSigner(sender sdk.AccAddress, caller common.Address) error {
	if !bytes.Equal(sender.Bytes(), caller.Bytes()) {
		return errors.New("invalid signer")
	}
	return nil
}

package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
)

func MustMarshalXatp(cdc codec.BinaryCodec, xatp *XATP) []byte {
	return cdc.MustMarshal(xatp)
}

func UnmarshalXatp(cdc codec.BinaryCodec, value []byte) (xatp XATP, err error) {
	err = cdc.Unmarshal(value, &xatp)
	return xatp, err
}

func MustUnmarshalXatp(cdc codec.BinaryCodec, value []byte) XATP {
	xatp, err := UnmarshalXatp(cdc, value)
	if err != nil {
		panic(err)
	}

	return xatp
}

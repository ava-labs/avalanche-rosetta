package mapper

import (
	"github.com/ava-labs/libevm/common"
	"github.com/coinbase/rosetta-sdk-go/types"
)

func Account(address *common.Address) *types.AccountIdentifier {
	if address == nil {
		return nil
	}
	return &types.AccountIdentifier{
		Address: address.String(),
	}
}

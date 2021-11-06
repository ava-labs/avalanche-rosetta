package mapper

import (
	"github.com/coinbase/rosetta-sdk-go/types"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

func Account(address *ethcommon.Address) *types.AccountIdentifier {
	if address == nil {
		return nil
	}
	return &types.AccountIdentifier{
		Address: address.String(),
	}
}

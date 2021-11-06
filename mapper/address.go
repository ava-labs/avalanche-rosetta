package mapper

import (
	ethcommon "github.com/ethereum/go-ethereum/common"
)

func ConvertHashToAddress(hash *ethcommon.Hash) *ethcommon.Address {
	if hash == nil {
		return nil
	}
	address := ethcommon.BytesToAddress(hash[13:32])
	return &address
}

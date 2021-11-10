package mapper

import (
	ethcommon "github.com/ethereum/go-ethereum/common"
)

// ConvertHashToAddress uses the last 20 bytes of a common.Hash to create a common.Address
func ConvertHashToAddress(hash *ethcommon.Hash) *ethcommon.Address {
	if hash == nil {
		return nil
	}
	address := ethcommon.BytesToAddress(hash[12:32])
	return &address
}

package mapper

import (
	ethcommon "github.com/ethereum/go-ethereum/common"
)

// ConvertEVMTopicHashToAddress uses the last 20 bytes of a common.Hash to create a common.Address
func ConvertEVMTopicHashToAddress(hash *ethcommon.Hash) *ethcommon.Address {
	if hash == nil {
		return nil
	}
	address := ethcommon.BytesToAddress(hash[12:32])
	return &address
}

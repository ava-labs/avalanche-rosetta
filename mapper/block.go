package mapper

import (
	corethTypes "github.com/ava-labs/coreth/core/types"
)

// BlockMetadata returns meta data for a block
func BlockMetadata(block *corethTypes.Block) map[string]interface{} {
	meta := map[string]interface{}{
		"gas_limit":  block.GasLimit(),
		"gas_used":   block.GasUsed(),
		"difficulty": block.Difficulty(),
		"nonce":      block.Nonce(),
		"size":       block.Size().String(),
	}
	if block.BaseFee() != nil {
		meta["base_fee"] = block.BaseFee().String()
	}
	return meta
}

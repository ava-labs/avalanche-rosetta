package mapper

import (
	"github.com/ava-labs/coreth/plugin/evm/customtypes"
	"github.com/ava-labs/libevm/common/hexutil"
	"github.com/ava-labs/libevm/core/types"
)

// BlockMetadata returns meta data for a block
func BlockMetadata(block *types.Block) map[string]interface{} {
	meta := map[string]interface{}{
		"gas_limit":  hexutil.EncodeUint64(block.GasLimit()),
		"gas_used":   hexutil.EncodeUint64(block.GasUsed()),
		"difficulty": block.Difficulty(),
		"nonce":      block.Nonce(),
		"size":       hexutil.EncodeUint64(block.Size()),
	}
	if block.BaseFee() != nil {
		meta["base_fee"] = hexutil.EncodeBig(block.BaseFee())
	}

	blockGasCost := customtypes.BlockGasCost(block)
	extDataGasUsed := customtypes.BlockExtDataGasUsed(block)
	if blockGasCost != nil {
		meta["block_gas_cost"] = hexutil.EncodeBig(blockGasCost)
	}
	if extDataGasUsed != nil {
		meta["ext_data_gas_used"] = hexutil.EncodeBig(extDataGasUsed)
	}
	return meta
}

package service

import (
	"context"
	"math/big"

	"github.com/coinbase/rosetta-sdk-go/types"
	ethcommon "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/figment-networks/avalanche-rosetta/client"
)

func blockHeaderFromInput(evm *client.EvmClient, input *types.PartialBlockIdentifier) (*ethtypes.Header, *types.Error) {
	var (
		header *ethtypes.Header
		err    error
	)

	if input == nil {
		header, err = evm.HeaderByNumber(context.Background(), nil)
	} else {
		if input.Hash == nil && input.Index == nil {
			return nil, errInvalidInput
		}

		if input.Index != nil {
			header, err = evm.HeaderByNumber(context.Background(), big.NewInt(*input.Index))
		} else {
			header, err = evm.HeaderByHash(context.Background(), ethcommon.HexToHash(*input.Hash))
		}
	}

	if err != nil {
		return nil, errInternalError
	}

	return header, nil
}

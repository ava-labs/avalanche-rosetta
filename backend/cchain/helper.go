package cchain

import (
	"context"
	"math/big"

	"github.com/ava-labs/avalanche-rosetta/client"
	ethtypes "github.com/ava-labs/coreth/core/types"
	"github.com/coinbase/rosetta-sdk-go/types"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

func blockHeaderFromInput(
	ctx context.Context,
	c client.Client,
	input *types.PartialBlockIdentifier,
) (*ethtypes.Header, *types.Error) {
	var (
		header *ethtypes.Header
		err    error
	)

	if input == nil {
		header, err = c.HeaderByNumber(ctx, nil)
	} else {
		if input.Hash == nil && input.Index == nil {
			return nil, ErrInvalidInput
		}

		if input.Index != nil {
			header, err = c.HeaderByNumber(ctx, big.NewInt(*input.Index))
		} else {
			header, err = c.HeaderByHash(ctx, ethcommon.HexToHash(*input.Hash))
		}
	}

	if err != nil {
		return nil, WrapError(ErrInternalError, err)
	}

	return header, nil
}

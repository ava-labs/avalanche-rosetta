package service

import (
	"context"
	"math/big"
	"strings"

	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ava-labs/avalanche-rosetta/client"

	ethtypes "github.com/ava-labs/coreth/core/types"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

const (
	nativeTransferGasLimit = uint64(21000)
	erc20TransferGasLimit  = uint64(250000)
	unwrapGasLimit         = uint64(750000)
	genesisTimestamp       = 946713601000 // min allowable timestamp
)

func makeGenesisBlock(hash string) *types.Block {
	return &types.Block{
		BlockIdentifier: &types.BlockIdentifier{
			Index: 0,
			Hash:  hash,
		},
		ParentBlockIdentifier: &types.BlockIdentifier{
			Index: 0,
			Hash:  hash,
		},
		Timestamp: genesisTimestamp,
	}
}

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

// ChecksumAddress ensures an Ethereum hex address
// is in Checksum Format. If the address cannot be converted,
// it returns !ok.
func ChecksumAddress(address string) (string, bool) {
	if !strings.HasPrefix(address, "0x") {
		return "", false
	}

	addr, err := ethcommon.NewMixedcaseAddressFromString(address)
	if err != nil {
		return "", false
	}

	return addr.Address().Hex(), true
}

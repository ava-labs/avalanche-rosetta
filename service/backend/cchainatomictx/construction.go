package cchainatomictx

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/crypto"
	"github.com/ava-labs/avalanchego/utils/formatting/address"
	"github.com/coinbase/rosetta-sdk-go/parser"
	"github.com/coinbase/rosetta-sdk-go/types"
	ethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/ava-labs/avalanche-rosetta/mapper"
	cmapper "github.com/ava-labs/avalanche-rosetta/mapper/cchainatomictx"
	"github.com/ava-labs/avalanche-rosetta/service"
	"github.com/ava-labs/avalanche-rosetta/service/backend/common"
)

func (b *Backend) ConstructionDerive(ctx context.Context, req *types.ConstructionDeriveRequest) (*types.ConstructionDeriveResponse, *types.Error) {
	return common.DeriveBech32Address(b.fac, mapper.CChainNetworkIdentifier, req)
}

func (b *Backend) ConstructionPreprocess(
	ctx context.Context,
	req *types.ConstructionPreprocessRequest,
) (*types.ConstructionPreprocessResponse, *types.Error) {
	matches, err := common.MatchOperations(req.Operations)
	if err != nil {
		return nil, service.WrapError(service.ErrInvalidInput, err)
	}

	firstIn, _ := matches[0].First()
	firstOut, _ := matches[1].First()

	if firstIn == nil || firstOut == nil {
		return nil, service.WrapError(service.ErrInvalidInput, "both input and output operations must be specified")
	}

	gasUsed, err := b.estimateGasUsed(firstIn.Type, matches)
	if err != nil {
		return nil, service.WrapError(service.ErrInvalidInput, err)
	}

	preprocessOptions := cmapper.Options{
		AtomicTxGas: new(big.Int).SetUint64(gasUsed),
	}

	switch firstIn.Type {
	case mapper.OpImport:
		v, ok := req.Metadata[cmapper.MetadataSourceChain]
		if !ok {
			return nil, service.WrapError(service.ErrInvalidInput, "source_chain metadata must be provided")
		}
		chainAlias, ok := v.(string)
		if !ok {
			return nil, service.WrapError(service.ErrInvalidInput, "invalid source_chain value")
		}

		preprocessOptions.SourceChain = chainAlias
	case mapper.OpExport:
		chain, _, _, err := address.Parse(firstOut.Account.Address)
		if err != nil {
			return nil, service.WrapError(service.ErrInternalError, err)
		}

		preprocessOptions.From = firstIn.Account.Address
		preprocessOptions.DestinationChain = chain

		if v, ok := req.Metadata[cmapper.MetadataNonce]; ok {
			stringObj, ok := v.(string)
			if !ok {
				return nil, service.WrapError(service.ErrInvalidInput, fmt.Errorf("%s is not a valid nonce string", v))
			}
			bigObj, ok := new(big.Int).SetString(stringObj, 10)
			if !ok {
				return nil, service.WrapError(service.ErrInvalidInput, fmt.Errorf("%s is not a valid nonce", v))
			}
			preprocessOptions.Nonce = bigObj
		}
	}

	optionsMap, err := mapper.MarshalJSONMap(preprocessOptions)
	if err != nil {
		return nil, service.WrapError(service.ErrInternalError, err)
	}

	return &types.ConstructionPreprocessResponse{
		Options: optionsMap,
	}, nil
}

func (b *Backend) estimateGasUsed(opType string, matches []*parser.Match) (uint64, error) {
	// building tx with dummy data to get byte size for fee estimate
	tx, _, err := cmapper.BuildTx(opType, matches, cmapper.Metadata{
		SourceChainID:      &ids.Empty,
		DestinationChainID: &ids.Empty,
	}, b.codec, b.avaxAssetID)
	if err != nil {
		return 0, err
	}

	err = tx.Sign(b.codec, [][]*crypto.PrivateKeySECP256K1R{})
	if err != nil {
		return 0, err
	}

	return tx.GasUsed(true)
}

func (b *Backend) ConstructionMetadata(
	ctx context.Context,
	req *types.ConstructionMetadataRequest,
) (*types.ConstructionMetadataResponse, *types.Error) {
	var input cmapper.Options
	err := mapper.UnmarshalJSONMap(req.Options, &input)
	if err != nil {
		return nil, service.WrapError(service.ErrInvalidInput, err)
	}

	networkID, err := b.cClient.GetNetworkID(ctx)
	if err != nil {
		return nil, service.WrapError(service.ErrClientError, err)
	}

	cChainID, err := b.cClient.GetBlockchainID(ctx, mapper.CChainNetworkIdentifier)
	if err != nil {
		return nil, service.WrapError(service.ErrClientError, err)
	}

	metadata := cmapper.Metadata{
		NetworkID: networkID,
		CChainID:  cChainID,
	}

	if input.SourceChain != "" {
		id, err := b.cClient.GetBlockchainID(ctx, input.SourceChain)
		if err != nil {
			return nil, service.WrapError(service.ErrClientError, err)
		}
		metadata.SourceChainID = &id
	}

	if input.DestinationChain != "" {
		id, err := b.cClient.GetBlockchainID(ctx, input.DestinationChain)
		if err != nil {
			return nil, service.WrapError(service.ErrClientError, err)
		}
		metadata.DestinationChain = input.DestinationChain
		metadata.DestinationChainID = &id
	}

	if input.From != "" {
		var nonce uint64
		if input.Nonce == nil {
			nonce, err = b.cClient.NonceAt(ctx, ethcommon.HexToAddress(input.From), nil)
			if err != nil {
				return nil, service.WrapError(service.ErrClientError, err)
			}
		} else {
			nonce = input.Nonce.Uint64()
		}
		metadata.Nonce = nonce
	}

	metadataMap, err := mapper.MarshalJSONMap(metadata)
	if err != nil {
		return nil, service.WrapError(service.ErrInternalError, err)
	}

	suggestedFeeAmount, err := b.calculateSuggestedFee(ctx, input.AtomicTxGas)
	if err != nil {
		return nil, service.WrapError(service.ErrClientError, err)
	}

	return &types.ConstructionMetadataResponse{
		Metadata: metadataMap,
		SuggestedFee: []*types.Amount{
			suggestedFeeAmount,
		},
	}, nil
}

func (b *Backend) calculateSuggestedFee(ctx context.Context, gasUsed *big.Int) (*types.Amount, error) {
	baseFee, err := b.cClient.EstimateBaseFee(ctx)
	if err != nil {
		return nil, err
	}

	suggestedFeeEth := new(big.Int).Mul(gasUsed, baseFee)
	suggestedFee := new(big.Int).Div(suggestedFeeEth, mapper.X2crate)
	return mapper.AtomicAvaxAmount(suggestedFee), nil
}

func (b *Backend) ConstructionPayloads(ctx context.Context, req *types.ConstructionPayloadsRequest) (*types.ConstructionPayloadsResponse, *types.Error) {
	return nil, service.ErrNotImplemented
}

func (b *Backend) ConstructionParse(ctx context.Context, req *types.ConstructionParseRequest) (*types.ConstructionParseResponse, *types.Error) {
	return nil, service.ErrNotImplemented
}

func (b *Backend) ConstructionCombine(ctx context.Context, req *types.ConstructionCombineRequest) (*types.ConstructionCombineResponse, *types.Error) {
	return nil, service.ErrNotImplemented
}

func (b *Backend) ConstructionHash(ctx context.Context, req *types.ConstructionHashRequest) (*types.TransactionIdentifierResponse, *types.Error) {
	return nil, service.ErrNotImplemented
}

func (b *Backend) ConstructionSubmit(ctx context.Context, req *types.ConstructionSubmitRequest) (*types.TransactionIdentifierResponse, *types.Error) {
	return nil, service.ErrNotImplemented
}

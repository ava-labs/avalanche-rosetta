package pchain

import (
	"context"
	"fmt"
	"math/big"

	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ava-labs/avalanche-rosetta/mapper"
	pmapper "github.com/ava-labs/avalanche-rosetta/mapper/pchain"
	"github.com/ava-labs/avalanche-rosetta/service"
	"github.com/ava-labs/avalanche-rosetta/service/backend/common"
)

func (b *Backend) ConstructionDerive(ctx context.Context, req *types.ConstructionDeriveRequest) (*types.ConstructionDeriveResponse, *types.Error) {
	return common.DeriveBech32Address(b.fac, mapper.PChainNetworkIdentifier, req)
}

func (b *Backend) ConstructionPreprocess(
	ctx context.Context,
	req *types.ConstructionPreprocessRequest,
) (*types.ConstructionPreprocessResponse, *types.Error) {
	matches, err := common.MatchOperations(req.Operations)
	if err != nil {
		return nil, service.WrapError(service.ErrInvalidInput, err)
	}

	reqMetadata := req.Metadata
	if reqMetadata == nil {
		reqMetadata = make(map[string]interface{})
	}
	reqMetadata[pmapper.MetadataOpType] = matches[0].Operations[0].Type

	return &types.ConstructionPreprocessResponse{
		Options: reqMetadata,
	}, nil
}

func (b *Backend) ConstructionMetadata(
	ctx context.Context,
	req *types.ConstructionMetadataRequest,
) (*types.ConstructionMetadataResponse, *types.Error) {
	opMetadata, err := pmapper.ParseOpMetadata(req.Options)
	if err != nil {
		return nil, service.WrapError(service.ErrInvalidInput, err)
	}

	var suggestedFee *types.Amount
	var metadata *pmapper.Metadata
	switch opMetadata.Type {
	case pmapper.OpImportAvax:
		metadata, suggestedFee, err = b.buildImportMetadata(ctx, req.Options)
	case pmapper.OpExportAvax:
		metadata, suggestedFee, err = b.buildExportMetadata(ctx, req.Options)
	case pmapper.OpAddValidator, pmapper.OpAddDelegator:
		metadata, suggestedFee, err = b.buildStakingMetadata(req.Options)
	default:
		return nil, service.WrapError(
			service.ErrInternalError,
			fmt.Errorf("invalid tx type for building metadata: %s", opMetadata.Type),
		)
	}
	if err != nil {
		return nil, service.WrapError(service.ErrInternalError, err)
	}

	networkID, err := b.pClient.GetNetworkID(ctx)
	if err != nil {
		return nil, service.WrapError(service.ErrInvalidInput, err)
	}
	metadata.NetworkID = networkID

	pChainID, err := b.pClient.GetBlockchainID(ctx, mapper.PChainNetworkIdentifier)
	if err != nil {
		return nil, service.WrapError(service.ErrInvalidInput, err)
	}

	metadata.BlockchainID = pChainID

	metadataMap, err := mapper.MarshalJSONMap(metadata)
	if err != nil {
		return nil, service.WrapError(service.ErrInternalError, err)
	}

	return &types.ConstructionMetadataResponse{
		Metadata:     metadataMap,
		SuggestedFee: []*types.Amount{suggestedFee},
	}, nil
}

func (b *Backend) buildImportMetadata(ctx context.Context, options map[string]interface{}) (*pmapper.Metadata, *types.Amount, error) {
	var preprocessOptions pmapper.ImportExportOptions
	if err := mapper.UnmarshalJSONMap(options, &preprocessOptions); err != nil {
		return nil, nil, err
	}

	sourceChainID, err := b.pClient.GetBlockchainID(ctx, preprocessOptions.SourceChain)
	if err != nil {
		return nil, nil, err
	}

	suggestedFee, err := b.getBaseTxFee(ctx)
	if err != nil {
		return nil, nil, err
	}

	importMetadata := &pmapper.ImportMetadata{
		SourceChainID: sourceChainID,
	}

	return &pmapper.Metadata{ImportMetadata: importMetadata}, suggestedFee, nil
}

func (b *Backend) buildExportMetadata(ctx context.Context, options map[string]interface{}) (*pmapper.Metadata, *types.Amount, error) {
	var preprocessOptions pmapper.ImportExportOptions
	if err := mapper.UnmarshalJSONMap(options, &preprocessOptions); err != nil {
		return nil, nil, err
	}

	destinationChainID, err := b.pClient.GetBlockchainID(ctx, preprocessOptions.DestinationChain)
	if err != nil {
		return nil, nil, err
	}

	suggestedFee, err := b.getBaseTxFee(ctx)
	if err != nil {
		return nil, nil, err
	}

	exportMetadata := &pmapper.ExportMetadata{
		DestinationChain:   preprocessOptions.DestinationChain,
		DestinationChainID: destinationChainID,
	}

	return &pmapper.Metadata{ExportMetadata: exportMetadata}, suggestedFee, nil
}

func (b *Backend) buildStakingMetadata(options map[string]interface{}) (*pmapper.Metadata, *types.Amount, error) {
	var preprocessOptions pmapper.StakingOptions
	if err := mapper.UnmarshalJSONMap(options, &preprocessOptions); err != nil {
		return nil, nil, err
	}

	stakingMetadata := &pmapper.StakingMetadata{
		NodeID:          preprocessOptions.NodeID,
		Start:           preprocessOptions.Start,
		End:             preprocessOptions.End,
		Memo:            preprocessOptions.Memo,
		Locktime:        preprocessOptions.Locktime,
		Threshold:       preprocessOptions.Threshold,
		RewardAddresses: preprocessOptions.RewardAddresses,
		Shares:          preprocessOptions.Shares,
	}

	zeroAvax := mapper.AtomicAvaxAmount(big.NewInt(0))

	return &pmapper.Metadata{StakingMetadata: stakingMetadata}, zeroAvax, nil
}

func (b *Backend) getBaseTxFee(ctx context.Context) (*types.Amount, error) {
	fees, err := b.pClient.GetTxFee(ctx)
	if err != nil {
		return nil, err
	}

	feeAmount := new(big.Int).SetUint64(uint64(fees.TxFee))
	suggestedFee := mapper.AtomicAvaxAmount(feeAmount)
	return suggestedFee, nil
}

func (b *Backend) ConstructionPayloads(ctx context.Context, req *types.ConstructionPayloadsRequest) (*types.ConstructionPayloadsResponse, *types.Error) {
	builder := pTxBuilder{
		avaxAssetID:  b.avaxAssetID,
		codec:        b.codec,
		codecVersion: b.codecVersion,
	}
	return common.BuildPayloads(builder, req)
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

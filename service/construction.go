package service

import (
	"context"

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"

	ethcommon "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	ethrpl "github.com/ethereum/go-ethereum/rlp"

	"github.com/figment-networks/avalanche-rosetta/client"
)

// ConstructionService implements /construction/* endpoints
type ConstructionService struct {
	network *types.NetworkIdentifier
	evm     *client.EvmClient
}

// NewConstructionService returns a new contruction servicer
func NewConstructionService(network *types.NetworkIdentifier, evmClient *client.EvmClient) server.ConstructionAPIServicer {
	return &ConstructionService{
		network: network,
		evm:     evmClient,
	}
}

// ConstructionSubmit implements /construction/submit endpoint
func (s ConstructionService) ConstructionSubmit(ctx context.Context, req *types.ConstructionSubmitRequest) (*types.TransactionIdentifierResponse, *types.Error) {
	rawTx := ethcommon.Hex2Bytes(req.SignedTransaction)
	tx := &ethtypes.Transaction{}

	if err := ethrpl.DecodeBytes(rawTx, tx); err != nil {
		return nil, errorWithInfo(errConstructionInvalidTx, err)
	}

	if err := s.evm.SendTransaction(ctx, tx); err != nil {
		return nil, errorWithInfo(errConstructionSubmit, err)
	}

	return &types.TransactionIdentifierResponse{
		TransactionIdentifier: &types.TransactionIdentifier{
			Hash: tx.Hash().String(),
		},
	}, nil
}

// ConstructionCombine implements /construction/combine endpoint
func (s ConstructionService) ConstructionCombine(ctx context.Context, req *types.ConstructionCombineRequest) (*types.ConstructionCombineResponse, *types.Error) {
	return nil, errNotSupported
}

// ConstructionDerive implements /construction/derive endpoint
func (s ConstructionService) ConstructionDerive(ctx context.Context, req *types.ConstructionDeriveRequest) (*types.ConstructionDeriveResponse, *types.Error) {
	return nil, errNotSupported
}

// ConstructionHash implements /construction/hash endpoint
func (s ConstructionService) ConstructionHash(ctx context.Context, req *types.ConstructionHashRequest) (*types.TransactionIdentifierResponse, *types.Error) {
	return nil, errNotSupported
}

// ConstructionMetadata implements /construction/metadata endpoint
func (s ConstructionService) ConstructionMetadata(ctx context.Context, req *types.ConstructionMetadataRequest) (*types.ConstructionMetadataResponse, *types.Error) {
	return nil, errNotImplemented
}

// ConstructionParse implements /construction/parse endpoint
func (s ConstructionService) ConstructionParse(ctx context.Context, req *types.ConstructionParseRequest) (*types.ConstructionParseResponse, *types.Error) {
	return nil, errNotSupported
}

// ConstructionPayloads implements /construction/payloads endpoint
func (s ConstructionService) ConstructionPayloads(ctx context.Context, req *types.ConstructionPayloadsRequest) (*types.ConstructionPayloadsResponse, *types.Error) {
	return nil, errNotSupported
}

// ConstructionPreprocess implements /construction/preprocess endpoint
func (s ConstructionService) ConstructionPreprocess(ctx context.Context, req *types.ConstructionPreprocessRequest) (*types.ConstructionPreprocessResponse, *types.Error) {
	return nil, errNotSupported
}

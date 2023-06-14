package client

import (
	"context"
	"strings"

	"github.com/ava-labs/avalanchego/api"
	"github.com/ava-labs/avalanchego/api/info"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/indexer"
	"github.com/ava-labs/avalanchego/utils/rpc"
	"github.com/ava-labs/avalanchego/vms/avm"
	"github.com/ava-labs/avalanchego/vms/platformvm"
	"github.com/ava-labs/avalanchego/vms/platformvm/signer"

	"github.com/ava-labs/avalanche-rosetta/constants"
)

// Interface compliance
var _ PChainClient = &pchainClient{}

// PChainClient contains all client methods used to interact with avalanchego in order to support P-chain operations in Rosetta.
//
// These methods are cloned from the underlying avalanchego client interfaces, following the example of Client interface used to support C-chain operations.
type PChainClient interface {
	// info.Client methods
	InfoClient
	GetNodeID(context.Context, ...rpc.Option) (ids.NodeID, *signer.ProofOfPossession, error)
	GetTxFee(context.Context, ...rpc.Option) (*info.GetTxFeeResponse, error)

	// indexer.Client methods
	// Note: we use indexer only to be able to retrieve blocks by height.
	// Blocks by ID are retrieved via platformVM.GetBlock, thus ignoring the proposerVM part
	// and using Pchain Block ID rather than encompassing Snowman++ block ID
	GetContainerByIndex(ctx context.Context, index uint64, options ...rpc.Option) (indexer.Container, error)
	GetLastAccepted(context.Context, ...rpc.Option) (indexer.Container, uint64, error)

	// platformvm.Client methods
	GetUTXOs(
		ctx context.Context,
		addrs []ids.ShortID,
		limit uint32,
		startAddress ids.ShortID,
		startUTXOID ids.ID,
		options ...rpc.Option,
	) ([][]byte, ids.ShortID, ids.ID, error)
	GetAtomicUTXOs(
		ctx context.Context,
		addrs []ids.ShortID,
		sourceChain string,
		limit uint32,
		startAddress ids.ShortID,
		startUTXOID ids.ID,
		options ...rpc.Option,
	) ([][]byte, ids.ShortID, ids.ID, error)
	GetRewardUTXOs(context.Context, *api.GetTxArgs, ...rpc.Option) ([][]byte, error)
	GetHeight(ctx context.Context, options ...rpc.Option) (uint64, error)
	GetBalance(ctx context.Context, addrs []ids.ShortID, options ...rpc.Option) (*platformvm.GetBalanceResponse, error)
	GetTx(ctx context.Context, txID ids.ID, options ...rpc.Option) ([]byte, error)
	GetBlock(ctx context.Context, blockID ids.ID, options ...rpc.Option) ([]byte, error)
	IssueTx(ctx context.Context, tx []byte, options ...rpc.Option) (ids.ID, error)
	GetStake(ctx context.Context, addrs []ids.ShortID, validatorsOnly bool, options ...rpc.Option) (map[ids.ID]uint64, [][]byte, error)
	GetCurrentValidators(ctx context.Context, subnetID ids.ID, nodeIDs []ids.NodeID, options ...rpc.Option) ([]platformvm.ClientPermissionlessValidator, error)

	// avm.Client methods
	GetAssetDescription(ctx context.Context, assetID string, options ...rpc.Option) (*avm.GetAssetDescriptionReply, error)
}

type (
	indexerClient    = indexer.Client
	platformvmClient = platformvm.Client
	infoClient       = info.Client
)

type pchainClient struct {
	platformvmClient
	indexerClient
	infoClient
	xChainClient avm.Client
}

// NewPChainClient returns a new client for Avalanche APIs related to P-chain
func NewPChainClient(_ context.Context, rpcBaseURL, indexerBaseURL string) PChainClient {
	rpcBaseURL = strings.TrimSuffix(rpcBaseURL, "/")

	return pchainClient{
		platformvmClient: platformvm.NewClient(rpcBaseURL),
		xChainClient:     avm.NewClient(rpcBaseURL, constants.XChain.String()),
		infoClient:       info.NewClient(rpcBaseURL),
		indexerClient:    indexer.NewClient(indexerBaseURL + "/ext/index/P/block"),
	}
}

func (p pchainClient) GetAssetDescription(ctx context.Context, assetID string, options ...rpc.Option) (*avm.GetAssetDescriptionReply, error) {
	return p.xChainClient.GetAssetDescription(ctx, assetID, options...)
}

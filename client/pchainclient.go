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
)

// Interface compliance
var _ PChainClient = &pchainClient{}

type PChainClient interface {
	// indexer.Client methods
	GetContainerByIndex(ctx context.Context, index uint64, options ...rpc.Option) (indexer.Container, error)
	GetContainerByID(ctx context.Context, containerID ids.ID, options ...rpc.Option) (indexer.Container, error)
	GetLastAccepted(context.Context, ...rpc.Option) (indexer.Container, error)

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
	GetStake(ctx context.Context, addrs []ids.ShortID, options ...rpc.Option) (uint64, [][]byte, error)

	// avm.Client methods

	GetAssetDescription(ctx context.Context, assetID string, options ...rpc.Option) (*avm.GetAssetDescriptionReply, error)

	// info.Client methods
	IsBootstrapped(context.Context, string, ...rpc.Option) (bool, error)
	Peers(context.Context, ...rpc.Option) ([]info.Peer, error)
	GetNodeID(context.Context, ...rpc.Option) (ids.NodeID, error)
	GetNetworkID(context.Context, ...rpc.Option) (uint32, error)
	GetBlockchainID(context.Context, string, ...rpc.Option) (ids.ID, error)
	GetTxFee(context.Context, ...rpc.Option) (*info.GetTxFeeResponse, error)
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
func NewPChainClient(ctx context.Context, endpoint, indexerEndpoint string) PChainClient {
	endpoint = strings.TrimSuffix(endpoint, "/")

	return pchainClient{
		platformvmClient: platformvm.NewClient(endpoint),
		xChainClient:     avm.NewClient(endpoint, "X"),
		infoClient:       info.NewClient(endpoint),
		indexerClient:    indexer.NewClient(indexerEndpoint + "/ext/index/P/block"),
	}
}

func (p pchainClient) GetAssetDescription(ctx context.Context, assetID string, options ...rpc.Option) (*avm.GetAssetDescriptionReply, error) {
	return p.xChainClient.GetAssetDescription(ctx, assetID, options...)
}

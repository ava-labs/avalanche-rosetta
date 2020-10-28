package service

import (
	"context"
	"strconv"

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"

	ethcommon "github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"

	"github.com/figment-networks/avalanche-rosetta/client"
	"github.com/figment-networks/avalanche-rosetta/mapper"
)

// ConstructionService implements /construction/* endpoints
type ConstructionService struct {
	config *Config
	evm    *client.EvmClient
}

// NewConstructionService returns a new contruction servicer
func NewConstructionService(config *Config, evmClient *client.EvmClient) server.ConstructionAPIServicer {
	return &ConstructionService{
		config: config,
		evm:    evmClient,
	}
}

// ConstructionMetadata implements /construction/metadata endpoint.
//
// Get any information required to construct a transaction for a specific network.
// Metadata returned here could be a recent hash to use, an account sequence number,
// or even arbitrary chain state. The request used when calling this endpoint
// is created by calling /construction/preprocess in an offline environment.
//
func (s ConstructionService) ConstructionMetadata(ctx context.Context, req *types.ConstructionMetadataRequest) (*types.ConstructionMetadataResponse, *types.Error) {
	if s.config.IsOfflineMode() {
		return nil, errUnavailableOffline
	}

	from, ok := req.Options["from"].(string)
	if !ok {
		return nil, errorWithInfo(errInvalidInput, "from address is not provided")
	}

	nonce, err := s.evm.Client.PendingNonceAt(context.Background(), ethcommon.HexToAddress(from))
	if err != nil {
		return nil, errorWithInfo(errInternalError, err)
	}

	gasPrice, err := s.evm.Client.SuggestGasPrice(context.Background())
	if err != nil {
		return nil, errorWithInfo(errInternalError, err)
	}

	suggestedFee := gasPrice.Int64() * int64(21000)

	return &types.ConstructionMetadataResponse{
		Metadata: map[string]interface{}{
			"nonce":     nonce,
			"gas_price": gasPrice,
		},
		SuggestedFee: []*types.Amount{
			{
				Value:    strconv.FormatInt(suggestedFee, 10),
				Currency: mapper.AvaxCurrency,
			},
		},
	}, nil
}

// ConstructionHash implements /construction/hash endpoint.
//
// TransactionHash returns the network-specific transaction hash for a signed transaction.
//
func (s ConstructionService) ConstructionHash(ctx context.Context, req *types.ConstructionHashRequest) (*types.TransactionIdentifierResponse, *types.Error) {
	if req.SignedTransaction == "" {
		return nil, errorWithInfo(errInvalidInput, "signed transaction value is not provided")
	}

	tx, err := txFromInput(req.SignedTransaction)
	if err != nil {
		return nil, errorWithInfo(errConstructionInvalidTx, err)
	}

	return &types.TransactionIdentifierResponse{
		TransactionIdentifier: &types.TransactionIdentifier{
			Hash: tx.Hash().Hex(),
		},
	}, nil
}

// ConstructionCombine implements /construction/combine endpoint.
//
// Combine creates a network-specific transaction from an unsigned transaction
// and an array of provided signatures. The signed transaction returned from
// this method will be sent to the /construction/submit endpoint by the caller.
//
func (s ConstructionService) ConstructionCombine(ctx context.Context, req *types.ConstructionCombineRequest) (*types.ConstructionCombineResponse, *types.Error) {
	if s.config.IsOfflineMode() {
		return nil, errUnavailableOffline
	}
	if req.UnsignedTransaction == "" {
		return nil, errorWithInfo(errInvalidInput, "transaction data is not provided")
	}
	if len(req.Signatures) == 0 {
		return nil, errorWithInfo(errInvalidInput, "signature is not provided")
	}

	tx, err := unsignedTxFromInput(req.UnsignedTransaction)
	if err != nil {
		return nil, errorWithInfo(errConstructionInvalidTx, err)
	}

	signedTx, err := tx.WithSignature(s.config.Signer(), req.Signatures[0].Bytes)
	if err != nil {
		return nil, errorWithInfo(errInternalError, err)
	}

	txData, err := signedTx.MarshalJSON()
	if err != nil {
		return nil, errorWithInfo(errInternalError, err)
	}

	return &types.ConstructionCombineResponse{
		SignedTransaction: string(txData),
	}, nil
}

// ConstructionDerive implements /construction/derive endpoint.
//
// Derive returns the AccountIdentifier associated with a public key. Blockchains
// that require an on-chain action to create an account should not implement this method.
//
func (s ConstructionService) ConstructionDerive(ctx context.Context, req *types.ConstructionDeriveRequest) (*types.ConstructionDeriveResponse, *types.Error) {
	if req.PublicKey == nil {
		return nil, errorWithInfo(errInvalidInput, "public key is not provided")
	}

	key, err := ethcrypto.DecompressPubkey(req.PublicKey.Bytes)
	if err != nil {
		return nil, errorWithInfo(errConstructionInvalidPubkey, err)
	}

	return &types.ConstructionDeriveResponse{
		AccountIdentifier: &types.AccountIdentifier{
			Address: ethcrypto.PubkeyToAddress(*key).Hex(),
		},
	}, nil
}

// ConstructionParse implements /construction/parse endpoint
//
// Parse is called on both unsigned and signed transactions to understand the
// intent of the formulated transaction. This is run as a sanity check before signing
// (after /construction/payloads) and before broadcast (after /construction/combine).
//
func (s ConstructionService) ConstructionParse(ctx context.Context, req *types.ConstructionParseRequest) (*types.ConstructionParseResponse, *types.Error) {
	return nil, errNotSupported
}

// ConstructionPayloads implements /construction/payloads endpoint
//
// Payloads is called with an array of operations and the response from /construction/metadata.
// It returns an unsigned transaction blob and a collection of payloads that must
// be signed by particular AccountIdentifiers using a certain SignatureType.
// The array of operations provided in transaction construction often times can
// not specify all "effects" of a transaction (consider invoked transactions in Ethereum).
// However, they can deterministically specify the "intent" of the transaction,
// which is sufficient for construction. For this reason, parsing the corresponding
// transaction in the Data API (when it lands on chain) will contain a superset of
// whatever operations were provided during construction.
//
func (s ConstructionService) ConstructionPayloads(ctx context.Context, req *types.ConstructionPayloadsRequest) (*types.ConstructionPayloadsResponse, *types.Error) {
	return nil, errNotSupported
}

// ConstructionPreprocess implements /construction/preprocess endpoint.
//
// Preprocess is called prior to /construction/payloads to construct a request for
// any metadata that is needed for transaction construction given (i.e. account nonce).
//
func (s ConstructionService) ConstructionPreprocess(ctx context.Context, req *types.ConstructionPreprocessRequest) (*types.ConstructionPreprocessResponse, *types.Error) {
	return nil, errNotSupported
}

// ConstructionSubmit implements /construction/submit endpoint.
//
// Submit a pre-signed transaction to the node.
//
func (s ConstructionService) ConstructionSubmit(ctx context.Context, req *types.ConstructionSubmitRequest) (*types.TransactionIdentifierResponse, *types.Error) {
	if s.config.IsOfflineMode() {
		return nil, errUnavailableOffline
	}

	tx, err := txFromInput(req.SignedTransaction)
	if err != nil {
		return nil, errorWithInfo(errConstructionInvalidTx, err)
	}

	if err := s.evm.SendTransaction(ctx, tx); err != nil {
		return nil, errorWithInfo(errConstructionSubmitFailed, err)
	}

	return &types.TransactionIdentifierResponse{
		TransactionIdentifier: &types.TransactionIdentifier{
			Hash: tx.Hash().String(),
		},
	}, nil
}

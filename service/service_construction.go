package service

import (
	"context"
	"encoding/json"
	"math/big"

	"github.com/coinbase/rosetta-sdk-go/parser"
	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"

	ethcommon "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
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
	if !ok || from == "" {
		return nil, errorWithInfo(errInvalidInput, "from address is not provided")
	}

	balance, err := s.evm.Client.BalanceAt(context.Background(), ethcommon.HexToAddress(from), nil)
	if err != nil {
		return nil, errorWithInfo(errClientError, err)
	}

	nonce, err := s.evm.Client.PendingNonceAt(context.Background(), ethcommon.HexToAddress(from))
	if err != nil {
		return nil, errorWithInfo(errClientError, err)
	}

	gasPrice, err := s.evm.Client.SuggestGasPrice(context.Background())
	if err != nil {
		return nil, errorWithInfo(errClientError, err)
	}

	suggestedFee := gasPrice.Int64() * int64(transferGasLimit)

	return &types.ConstructionMetadataResponse{
		Metadata: map[string]interface{}{
			"nonce":         nonce,
			"balance":       balance,
			"gas_limit":     transferGasLimit,
			"gas_price":     gasPrice,
			"suggested_fee": suggestedFee,
		},
		SuggestedFee: []*types.Amount{
			mapper.FeeAmount(suggestedFee),
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
	var tx unsignedTx

	if !req.Signed {
		if err := json.Unmarshal([]byte(req.Transaction), &tx); err != nil {
			return nil, errorWithInfo(errInvalidInput, err)
		}
	} else {
		t := new(ethtypes.Transaction)
		if err := t.UnmarshalJSON([]byte(req.Transaction)); err != nil {
			return nil, errorWithInfo(errInvalidInput, err)
		}

		tx.To = t.To().String()
		tx.Value = t.Value()
		tx.Input = t.Data()
		tx.Nonce = t.Nonce()
		tx.GasPrice = t.GasPrice()
		tx.GasLimit = t.Gas()

		msg, err := t.AsMessage(s.config.Signer())
		if err != nil {
			return nil, errorWithInfo(errInvalidInput, err)
		}
		tx.From = msg.From().Hex()
	}

	ops := []*types.Operation{
		{
			Type: mapper.OpCall,
			OperationIdentifier: &types.OperationIdentifier{
				Index: 0,
			},
			Account: &types.AccountIdentifier{
				Address: tx.From,
			},
			Amount: &types.Amount{
				Value:    new(big.Int).Neg(tx.Value).String(),
				Currency: mapper.AvaxCurrency,
			},
		},
		{
			Type: mapper.OpCall,
			OperationIdentifier: &types.OperationIdentifier{
				Index: 1,
			},
			RelatedOperations: []*types.OperationIdentifier{
				{
					Index: 0,
				},
			},
			Account: &types.AccountIdentifier{
				Address: tx.To,
			},
			Amount: &types.Amount{
				Value:    tx.Value.String(),
				Currency: mapper.AvaxCurrency,
			},
		},
	}

	metadata := map[string]interface{}{
		"nonce":     tx.Nonce,
		"gas_price": tx.GasPrice,
		"gas_limit": tx.GasLimit,
	}

	if req.Signed {
		return &types.ConstructionParseResponse{
			Operations: ops,
			AccountIdentifierSigners: []*types.AccountIdentifier{
				{
					Address: tx.From,
				},
			},
			Metadata: metadata,
		}, nil
	}

	return &types.ConstructionParseResponse{
		Operations:               ops,
		AccountIdentifierSigners: []*types.AccountIdentifier{},
		Metadata:                 metadata,
	}, nil
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
	descriptions := &parser.Descriptions{
		OperationDescriptions: []*parser.OperationDescription{
			{
				Type: mapper.OpCall,
				Account: &parser.AccountDescription{
					Exists: true,
				},
				Amount: &parser.AmountDescription{
					Exists:   true,
					Sign:     parser.NegativeAmountSign,
					Currency: mapper.AvaxCurrency,
				},
			},
			{
				Type: mapper.OpCall,
				Account: &parser.AccountDescription{
					Exists: true,
				},
				Amount: &parser.AmountDescription{
					Exists:   true,
					Sign:     parser.PositiveAmountSign,
					Currency: mapper.AvaxCurrency,
				},
			},
		},
		ErrUnmatched: true,
	}

	matches, err := parser.MatchOperations(descriptions, req.Operations)
	if err != nil {
		return nil, errorWithInfo(errInvalidInput, "unclear intent")
	}
	tx, unTx, err := txFromMatches(matches, req.Metadata)
	if err != nil {
		return nil, errorWithInfo(errInternalError, "cant parse matches")
	}
	if tx == nil {
		return nil, errorWithInfo(errInternalError, "cant build eth transaction")
	}

	unsignedTxData, err := json.Marshal(unTx)
	if err != nil {
		return nil, errorWithInfo(errInternalError, err)
	}

	payload := &types.SigningPayload{
		AccountIdentifier: &types.AccountIdentifier{Address: unTx.From},
		Bytes:             s.config.Signer().Hash(tx).Bytes(),
		SignatureType:     types.EcdsaRecovery,
	}

	return &types.ConstructionPayloadsResponse{
		UnsignedTransaction: string(unsignedTxData),
		Payloads:            []*types.SigningPayload{payload},
	}, nil
}

// ConstructionPreprocess implements /construction/preprocess endpoint.
//
// Preprocess is called prior to /construction/payloads to construct a request for
// any metadata that is needed for transaction construction given (i.e. account nonce).
//
func (s ConstructionService) ConstructionPreprocess(ctx context.Context, req *types.ConstructionPreprocessRequest) (*types.ConstructionPreprocessResponse, *types.Error) {
	descriptions := &parser.Descriptions{
		OperationDescriptions: []*parser.OperationDescription{
			{
				Type: mapper.OpCall,
				Account: &parser.AccountDescription{
					Exists: true,
				},
				Amount: &parser.AmountDescription{
					Exists:   true,
					Sign:     parser.NegativeAmountSign,
					Currency: mapper.AvaxCurrency,
				},
			},
			{
				Type: mapper.OpCall,
				Account: &parser.AccountDescription{
					Exists: true,
				},
				Amount: &parser.AmountDescription{
					Exists:   true,
					Sign:     parser.PositiveAmountSign,
					Currency: mapper.AvaxCurrency,
				},
			},
		},
		ErrUnmatched: true,
	}

	matches, err := parser.MatchOperations(descriptions, req.Operations)
	if err != nil {
		return nil, errorWithInfo(errInvalidInput, "unclear intent")
	}

	fromOp, _ := matches[0].First()
	fromAddress := fromOp.Account.Address

	return &types.ConstructionPreprocessResponse{
		Options: map[string]interface{}{
			"from": fromAddress,
		},
	}, nil
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

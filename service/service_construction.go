package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/coinbase/rosetta-sdk-go/parser"
	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"
	"golang.org/x/crypto/sha3"

	"github.com/ava-labs/coreth/core"
	ethtypes "github.com/ava-labs/coreth/core/types"
	"github.com/ava-labs/coreth/interfaces"
	ethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/ethereum/go-ethereum/common/hexutil"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"

	"github.com/ava-labs/avalanche-rosetta/client"
	"github.com/ava-labs/avalanche-rosetta/mapper"
)

const (
	// 68 Bytes = methodID (4 Bytes) + param 1 (32 Bytes) + param 2 (32 Bytes)
	genericTransferBytesLength = 68
	genericUnwrapBytesLength   = 68

	requiredPaddingBytes = 32
	defaultUnwrapChainID = 0

	// do not include spaces in the Fn Signature strings
	transferFnSignature = "transfer(address,uint256)"
	unwrapFnSignature   = "unwrap(uint256,uint256)"
)

var (
	// preallocate methodIDs used in parse functions
	transferMethodID = hexutil.Encode(getMethodID(transferFnSignature))
	unwrapMethodID   = hexutil.Encode(getMethodID(unwrapFnSignature))
)

// ConstructionBackend represents a backend that implements /construction family of apis for a subset of requests.
// Endpoint handlers in this file delegates requests to corresponding backends based on the request.
// Each backend implements a ShouldHandleRequest method to determine whether that backend should handle the given request.
//
// P-chain and C-chain atomic transaction logic are implemented in pchain.Backend and cchainatomictx.Backend respectively.
// Eventually, the C-chain non-atomic transaction logic implemented in this file should be extracted to its own backend as well.
type ConstructionBackend interface {
	// ShouldHandleRequest returns whether a given request should be handled by this backend
	ShouldHandleRequest(req interface{}) bool
	// ConstructionDerive implements /construction/derive endpoint for this backend
	ConstructionDerive(ctx context.Context, req *types.ConstructionDeriveRequest) (*types.ConstructionDeriveResponse, *types.Error)
	// ConstructionPreprocess implements /construction/preprocess endpoint for this backend
	ConstructionPreprocess(ctx context.Context, req *types.ConstructionPreprocessRequest) (*types.ConstructionPreprocessResponse, *types.Error)
	// ConstructionMetadata implements /construction/metadata endpoint for this backend
	ConstructionMetadata(ctx context.Context, req *types.ConstructionMetadataRequest) (*types.ConstructionMetadataResponse, *types.Error)
	// ConstructionPayloads implements /construction/payloads endpoint for this backend
	ConstructionPayloads(ctx context.Context, req *types.ConstructionPayloadsRequest) (*types.ConstructionPayloadsResponse, *types.Error)
	// ConstructionParse implements /construction/parse endpoint for this backend
	ConstructionParse(ctx context.Context, req *types.ConstructionParseRequest) (*types.ConstructionParseResponse, *types.Error)
	// ConstructionCombine implements /construction/combine endpoint for this backend
	ConstructionCombine(ctx context.Context, req *types.ConstructionCombineRequest) (*types.ConstructionCombineResponse, *types.Error)
	// ConstructionHash implements /construction/hash endpoint for this backend
	ConstructionHash(ctx context.Context, req *types.ConstructionHashRequest) (*types.TransactionIdentifierResponse, *types.Error)
	// ConstructionSubmit implements /construction/submit endpoint for this backend
	ConstructionSubmit(ctx context.Context, req *types.ConstructionSubmitRequest) (*types.TransactionIdentifierResponse, *types.Error)
}

// ConstructionService implements /construction/* endpoints
type ConstructionService struct {
	config                *Config
	client                client.Client
	cChainAtomicTxBackend ConstructionBackend
	pChainBackend         ConstructionBackend
}

// NewConstructionService returns a new construction servicer
func NewConstructionService(
	config *Config,
	client client.Client,
	pChainBackend ConstructionBackend,
	cChainAtomicTxBackend ConstructionBackend,
) server.ConstructionAPIServicer {
	return &ConstructionService{
		config:                config,
		client:                client,
		cChainAtomicTxBackend: cChainAtomicTxBackend,
		pChainBackend:         pChainBackend,
	}
}

// ConstructionMetadata implements /construction/metadata endpoint.
//
// Get any information required to construct a transaction for a specific network.
// Metadata returned here could be a recent hash to use, an account sequence number,
// or even arbitrary chain state. The request used when calling this endpoint
// is created by calling /construction/preprocess in an offline environment.
func (s ConstructionService) ConstructionMetadata(
	ctx context.Context,
	req *types.ConstructionMetadataRequest,
) (*types.ConstructionMetadataResponse, *types.Error) {
	if s.config.IsOfflineMode() {
		return nil, ErrUnavailableOffline
	}

	if s.pChainBackend.ShouldHandleRequest(req) {
		return s.pChainBackend.ConstructionMetadata(ctx, req)
	}

	if s.cChainAtomicTxBackend.ShouldHandleRequest(req) {
		return s.cChainAtomicTxBackend.ConstructionMetadata(ctx, req)
	}

	var input options
	if err := mapper.UnmarshalJSONMap(req.Options, &input); err != nil {
		return nil, WrapError(ErrInvalidInput, err)
	}

	if len(input.From) == 0 {
		return nil, WrapError(ErrInvalidInput, "from address is not provided")
	}

	var nonce uint64
	var err error
	if input.Nonce == nil {
		nonce, err = s.client.NonceAt(ctx, ethcommon.HexToAddress(input.From), nil)
		if err != nil {
			return nil, WrapError(ErrClientError, err)
		}
	} else {
		nonce = input.Nonce.Uint64()
	}

	var gasPrice *big.Int
	if input.GasPrice == nil {
		if gasPrice, err = s.client.SuggestGasPrice(ctx); err != nil {
			return nil, WrapError(ErrClientError, err)
		}

		if input.SuggestedFeeMultiplier != nil {
			newGasPrice := new(big.Float).Mul(
				big.NewFloat(*input.SuggestedFeeMultiplier),
				new(big.Float).SetInt(gasPrice),
			)
			newGasPrice.Int(gasPrice)
		}
	} else {
		gasPrice = input.GasPrice
	}

	var gasLimit uint64
	if input.GasLimit == nil {
		if input.Currency == nil || types.Hash(input.Currency) == types.Hash(mapper.AvaxCurrency) {
			gasLimit, err = s.getNativeTransferGasLimit(ctx, input.To, input.From, input.Value)
			if err != nil {
				return nil, WrapError(ErrClientError, err)
			}
		} else {
			if input.Metadata != nil {
				if !input.Metadata.UnwrapBridgeTx {
					return nil, WrapError(ErrInvalidInput, "UnwrapBridgeTx must be populated if input.Metadata is provided")
				}

				gasLimit, err = s.getBridgeUnwrapTransferGasLimit(ctx, input.From, input.Value, input.Currency)
			} else {
				gasLimit, err = s.getErc20TransferGasLimit(ctx, input.To, input.From, input.Value, input.Currency)
			}
			if err != nil {
				return nil, WrapError(ErrClientError, err)
			}
		}
	} else {
		gasLimit = input.GasLimit.Uint64()
	}

	metadata := &metadata{
		Nonce:    nonce,
		GasPrice: gasPrice,
		GasLimit: gasLimit,
	}

	if input.Metadata != nil {
		if input.Metadata.UnwrapBridgeTx {
			metadata.UnwrapBridgeTx = true
		}
	}

	metadataMap, err := mapper.MarshalJSONMap(metadata)
	if err != nil {
		return nil, WrapError(ErrInternalError, err)
	}

	suggestedFee := gasPrice.Int64() * int64(gasLimit)
	return &types.ConstructionMetadataResponse{
		Metadata: metadataMap,
		SuggestedFee: []*types.Amount{
			mapper.AvaxAmount(big.NewInt(suggestedFee)),
		},
	}, nil
}

// ConstructionHash implements /construction/hash endpoint.
//
// TransactionHash returns the network-specific transaction hash for a signed transaction.
func (s ConstructionService) ConstructionHash(
	ctx context.Context,
	req *types.ConstructionHashRequest,
) (*types.TransactionIdentifierResponse, *types.Error) {
	if len(req.SignedTransaction) == 0 {
		return nil, WrapError(ErrInvalidInput, "signed transaction value is not provided")
	}

	if s.pChainBackend.ShouldHandleRequest(req) {
		return s.pChainBackend.ConstructionHash(ctx, req)
	}

	if s.cChainAtomicTxBackend.ShouldHandleRequest(req) {
		return s.cChainAtomicTxBackend.ConstructionHash(ctx, req)
	}

	var wrappedTx signedTransactionWrapper
	if err := json.Unmarshal([]byte(req.SignedTransaction), &wrappedTx); err != nil {
		return nil, WrapError(ErrInvalidInput, err)
	}

	var signedTx ethtypes.Transaction
	if err := signedTx.UnmarshalJSON(wrappedTx.SignedTransaction); err != nil {
		return nil, WrapError(ErrInvalidInput, err)
	}

	return &types.TransactionIdentifierResponse{
		TransactionIdentifier: &types.TransactionIdentifier{
			Hash: signedTx.Hash().Hex(),
		},
	}, nil
}

// ConstructionCombine implements /construction/combine endpoint.
//
// Combine creates a network-specific transaction from an unsigned transaction
// and an array of provided signatures. The signed transaction returned from
// this method will be sent to the /construction/submit endpoint by the caller.
func (s ConstructionService) ConstructionCombine(
	ctx context.Context,
	req *types.ConstructionCombineRequest,
) (*types.ConstructionCombineResponse, *types.Error) {
	if len(req.UnsignedTransaction) == 0 {
		return nil, WrapError(ErrInvalidInput, "transaction data is not provided")
	}
	if len(req.Signatures) == 0 {
		return nil, WrapError(ErrInvalidInput, "signature is not provided")
	}

	if s.pChainBackend.ShouldHandleRequest(req) {
		return s.pChainBackend.ConstructionCombine(ctx, req)
	}

	if s.cChainAtomicTxBackend.ShouldHandleRequest(req) {
		return s.cChainAtomicTxBackend.ConstructionCombine(ctx, req)
	}

	var unsignedTx transaction
	if err := json.Unmarshal([]byte(req.UnsignedTransaction), &unsignedTx); err != nil {
		return nil, WrapError(ErrInvalidInput, err)
	}

	ethTransaction := ethtypes.NewTransaction(
		unsignedTx.Nonce,
		ethcommon.HexToAddress(unsignedTx.To),
		unsignedTx.Value,
		unsignedTx.GasLimit,
		unsignedTx.GasPrice,
		unsignedTx.Data,
	)

	signer := ethtypes.LatestSignerForChainID(unsignedTx.ChainID)
	signedTx, err := ethTransaction.WithSignature(signer, req.Signatures[0].Bytes)
	if err != nil {
		return nil, WrapError(ErrInvalidInput, err)
	}

	signedTxJSON, err := signedTx.MarshalJSON()
	if err != nil {
		return nil, WrapError(ErrInternalError, err)
	}

	wrappedSignedTx := signedTransactionWrapper{
		SignedTransaction: signedTxJSON,
		Currency:          unsignedTx.Currency,
	}

	wrappedSignedTxJSON, err := json.Marshal(wrappedSignedTx)
	if err != nil {
		return nil, WrapError(ErrInternalError, err)
	}

	return &types.ConstructionCombineResponse{
		SignedTransaction: string(wrappedSignedTxJSON),
	}, nil
}

// ConstructionDerive implements /construction/derive endpoint.
//
// Derive returns the AccountIdentifier associated with a public key. Blockchains
// that require an on-chain action to create an account should not implement this method.
func (s ConstructionService) ConstructionDerive(
	ctx context.Context,
	req *types.ConstructionDeriveRequest,
) (*types.ConstructionDeriveResponse, *types.Error) {
	if req.PublicKey == nil {
		return nil, WrapError(ErrInvalidInput, "public key is not provided")
	}

	if s.pChainBackend.ShouldHandleRequest(req) {
		return s.pChainBackend.ConstructionDerive(ctx, req)
	}

	if s.cChainAtomicTxBackend.ShouldHandleRequest(req) {
		return s.cChainAtomicTxBackend.ConstructionDerive(ctx, req)
	}

	key, err := ethcrypto.DecompressPubkey(req.PublicKey.Bytes)
	if err != nil {
		return nil, WrapError(ErrInvalidInput, err)
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
func (s ConstructionService) ConstructionParse(
	ctx context.Context,
	req *types.ConstructionParseRequest,
) (*types.ConstructionParseResponse, *types.Error) {
	if s.pChainBackend.ShouldHandleRequest(req) {
		return s.pChainBackend.ConstructionParse(ctx, req)
	}

	if s.cChainAtomicTxBackend.ShouldHandleRequest(req) {
		return s.cChainAtomicTxBackend.ConstructionParse(ctx, req)
	}

	var tx transaction

	if !req.Signed {
		if err := json.Unmarshal([]byte(req.Transaction), &tx); err != nil {
			return nil, WrapError(ErrInvalidInput, err)
		}
	} else {
		var wrappedTx signedTransactionWrapper
		if err := json.Unmarshal([]byte(req.Transaction), &wrappedTx); err != nil {
			return nil, WrapError(ErrInvalidInput, err)
		}

		var t ethtypes.Transaction
		if err := t.UnmarshalJSON(wrappedTx.SignedTransaction); err != nil {
			return nil, WrapError(ErrInvalidInput, err)
		}

		tx.To = t.To().String()
		tx.Value = t.Value()
		tx.Data = t.Data()
		tx.Nonce = t.Nonce()
		tx.GasPrice = t.GasPrice()
		tx.GasLimit = t.Gas()
		tx.ChainID = s.config.ChainID
		tx.Currency = wrappedTx.Currency

		msg, err := core.TransactionToMessage(&t, s.config.Signer(), nil)
		if err != nil {
			return nil, WrapError(ErrInvalidInput, err)
		}
		tx.From = msg.From.Hex()
	}

	metadata := &parseMetadata{
		Nonce:    tx.Nonce,
		GasPrice: tx.GasPrice,
		GasLimit: tx.GasLimit,
		ChainID:  tx.ChainID,
	}
	metaMap, err := mapper.MarshalJSONMap(metadata)
	if err != nil {
		return nil, WrapError(ErrInternalError, err)
	}

	var (
		ops        []*types.Operation
		checkFrom  *string
		wrappedErr *types.Error
	)
	if len(tx.Data) != 0 {
		switch hexutil.Encode(tx.Data[:4]) {
		case transferMethodID:
			ops, checkFrom, wrappedErr = createTransferOps(tx)
		case unwrapMethodID:
			ops, checkFrom, wrappedErr = createUnwrapOps(tx)
		default:
			wrappedErr = WrapError(
				ErrInvalidInput,
				fmt.Errorf("method %x is not supported", tx.Data[:4]),
			)
		}
	} else {
		ops, checkFrom, wrappedErr = createTransferOps(tx)
	}
	if wrappedErr != nil {
		return nil, wrappedErr
	}

	if req.Signed {
		return &types.ConstructionParseResponse{
			Operations: ops,
			AccountIdentifierSigners: []*types.AccountIdentifier{
				{
					Address: *checkFrom,
				},
			},
			Metadata: metaMap,
		}, nil
	}

	return &types.ConstructionParseResponse{
		Operations:               ops,
		AccountIdentifierSigners: []*types.AccountIdentifier{},
		Metadata:                 metaMap,
	}, nil
}

func createTransferOps(tx transaction) ([]*types.Operation, *string, *types.Error) {
	var (
		opMethod     string
		value        *big.Int
		toAddressHex string
	)

	// Erc20 transfer
	if len(tx.Data) != 0 {
		toAddress, amountSent, err := parseErc20TransferData(tx.Data)
		if err != nil {
			return nil, nil, WrapError(ErrInvalidInput, err)
		}

		value = amountSent
		opMethod = mapper.OpErc20Transfer
		toAddressHex = toAddress.Hex()
	} else {
		value = tx.Value
		opMethod = mapper.OpCall
		toAddressHex = tx.To
	}

	// Ensure valid from address
	checkFrom, ok := ChecksumAddress(tx.From)
	if !ok {
		return nil, nil, WrapError(
			ErrInvalidInput,
			fmt.Errorf("%s is not a valid address", tx.From),
		)
	}

	// Ensure valid to address
	checkTo, ok := ChecksumAddress(toAddressHex)
	if !ok {
		return nil, nil, WrapError(ErrInvalidInput, fmt.Errorf("%s is not a valid address", tx.To))
	}

	ops := []*types.Operation{
		{
			Type: opMethod,
			OperationIdentifier: &types.OperationIdentifier{
				Index: 0,
			},
			Account: &types.AccountIdentifier{
				Address: checkFrom,
			},
			Amount: &types.Amount{
				Value:    new(big.Int).Neg(value).String(),
				Currency: tx.Currency,
			},
		},
		{
			Type: opMethod,
			OperationIdentifier: &types.OperationIdentifier{
				Index: 1,
			},
			RelatedOperations: []*types.OperationIdentifier{
				{
					Index: 0,
				},
			},
			Account: &types.AccountIdentifier{
				Address: checkTo,
			},
			Amount: &types.Amount{
				Value:    value.String(),
				Currency: tx.Currency,
			},
		},
	}
	return ops, &checkFrom, nil
}

func createUnwrapOps(tx transaction) ([]*types.Operation, *string, *types.Error) {
	amount, _, err := parseUnwrapData(tx.Data)
	if err != nil {
		return nil, nil, WrapError(ErrInvalidInput, err)
	}

	// Ensure valid from address
	checkFrom, ok := ChecksumAddress(tx.From)
	if !ok {
		return nil, nil, WrapError(
			ErrInvalidInput,
			fmt.Errorf("%s is not a valid address", tx.From),
		)
	}

	ops := []*types.Operation{
		{
			Type: mapper.OpErc20Burn,
			OperationIdentifier: &types.OperationIdentifier{
				Index: 0,
			},
			Account: &types.AccountIdentifier{
				Address: checkFrom,
			},
			Amount: &types.Amount{
				Value:    new(big.Int).Neg(amount).String(),
				Currency: tx.Currency,
			},
		},
	}
	return ops, &checkFrom, nil
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
func (s ConstructionService) ConstructionPayloads(
	ctx context.Context,
	req *types.ConstructionPayloadsRequest,
) (*types.ConstructionPayloadsResponse, *types.Error) {
	if s.pChainBackend.ShouldHandleRequest(req) {
		return s.pChainBackend.ConstructionPayloads(ctx, req)
	}

	if s.cChainAtomicTxBackend.ShouldHandleRequest(req) {
		return s.cChainAtomicTxBackend.ConstructionPayloads(ctx, req)
	}

	var (
		tx         *ethtypes.Transaction
		unsignedTx *transaction
		checkFrom  *string
		wrappedErr *types.Error
	)

	if isUnwrapRequest(req.Metadata) {
		tx, unsignedTx, checkFrom, wrappedErr = s.createUnwrapPayload(req)
	} else {
		tx, unsignedTx, checkFrom, wrappedErr = s.createTransferPayload(req)
	}
	if wrappedErr != nil {
		return nil, wrappedErr
	}

	// Construct SigningPayload
	signer := ethtypes.LatestSignerForChainID(s.config.ChainID)

	payload := &types.SigningPayload{
		AccountIdentifier: &types.AccountIdentifier{Address: *checkFrom},
		Bytes:             signer.Hash(tx).Bytes(),
		SignatureType:     types.EcdsaRecovery,
	}

	unsignedTxJSON, err := json.Marshal(unsignedTx)
	if err != nil {
		return nil, WrapError(ErrInternalError, err)
	}

	return &types.ConstructionPayloadsResponse{
		UnsignedTransaction: string(unsignedTxJSON),
		Payloads:            []*types.SigningPayload{payload},
	}, nil
}

func (s ConstructionService) createTransferPayload(
	req *types.ConstructionPayloadsRequest,
) (*ethtypes.Transaction, *transaction, *string, *types.Error) {
	operationDescriptions, err := s.CreateTransferOperationDescription(req.Operations)
	if err != nil {
		return nil, nil, nil, WrapError(ErrInvalidInput, err.Error())
	}

	descriptions := &parser.Descriptions{
		OperationDescriptions: operationDescriptions,
		ErrUnmatched:          true,
	}

	matches, err := parser.MatchOperations(descriptions, req.Operations)
	if err != nil {
		return nil, nil, nil, WrapError(ErrInvalidInput, "unclear intent")
	}

	toOp, amount := matches[1].First()
	toAddress := toOp.Account.Address

	fromOp, _ := matches[0].First()
	fromAddress := fromOp.Account.Address
	fromCurrency := fromOp.Amount.Currency

	checkFrom, ok := ChecksumAddress(fromAddress)
	if !ok {
		return nil, nil, nil, WrapError(
			ErrInvalidInput,
			fmt.Errorf("%s is not a valid address", fromAddress),
		)
	}

	checkTo, ok := ChecksumAddress(toAddress)
	if !ok {
		return nil, nil, nil, WrapError(
			ErrInvalidInput,
			fmt.Errorf("%s is not a valid address", toAddress),
		)
	}
	var transferData []byte
	var sendToAddress ethcommon.Address
	if types.Hash(fromCurrency) == types.Hash(mapper.AvaxCurrency) {
		transferData = []byte{}
		sendToAddress = ethcommon.HexToAddress(checkTo)
	} else {
		contract, ok := fromCurrency.Metadata[mapper.ContractAddressMetadata].(string)
		if !ok {
			return nil, nil, nil, WrapError(ErrInvalidInput,
				fmt.Errorf("%s currency doesn't have a contract address in metadata", fromCurrency.Symbol))
		}

		transferData = generateErc20TransferData(toAddress, amount)
		sendToAddress = ethcommon.HexToAddress(contract)
		amount = big.NewInt(0)
	}

	var metadata metadata
	if err := mapper.UnmarshalJSONMap(req.Metadata, &metadata); err != nil {
		return nil, nil, nil, WrapError(ErrInvalidInput, err)
	}

	nonce := metadata.Nonce
	gasPrice := metadata.GasPrice
	gasLimit := metadata.GasLimit
	chainID := s.config.ChainID

	tx := ethtypes.NewTransaction(
		nonce,
		sendToAddress,
		amount,
		gasLimit,
		gasPrice,
		transferData,
	)

	unsignedTx := &transaction{
		From:     checkFrom,
		To:       sendToAddress.Hex(),
		Value:    amount,
		Data:     tx.Data(),
		Nonce:    tx.Nonce(),
		GasPrice: gasPrice,
		GasLimit: tx.Gas(),
		ChainID:  chainID,
		Currency: fromCurrency,
	}
	return tx, unsignedTx, &checkFrom, nil
}

func (s ConstructionService) createUnwrapPayload(
	req *types.ConstructionPayloadsRequest,
) (*ethtypes.Transaction, *transaction, *string, *types.Error) {
	operationDescriptions, err := s.CreateUnwrapOperationDescription(req.Operations)
	if err != nil {
		return nil, nil, nil, WrapError(ErrInvalidInput, err.Error())
	}

	descriptions := &parser.Descriptions{
		OperationDescriptions: operationDescriptions,
		ErrUnmatched:          true,
	}

	matches, err := parser.MatchOperations(descriptions, req.Operations)
	if err != nil {
		return nil, nil, nil, WrapError(ErrInvalidInput, "unclear intent")
	}

	fromOp, amount := matches[0].First()
	fromAddress := fromOp.Account.Address
	fromCurrency := fromOp.Amount.Currency

	// op match will return a negative amount since it's from a balance losing funds
	amount = new(big.Int).Neg(amount)

	checkFrom, ok := ChecksumAddress(fromAddress)
	if !ok {
		return nil, nil, nil, WrapError(
			ErrInvalidInput,
			fmt.Errorf("%s is not a valid address", fromAddress),
		)
	}

	contract, ok := fromCurrency.Metadata[mapper.ContractAddressMetadata].(string)
	if !ok {
		return nil, nil, nil, WrapError(
			ErrInvalidInput,
			fmt.Errorf(
				"%s currency doesn't have a contract address in metadata",
				fromCurrency.Symbol,
			),
		)
	}

	if !mapper.EqualFoldContains(s.config.BridgeTokenList, contract) {
		return nil, nil, nil, WrapError(
			ErrInvalidInput,
			fmt.Errorf(
				"%s contract address not in configured list of supported bridge tokens",
				contract,
			),
		)
	}

	unwrapData := generateBridgeUnwrapTransferData(amount, big.NewInt(defaultUnwrapChainID))
	sendToAddress := ethcommon.HexToAddress(contract)

	var metadata metadata
	if err := mapper.UnmarshalJSONMap(req.Metadata, &metadata); err != nil {
		return nil, nil, nil, WrapError(ErrInvalidInput, err)
	}

	nonce := metadata.Nonce
	gasPrice := metadata.GasPrice
	gasLimit := metadata.GasLimit
	chainID := s.config.ChainID

	// amount refers to native currency being transferred, which should be zero in the case of an unwrap
	amount = big.NewInt(0)
	tx := ethtypes.NewTransaction(
		nonce,
		sendToAddress,
		amount,
		gasLimit,
		gasPrice,
		unwrapData,
	)

	unsignedTx := &transaction{
		From:     checkFrom,
		To:       sendToAddress.Hex(),
		Value:    amount,
		Data:     tx.Data(),
		Nonce:    tx.Nonce(),
		GasPrice: gasPrice,
		GasLimit: tx.Gas(),
		ChainID:  chainID,
		Currency: fromCurrency,
	}
	return tx, unsignedTx, &checkFrom, nil
}

// ConstructionPreprocess implements /construction/preprocess endpoint.
//
// Preprocess is called prior to /construction/payloads to construct a request for
// any metadata that is needed for transaction construction given (i.e. account nonce).
func (s ConstructionService) ConstructionPreprocess(
	ctx context.Context,
	req *types.ConstructionPreprocessRequest,
) (*types.ConstructionPreprocessResponse, *types.Error) {
	if s.pChainBackend.ShouldHandleRequest(req) {
		return s.pChainBackend.ConstructionPreprocess(ctx, req)
	}
	if s.cChainAtomicTxBackend.ShouldHandleRequest(req) {
		return s.cChainAtomicTxBackend.ConstructionPreprocess(ctx, req)
	}

	var (
		operationDescriptions []*parser.OperationDescription
		preprocessOptions     *options
		err                   error
		typesError            *types.Error
	)

	if isUnwrapRequest(req.Metadata) {
		operationDescriptions, err = s.CreateUnwrapOperationDescription(req.Operations)
		if err != nil {
			return nil, WrapError(ErrInvalidInput, err.Error())
		}
		preprocessOptions, typesError = s.createUnwrapPreprocessOptions(operationDescriptions, req)
		if typesError != nil {
			return nil, typesError
		}
	} else {
		operationDescriptions, err = s.CreateTransferOperationDescription(req.Operations)
		if err != nil {
			return nil, WrapError(ErrInvalidInput, err.Error())
		}
		preprocessOptions, typesError = s.createTransferPreprocessOptions(operationDescriptions, req)
		if typesError != nil {
			return nil, typesError
		}
	}

	if v, ok := req.Metadata["gas_price"]; ok {
		stringObj, ok := v.(string)
		if !ok {
			return nil, WrapError(
				ErrInvalidInput,
				fmt.Errorf("%s is not a valid gas price string", v),
			)
		}
		bigObj, ok := new(big.Int).SetString(stringObj, 10)
		if !ok {
			return nil, WrapError(ErrInvalidInput, fmt.Errorf("%s is not a valid gas price", v))
		}
		preprocessOptions.GasPrice = bigObj
	}
	if v, ok := req.Metadata["gas_limit"]; ok {
		stringObj, ok := v.(string)
		if !ok {
			return nil, WrapError(
				ErrInvalidInput,
				fmt.Errorf("%s is not a valid gas limit string", v),
			)
		}
		bigObj, ok := new(big.Int).SetString(stringObj, 10)
		if !ok {
			return nil, WrapError(ErrInvalidInput, fmt.Errorf("%s is not a valid gas limit", v))
		}
		preprocessOptions.GasLimit = bigObj
	}
	if v, ok := req.Metadata["nonce"]; ok {
		stringObj, ok := v.(string)
		if !ok {
			return nil, WrapError(ErrInvalidInput, fmt.Errorf("%s is not a valid nonce string", v))
		}
		bigObj, ok := new(big.Int).SetString(stringObj, 10)
		if !ok {
			return nil, WrapError(ErrInvalidInput, fmt.Errorf("%s is not a valid nonce", v))
		}
		preprocessOptions.Nonce = bigObj
	}

	marshaled, err := mapper.MarshalJSONMap(preprocessOptions)
	if err != nil {
		return nil, WrapError(ErrInternalError, err)
	}

	return &types.ConstructionPreprocessResponse{
		Options: marshaled,
	}, nil
}

// ConstructionSubmit implements /construction/submit endpoint.
//
// Submit a pre-signed transaction to the node.
func (s ConstructionService) ConstructionSubmit(
	ctx context.Context,
	req *types.ConstructionSubmitRequest,
) (*types.TransactionIdentifierResponse, *types.Error) {
	if s.config.IsOfflineMode() {
		return nil, ErrUnavailableOffline
	}

	if len(req.SignedTransaction) == 0 {
		return nil, WrapError(ErrInvalidInput, "signed transaction value is not provided")
	}

	if s.pChainBackend.ShouldHandleRequest(req) {
		return s.pChainBackend.ConstructionSubmit(ctx, req)
	}

	if s.cChainAtomicTxBackend.ShouldHandleRequest(req) {
		return s.cChainAtomicTxBackend.ConstructionSubmit(ctx, req)
	}

	var wrappedTx signedTransactionWrapper
	if err := json.Unmarshal([]byte(req.SignedTransaction), &wrappedTx); err != nil {
		return nil, WrapError(ErrInvalidInput, err)
	}

	var signedTx ethtypes.Transaction
	if err := signedTx.UnmarshalJSON(wrappedTx.SignedTransaction); err != nil {
		return nil, WrapError(ErrInvalidInput, err)
	}

	if err := s.client.SendTransaction(ctx, &signedTx); err != nil {
		return nil, WrapError(ErrClientError, err)
	}

	return &types.TransactionIdentifierResponse{
		TransactionIdentifier: &types.TransactionIdentifier{
			Hash: signedTx.Hash().String(),
		},
	}, nil
}

func (s ConstructionService) CreateTransferOperationDescription(
	operations []*types.Operation,
) ([]*parser.OperationDescription, error) {
	if len(operations) != 2 {
		return nil, fmt.Errorf("invalid number of operations")
	}

	firstCurrency := operations[0].Amount.Currency
	secondCurrency := operations[1].Amount.Currency

	if firstCurrency == nil || secondCurrency == nil {
		return nil, fmt.Errorf("invalid currency on operation")
	}

	if types.Hash(firstCurrency) != types.Hash(secondCurrency) {
		return nil, fmt.Errorf("currency info doesn't match between the operations")
	}

	if types.Hash(firstCurrency) == types.Hash(mapper.AvaxCurrency) {
		return s.createOperationDescriptionTransfer(mapper.AvaxCurrency, mapper.OpCall), nil
	}

	// Not Native Avax, we require contractInfo in metadata.
	if _, ok := firstCurrency.Metadata[mapper.ContractAddressMetadata].(string); !ok {
		return nil, fmt.Errorf("non-native currency must have contractAddress in metadata")
	}

	return s.createOperationDescriptionTransfer(firstCurrency, mapper.OpErc20Transfer), nil
}

func (s ConstructionService) CreateUnwrapOperationDescription(
	operations []*types.Operation,
) ([]*parser.OperationDescription, error) {
	if len(operations) != 1 {
		return nil, fmt.Errorf("invalid number of operations")
	}

	firstCurrency := operations[0].Amount.Currency

	if types.Hash(firstCurrency) == types.Hash(mapper.AvaxCurrency) {
		return nil, fmt.Errorf("cannot unwrap native avax")
	}
	tokenAddress, firstOk := firstCurrency.Metadata[mapper.ContractAddressMetadata].(string)

	// Not Native Avax, we require contractInfo in metadata
	if !firstOk {
		return nil, fmt.Errorf("non-native currency must have contractAddress in metadata")
	}

	if !mapper.EqualFoldContains(s.config.BridgeTokenList, tokenAddress) {
		return nil, fmt.Errorf("only configured bridge tokens may use try to use unwrap function")
	}

	return s.createOperationDescriptionBridgeUnwrap(firstCurrency), nil
}

func (s ConstructionService) createTransferPreprocessOptions(
	operationDescriptions []*parser.OperationDescription,
	req *types.ConstructionPreprocessRequest,
) (*options, *types.Error) {
	descriptions := &parser.Descriptions{
		OperationDescriptions: operationDescriptions,
		ErrUnmatched:          true,
	}
	matches, err := parser.MatchOperations(descriptions, req.Operations)
	if err != nil {
		return nil, WrapError(ErrInvalidInput, "unclear intent")
	}

	fromOp, _ := matches[0].First()
	fromAddress := fromOp.Account.Address
	toOp, amount := matches[1].First()
	toAddress := toOp.Account.Address

	fromCurrency := fromOp.Amount.Currency

	checkFrom, ok := ChecksumAddress(fromAddress)
	if !ok {
		return nil, WrapError(ErrInvalidInput, fmt.Errorf("%s is not a valid address", fromAddress))
	}
	checkTo, ok := ChecksumAddress(toAddress)
	if !ok {
		return nil, WrapError(ErrInvalidInput, fmt.Errorf("%s is not a valid address", toAddress))
	}

	return &options{
		From:                   checkFrom,
		To:                     checkTo,
		Value:                  amount,
		SuggestedFeeMultiplier: req.SuggestedFeeMultiplier,
		Currency:               fromCurrency,
	}, nil
}

func (s ConstructionService) createUnwrapPreprocessOptions(
	operationDescriptions []*parser.OperationDescription,
	req *types.ConstructionPreprocessRequest,
) (*options, *types.Error) {
	descriptions := &parser.Descriptions{
		OperationDescriptions: operationDescriptions,
		ErrUnmatched:          true,
	}
	matches, err := parser.MatchOperations(descriptions, req.Operations)
	if err != nil {
		return nil, WrapError(ErrInvalidInput, "unclear intent")
	}

	fromOp, amount := matches[0].First()
	fromAddress := fromOp.Account.Address

	// op match will return a negative amount since it's from a balance losing funds
	amount = new(big.Int).Neg(amount)

	fromCurrency := fromOp.Amount.Currency

	checkFrom, ok := ChecksumAddress(fromAddress)
	if !ok {
		return nil, WrapError(ErrInvalidInput, fmt.Errorf("%s is not a valid address", fromAddress))
	}

	metadata := &metadataOptions{
		UnwrapBridgeTx: true,
	}

	return &options{
		From:                   checkFrom,
		Value:                  amount,
		SuggestedFeeMultiplier: req.SuggestedFeeMultiplier,
		Currency:               fromCurrency,
		Metadata:               metadata,
	}, nil
}

func (s ConstructionService) createOperationDescriptionTransfer(
	currency *types.Currency,
	opCode string,
) []*parser.OperationDescription {
	return []*parser.OperationDescription{
		{
			Type: opCode,
			Account: &parser.AccountDescription{
				Exists: true,
			},
			Amount: &parser.AmountDescription{
				Exists:   true,
				Sign:     parser.NegativeAmountSign,
				Currency: currency,
			},
		},
		{
			Type: opCode,
			Account: &parser.AccountDescription{
				Exists: true,
			},
			Amount: &parser.AmountDescription{
				Exists:   true,
				Sign:     parser.PositiveAmountSign,
				Currency: currency,
			},
		},
	}
}

func (s ConstructionService) createOperationDescriptionBridgeUnwrap(
	currency *types.Currency,
) []*parser.OperationDescription {
	return []*parser.OperationDescription{
		{
			Type: mapper.OpErc20Burn,
			Account: &parser.AccountDescription{
				Exists: true,
			},
			Amount: &parser.AmountDescription{
				Exists:   true,
				Sign:     parser.NegativeAmountSign,
				Currency: currency,
			},
		},
	}
}

func (s ConstructionService) getNativeTransferGasLimit(
	ctx context.Context, toAddress string,
	fromAddress string, value *big.Int,
) (uint64, error) {
	if len(toAddress) == 0 || value == nil {
		// We guard against malformed inputs that may have been generated using
		// a previous version of avalanche-rosetta.
		return nativeTransferGasLimit, nil
	}
	to := ethcommon.HexToAddress(toAddress)
	gasLimit, err := s.client.EstimateGas(ctx, interfaces.CallMsg{
		From:  ethcommon.HexToAddress(fromAddress),
		To:    &to,
		Value: value,
	})
	if err != nil {
		return 0, err
	}
	return gasLimit, nil
}

func (s ConstructionService) getErc20TransferGasLimit(
	ctx context.Context, toAddress string,
	fromAddress string, value *big.Int, currency *types.Currency,
) (uint64, error) {
	contract, ok := currency.Metadata[mapper.ContractAddressMetadata]
	if len(toAddress) == 0 || value == nil || !ok {
		return erc20TransferGasLimit, nil
	}
	// ToAddress for erc20 transfers is the contract address
	contractAddress := ethcommon.HexToAddress(contract.(string))
	data := generateErc20TransferData(toAddress, value)
	gasLimit, err := s.client.EstimateGas(ctx, interfaces.CallMsg{
		From: ethcommon.HexToAddress(fromAddress),
		To:   &contractAddress,
		Data: data,
	})
	if err != nil {
		return 0, err
	}
	return gasLimit, nil
}

func (s ConstructionService) getBridgeUnwrapTransferGasLimit(
	ctx context.Context,
	fromAddress string,
	value *big.Int,
	currency *types.Currency,
) (uint64, error) {
	contract, ok := currency.Metadata[mapper.ContractAddressMetadata]
	if len(fromAddress) == 0 || value == nil || !ok {
		return unwrapGasLimit, nil
	}
	// ToAddress for bridge unwrap is the contract address
	contractAddress := ethcommon.HexToAddress(contract.(string))
	chainID := big.NewInt(defaultUnwrapChainID)
	data := generateBridgeUnwrapTransferData(value, chainID)

	gasLimit, err := s.client.EstimateGas(ctx, interfaces.CallMsg{
		From: ethcommon.HexToAddress(fromAddress),
		To:   &contractAddress,
		Data: data,
	})
	if err != nil {
		return 0, err
	}
	return gasLimit, nil
}

func generateErc20TransferData(toAddress string, value *big.Int) []byte {
	to := ethcommon.HexToAddress(toAddress)
	methodID := getMethodID(transferFnSignature)

	paddedAddress := ethcommon.LeftPadBytes(to.Bytes(), requiredPaddingBytes)
	paddedAmount := ethcommon.LeftPadBytes(value.Bytes(), requiredPaddingBytes)

	var data []byte
	data = append(data, methodID...)
	data = append(data, paddedAddress...)
	data = append(data, paddedAmount...)
	return data
}

func generateBridgeUnwrapTransferData(value *big.Int, chainID *big.Int) []byte {
	methodID := getMethodID(unwrapFnSignature)

	paddedAmount := ethcommon.LeftPadBytes(value.Bytes(), requiredPaddingBytes)
	paddedChainID := ethcommon.LeftPadBytes(chainID.Bytes(), requiredPaddingBytes)

	var data []byte
	data = append(data, methodID...)
	data = append(data, paddedAmount...)
	data = append(data, paddedChainID...)
	return data
}

func parseErc20TransferData(data []byte) (*ethcommon.Address, *big.Int, error) {
	if len(data) != genericTransferBytesLength {
		return nil, nil, fmt.Errorf("incorrect length for data array")
	}
	if hexutil.Encode(data[:4]) != transferMethodID {
		return nil, nil, fmt.Errorf("incorrect methodID signature")
	}

	address := ethcommon.BytesToAddress(data[5:36])
	amount := new(big.Int).SetBytes(data[37:])
	return &address, amount, nil
}

func parseUnwrapData(data []byte) (*big.Int, *big.Int, error) {
	if len(data) != genericUnwrapBytesLength {
		return nil, nil, fmt.Errorf("incorrect length for data array")
	}
	if hexutil.Encode(data[:4]) != unwrapMethodID {
		return nil, nil, fmt.Errorf("incorrect methodID signature")
	}

	amount := new(big.Int).SetBytes(data[5:36])
	chainID := new(big.Int).SetBytes(data[37:])

	if chainID.Uint64() != 0 {
		return nil, nil, fmt.Errorf("incorrect chainId value")
	}

	return amount, chainID, nil
}

func isUnwrapRequest(metadata map[string]interface{}) bool {
	if isUnwrap, ok := metadata["bridge_unwrap"]; ok {
		return isUnwrap.(bool)
	}
	return false
}

func getMethodID(signature string) []byte {
	transferSignature := []byte(signature)
	hash := sha3.NewLegacyKeccak256()
	hash.Write(transferSignature)
	methodID := hash.Sum(nil)[:4]
	return methodID
}

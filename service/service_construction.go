package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ava-labs/avalanchego/utils/crypto"
	"github.com/ava-labs/avalanchego/utils/formatting/address"
	ethtypes "github.com/ava-labs/coreth/core/types"
	"github.com/ava-labs/coreth/interfaces"
	"github.com/coinbase/rosetta-sdk-go/parser"
	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/coinbase/rosetta-sdk-go/utils"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/sha3"

	"github.com/ava-labs/avalanche-rosetta/client"
	"github.com/ava-labs/avalanche-rosetta/mapper"
)

const (
	padLength = 32

	transferFnSignature = "transfer(address,uint256)" // do not include spaces in the string
	transferDataLength  = 68                          // 4 (method id) + 2*32 (args)
)

// ConstructionService implements /construction/* endpoints
type ConstructionService struct {
	config *Config
	client client.Client
}

// NewConstructionService returns a new construction servicer
func NewConstructionService(config *Config, client client.Client) server.ConstructionAPIServicer {
	return &ConstructionService{
		config: config,
		client: client,
	}
}

// ConstructionMetadata implements /construction/metadata endpoint.
//
// Get any information required to construct a transaction for a specific network.
// Metadata returned here could be a recent hash to use, an account sequence number,
// or even arbitrary chain state. The request used when calling this endpoint
// is created by calling /construction/preprocess in an offline environment.
//
func (s ConstructionService) ConstructionMetadata(
	ctx context.Context,
	req *types.ConstructionMetadataRequest,
) (*types.ConstructionMetadataResponse, *types.Error) {
	if s.config.IsOfflineMode() {
		return nil, errUnavailableOffline
	}

	var input options
	if err := unmarshalJSONMap(req.Options, &input); err != nil {
		return nil, wrapError(errInvalidInput, err)
	}

	if len(input.From) == 0 {
		return nil, wrapError(errInvalidInput, "from address is not provided")
	}

	var nonce uint64
	var err error
	if input.Nonce == nil {
		nonce, err = s.client.NonceAt(ctx, ethcommon.HexToAddress(input.From), nil)
		if err != nil {
			return nil, wrapError(errClientError, err)
		}
	} else {
		nonce = input.Nonce.Uint64()
	}

	var gasPrice *big.Int
	if input.GasPrice == nil {
		if gasPrice, err = s.client.SuggestGasPrice(ctx); err != nil {
			return nil, wrapError(errClientError, err)
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
		if input.Currency == nil || utils.Equal(input.Currency, mapper.AvaxCurrency) {
			gasLimit, err = s.getNativeTransferGasLimit(ctx, input.To, input.From, input.Value)
		} else {
			gasLimit, err = s.getErc20TransferGasLimit(ctx, input.To, input.From, input.Value, input.Currency)
		}

		if err != nil {
			return nil, wrapError(errClientError, err)
		}
	} else {
		gasLimit = input.GasLimit.Uint64()
	}

	metadata := &metadata{
		Nonce:    nonce,
		GasPrice: gasPrice,
		GasLimit: gasLimit,
	}

	metadataMap, err := marshalJSONMap(metadata)
	if err != nil {
		return nil, wrapError(errInternalError, err)
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
//
func (s ConstructionService) ConstructionHash(
	ctx context.Context,
	req *types.ConstructionHashRequest,
) (*types.TransactionIdentifierResponse, *types.Error) {
	if len(req.SignedTransaction) == 0 {
		return nil, wrapError(errInvalidInput, "signed transaction value is not provided")
	}

	var wrappedTx signedTransactionWrapper
	if err := json.Unmarshal([]byte(req.SignedTransaction), &wrappedTx); err != nil {
		return nil, wrapError(errInvalidInput, err)
	}

	var signedTx ethtypes.Transaction
	if err := signedTx.UnmarshalJSON(wrappedTx.SignedTransaction); err != nil {
		return nil, wrapError(errInvalidInput, err)
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
//
func (s ConstructionService) ConstructionCombine(
	ctx context.Context,
	req *types.ConstructionCombineRequest,
) (*types.ConstructionCombineResponse, *types.Error) {
	if len(req.UnsignedTransaction) == 0 {
		return nil, wrapError(errInvalidInput, "transaction data is not provided")
	}
	if len(req.Signatures) == 0 {
		return nil, wrapError(errInvalidInput, "signature is not provided")
	}

	var unsignedTx transaction
	if err := json.Unmarshal([]byte(req.UnsignedTransaction), &unsignedTx); err != nil {
		return nil, wrapError(errInvalidInput, err)
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
		return nil, wrapError(errInvalidInput, err)
	}

	signedTxJSON, err := signedTx.MarshalJSON()
	if err != nil {
		return nil, wrapError(errInternalError, err)
	}

	wrappedSignedTx := signedTransactionWrapper{SignedTransaction: signedTxJSON, Currency: unsignedTx.Currency}

	wrappedSignedTxJSON, err := json.Marshal(wrappedSignedTx)
	if err != nil {
		return nil, wrapError(errInternalError, err)
	}

	return &types.ConstructionCombineResponse{
		SignedTransaction: string(wrappedSignedTxJSON),
	}, nil
}

// ConstructionDerive implements /construction/derive endpoint.
//
// Derive returns the AccountIdentifier associated with a public key. Blockchains
// that require an on-chain action to create an account should not implement this method.
//
func (s ConstructionService) ConstructionDerive(
	ctx context.Context,
	req *types.ConstructionDeriveRequest,
) (*types.ConstructionDeriveResponse, *types.Error) {
	if isPChain(req.NetworkIdentifier) {
		fac := crypto.FactorySECP256K1R{}
		pub, err := fac.ToPublicKey(req.PublicKey.Bytes)
		if err != nil {
			return nil, wrapError(errInvalidInput, err)
		}

		chainIDAlias, hrp, getErr := getAliasAndHRP(req.NetworkIdentifier)
		if getErr != nil {
			return nil, getErr
		}

		addr, err := address.Format(chainIDAlias, hrp, pub.Address().Bytes())
		if err != nil {
			return nil, wrapError(errInvalidInput, err)
		}

		return &types.ConstructionDeriveResponse{
			AccountIdentifier: &types.AccountIdentifier{
				Address: addr,
			},
		}, nil
	}

	if req.PublicKey == nil {
		return nil, wrapError(errInvalidInput, "public key is not provided")
	}

	key, err := ethcrypto.DecompressPubkey(req.PublicKey.Bytes)
	if err != nil {
		return nil, wrapError(errInvalidInput, err)
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
func (s ConstructionService) ConstructionParse(
	ctx context.Context,
	req *types.ConstructionParseRequest,
) (*types.ConstructionParseResponse, *types.Error) {
	var tx transaction

	if !req.Signed {
		if err := json.Unmarshal([]byte(req.Transaction), &tx); err != nil {
			return nil, wrapError(errInvalidInput, err)
		}
	} else {
		var wrappedTx signedTransactionWrapper
		if err := json.Unmarshal([]byte(req.Transaction), &wrappedTx); err != nil {
			return nil, wrapError(errInvalidInput, err)
		}

		var t ethtypes.Transaction
		if err := t.UnmarshalJSON(wrappedTx.SignedTransaction); err != nil {
			return nil, wrapError(errInvalidInput, err)
		}

		tx.To = t.To().String()
		tx.Value = t.Value()
		tx.Data = t.Data()
		tx.Nonce = t.Nonce()
		tx.GasPrice = t.GasPrice()
		tx.GasLimit = t.Gas()
		tx.ChainID = s.config.ChainID
		tx.Currency = wrappedTx.Currency

		msg, err := t.AsMessage(s.config.Signer(), nil)
		if err != nil {
			return nil, wrapError(errInvalidInput, err)
		}
		tx.From = msg.From().Hex()
	}

	var opMethod string
	var value *big.Int
	var toAddressHex string
	// Erc20 transfer
	if len(tx.Data) != 0 {
		toAddress, amountSent, err := parseErc20TransferData(tx.Data)
		if err != nil {
			return nil, wrapError(errInvalidInput, err)
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
		return nil, wrapError(errInvalidInput, fmt.Errorf("%s is not a valid address", tx.From))
	}

	// Ensure valid to address
	checkTo, ok := ChecksumAddress(toAddressHex)
	if !ok {
		return nil, wrapError(errInvalidInput, fmt.Errorf("%s is not a valid address", tx.To))
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

	metadata := &parseMetadata{
		Nonce:    tx.Nonce,
		GasPrice: tx.GasPrice,
		GasLimit: tx.GasLimit,
		ChainID:  tx.ChainID,
	}
	metaMap, err := marshalJSONMap(metadata)
	if err != nil {
		return nil, wrapError(errInternalError, err)
	}

	if req.Signed {
		return &types.ConstructionParseResponse{
			Operations: ops,
			AccountIdentifierSigners: []*types.AccountIdentifier{
				{
					Address: checkFrom,
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
func (s ConstructionService) ConstructionPayloads(
	ctx context.Context,
	req *types.ConstructionPayloadsRequest,
) (*types.ConstructionPayloadsResponse, *types.Error) {
	operationDescriptions, err := s.CreateOperationDescription(req.Operations)
	if err != nil {
		return nil, wrapError(errInvalidInput, err)
	}

	descriptions := &parser.Descriptions{
		OperationDescriptions: operationDescriptions,
		ErrUnmatched:          true,
	}

	matches, err := parser.MatchOperations(descriptions, req.Operations)
	if err != nil {
		return nil, wrapError(errInvalidInput, "unclear intent")
	}

	var metadata metadata
	if err := unmarshalJSONMap(req.Metadata, &metadata); err != nil {
		return nil, wrapError(errInvalidInput, err)
	}

	toOp, amount := matches[1].First()
	toAddress := toOp.Account.Address
	nonce := metadata.Nonce
	gasPrice := metadata.GasPrice
	gasLimit := metadata.GasLimit
	chainID := s.config.ChainID

	fromOp, _ := matches[0].First()
	fromAddress := fromOp.Account.Address
	fromCurrency := fromOp.Amount.Currency

	checkFrom, ok := ChecksumAddress(fromAddress)
	if !ok {
		return nil, wrapError(errInvalidInput, fmt.Errorf("%s is not a valid address", fromAddress))
	}

	checkTo, ok := ChecksumAddress(toAddress)
	if !ok {
		return nil, wrapError(errInvalidInput, fmt.Errorf("%s is not a valid address", toAddress))
	}
	var transferData []byte
	var sendToAddress ethcommon.Address
	if utils.Equal(fromCurrency, mapper.AvaxCurrency) {
		transferData = []byte{}
		sendToAddress = ethcommon.HexToAddress(checkTo)
	} else {
		contract, ok := fromCurrency.Metadata[mapper.ContractAddressMetadata].(string)
		if !ok {
			return nil, wrapError(errInvalidInput,
				fmt.Errorf("%s currency doesn't have a contract address in metadata", fromCurrency.Symbol))
		}

		transferData = generateErc20TransferData(toAddress, amount)
		sendToAddress = ethcommon.HexToAddress(contract)
		amount = big.NewInt(0)
	}

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

	payload := &types.SigningPayload{
		AccountIdentifier: &types.AccountIdentifier{Address: checkFrom},
		Bytes:             s.config.Signer().Hash(tx).Bytes(),
		SignatureType:     types.EcdsaRecovery,
	}

	unsignedTxJSON, err := json.Marshal(unsignedTx)
	if err != nil {
		return nil, wrapError(errInternalError, err)
	}

	return &types.ConstructionPayloadsResponse{
		UnsignedTransaction: string(unsignedTxJSON),
		Payloads:            []*types.SigningPayload{payload},
	}, nil
}

// ConstructionPreprocess implements /construction/preprocess endpoint.
//
// Preprocess is called prior to /construction/payloads to construct a request for
// any metadata that is needed for transaction construction given (i.e. account nonce).
//
func (s ConstructionService) ConstructionPreprocess(
	ctx context.Context,
	req *types.ConstructionPreprocessRequest,
) (*types.ConstructionPreprocessResponse, *types.Error) {
	operationDescriptions, err := s.CreateOperationDescription(req.Operations)
	if err != nil {
		return nil, wrapError(errInvalidInput, err)
	}

	descriptions := &parser.Descriptions{
		OperationDescriptions: operationDescriptions,
		ErrUnmatched:          true,
	}

	matches, err := parser.MatchOperations(descriptions, req.Operations)
	if err != nil {
		return nil, wrapError(errInvalidInput, "unclear intent")
	}

	fromOp, _ := matches[0].First()
	fromAddress := fromOp.Account.Address
	toOp, amount := matches[1].First()
	toAddress := toOp.Account.Address

	fromCurrency := fromOp.Amount.Currency

	checkFrom, ok := ChecksumAddress(fromAddress)
	if !ok {
		return nil, wrapError(errInvalidInput, fmt.Errorf("%s is not a valid address", fromAddress))
	}
	checkTo, ok := ChecksumAddress(toAddress)
	if !ok {
		return nil, wrapError(errInvalidInput, fmt.Errorf("%s is not a valid address", toAddress))
	}

	preprocessOptions := &options{
		From:                   checkFrom,
		To:                     checkTo,
		Value:                  amount,
		SuggestedFeeMultiplier: req.SuggestedFeeMultiplier,
		Currency:               fromCurrency,
	}

	if v, ok := req.Metadata["gas_price"]; ok {
		stringObj, ok := v.(string)
		if !ok {
			return nil, wrapError(errInvalidInput, fmt.Errorf("%s is not a valid gas price string", v))
		}
		bigObj, ok := new(big.Int).SetString(stringObj, 10)
		if !ok {
			return nil, wrapError(errInvalidInput, fmt.Errorf("%s is not a valid gas price", v))
		}
		preprocessOptions.GasPrice = bigObj
	}
	if v, ok := req.Metadata["gas_limit"]; ok {
		stringObj, ok := v.(string)
		if !ok {
			return nil, wrapError(errInvalidInput, fmt.Errorf("%s is not a valid gas limit string", v))
		}
		bigObj, ok := new(big.Int).SetString(stringObj, 10)
		if !ok {
			return nil, wrapError(errInvalidInput, fmt.Errorf("%s is not a valid gas limit", v))
		}
		preprocessOptions.GasLimit = bigObj
	}
	if v, ok := req.Metadata["nonce"]; ok {
		stringObj, ok := v.(string)
		if !ok {
			return nil, wrapError(errInvalidInput, fmt.Errorf("%s is not a valid nonce string", v))
		}
		bigObj, ok := new(big.Int).SetString(stringObj, 10)
		if !ok {
			return nil, wrapError(errInvalidInput, fmt.Errorf("%s is not a valid nonce", v))
		}
		preprocessOptions.Nonce = bigObj
	}

	marshaled, err := marshalJSONMap(preprocessOptions)
	if err != nil {
		return nil, wrapError(errInternalError, err)
	}

	return &types.ConstructionPreprocessResponse{
		Options: marshaled,
	}, nil
}

// ConstructionSubmit implements /construction/submit endpoint.
//
// Submit a pre-signed transaction to the node.
//
func (s ConstructionService) ConstructionSubmit(
	ctx context.Context,
	req *types.ConstructionSubmitRequest,
) (*types.TransactionIdentifierResponse, *types.Error) {
	if s.config.IsOfflineMode() {
		return nil, errUnavailableOffline
	}

	if len(req.SignedTransaction) == 0 {
		return nil, wrapError(errInvalidInput, "signed transaction value is not provided")
	}

	var wrappedTx signedTransactionWrapper
	if err := json.Unmarshal([]byte(req.SignedTransaction), &wrappedTx); err != nil {
		return nil, wrapError(errInvalidInput, err)
	}

	var signedTx ethtypes.Transaction
	if err := signedTx.UnmarshalJSON(wrappedTx.SignedTransaction); err != nil {
		return nil, wrapError(errInvalidInput, err)
	}

	if err := s.client.SendTransaction(ctx, &signedTx); err != nil {
		return nil, wrapError(errClientError, err)
	}

	return &types.TransactionIdentifierResponse{
		TransactionIdentifier: &types.TransactionIdentifier{
			Hash: signedTx.Hash().String(),
		},
	}, nil
}

func (s ConstructionService) CreateOperationDescription(
	operations []*types.Operation,
) ([]*parser.OperationDescription, error) {
	if len(operations) != 2 {
		return nil, fmt.Errorf("invalid number of operations")
	}

	currency := operations[0].Amount.Currency

	if currency == nil || operations[1].Amount.Currency == nil {
		return nil, fmt.Errorf("invalid currency on operation")
	}

	if !utils.Equal(currency, operations[1].Amount.Currency) {
		return nil, fmt.Errorf("currency info doesn't match between the operations")
	}

	if utils.Equal(currency, mapper.AvaxCurrency) {
		return s.createOperationDescription(currency, mapper.OpCall), nil
	}

	// ERC-20s must have contract address in metadata
	if _, ok := currency.Metadata[mapper.ContractAddressMetadata].(string); !ok {
		return nil, fmt.Errorf("contractAddress must be populated in currency metadata")
	}

	return s.createOperationDescription(currency, mapper.OpErc20Transfer), nil
}

func (s ConstructionService) createOperationDescription(
	currency *types.Currency,
	opType string,
) []*parser.OperationDescription {
	return []*parser.OperationDescription{
		// Send
		{
			Type: opType,
			Account: &parser.AccountDescription{
				Exists: true,
			},
			Amount: &parser.AmountDescription{
				Exists:   true,
				Sign:     parser.NegativeAmountSign,
				Currency: currency,
			},
		},

		// Receive
		{
			Type: opType,
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

func (s ConstructionService) getNativeTransferGasLimit(
	ctx context.Context,
	to string,
	from string,
	value *big.Int,
) (uint64, error) {
	// Guard against malformed inputs that may have been generated using
	// a previous version of avalanche-rosetta.
	if len(to) == 0 || value == nil {
		return nativeTransferGasLimit, nil
	}

	toAddr := ethcommon.HexToAddress(to)
	return s.client.EstimateGas(ctx, interfaces.CallMsg{
		From:  ethcommon.HexToAddress(from),
		To:    &toAddr,
		Value: value,
	})
}

// Ref: https://goethereumbook.org/en/transfer-tokens/#set-gas-limit
func (s ConstructionService) getErc20TransferGasLimit(
	ctx context.Context,
	to string,
	from string,
	value *big.Int,
	currency *types.Currency,
) (uint64, error) {
	contract, ok := currency.Metadata[mapper.ContractAddressMetadata]
	if len(to) == 0 || value == nil || !ok {
		return erc20TransferGasLimit, nil
	}

	contractAddress := ethcommon.HexToAddress(contract.(string))
	return s.client.EstimateGas(ctx, interfaces.CallMsg{
		From: ethcommon.HexToAddress(from),
		To:   &contractAddress,
		Data: generateErc20TransferData(to, value),
	})
}

// Ref: https://goethereumbook.org/en/transfer-tokens/#forming-the-data-field
func generateErc20TransferData(to string, value *big.Int) []byte {
	toAddr := ethcommon.HexToAddress(to)
	methodID := getMethodID(transferFnSignature)

	paddedAddress := ethcommon.LeftPadBytes(toAddr.Bytes(), padLength)
	paddedAmount := ethcommon.LeftPadBytes(value.Bytes(), padLength)

	var data []byte
	data = append(data, methodID...)
	data = append(data, paddedAddress...)
	data = append(data, paddedAmount...)
	return data
}

// Ref: https://goethereumbook.org/en/transfer-tokens/#forming-the-data-field
func parseErc20TransferData(data []byte) (*ethcommon.Address, *big.Int, error) {
	if len(data) != transferDataLength {
		return nil, nil, fmt.Errorf("incorrect length for data array")
	}

	methodBytes, addrBytes, amtBytes := data[:4], data[5:36], data[37:]

	if hexutil.Encode(methodBytes) != hexutil.Encode(getMethodID(transferFnSignature)) {
		return nil, nil, fmt.Errorf("incorrect methodID signature")
	}

	addr := ethcommon.BytesToAddress(addrBytes)
	amt := new(big.Int).SetBytes(amtBytes)
	return &addr, amt, nil
}

// Ref: https://goethereumbook.org/en/transfer-tokens/#forming-the-data-field
func getMethodID(signature string) []byte {
	bytes := []byte(signature)
	hash := sha3.NewLegacyKeccak256()
	hash.Write(bytes)
	return hash.Sum(nil)[:4]
}

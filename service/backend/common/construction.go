package common

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/crypto/secp256k1"
	"github.com/ava-labs/avalanchego/utils/formatting/address"
	"github.com/ava-labs/avalanchego/utils/rpc"
	"github.com/ava-labs/avalanchego/vms/components/avax"
	"github.com/ava-labs/avalanchego/vms/components/verify"
	"github.com/ava-labs/avalanchego/vms/secp256k1fx"
	"github.com/coinbase/rosetta-sdk-go/parser"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ava-labs/avalanche-rosetta/constants"
	"github.com/ava-labs/avalanche-rosetta/mapper"
	pmapper "github.com/ava-labs/avalanche-rosetta/mapper/pchain"
	"github.com/ava-labs/avalanche-rosetta/service"
)

var (
	errNoOperationsToMatch      = errors.New("no operations were passed to match")
	errInvalidInput             = errors.New("invalid input")
	errInvalidInputSignatureLen = errors.New("input signature length doesn't match credentials needed")
	errInsufficientSignatures   = errors.New("insufficient signatures")
	errInvalidSignatureLen      = errors.New("invalid signature length")
)

// DeriveBech32Address derives Bech32 addresses for the given chain using public key and hrp provided in the request
func DeriveBech32Address(fac *secp256k1.Factory, chainIDAlias constants.ChainIDAlias, req *types.ConstructionDeriveRequest) (*types.ConstructionDeriveResponse, *types.Error) {
	pub, err := fac.ToPublicKey(req.PublicKey.Bytes)
	if err != nil {
		return nil, service.WrapError(service.ErrInvalidInput, err)
	}

	hrp, getErr := mapper.GetHRP(req.NetworkIdentifier)
	if getErr != nil {
		return nil, service.WrapError(service.ErrInvalidInput, err)
	}

	addr, err := address.Format(chainIDAlias.String(), hrp, pub.Address().Bytes())
	if err != nil {
		return nil, service.WrapError(service.ErrInvalidInput, err)
	}

	return &types.ConstructionDeriveResponse{
		AccountIdentifier: &types.AccountIdentifier{
			Address: addr,
		},
	}, nil
}

// MatchOperations defines the operation Rosetta parser matching rules and parses the input operations
//
// We require 2 types of operations; inputs with negative amounts and outputs with positive amounts
// parser guarantees there will be 2 matches.
func MatchOperations(operations []*types.Operation) ([]*parser.Match, error) {
	if len(operations) == 0 {
		return nil, errNoOperationsToMatch
	}
	opType := operations[0].Type

	var coinAction types.CoinAction
	var allowRepeatOutputs bool

	switch opType {
	case mapper.OpExport:
		coinAction = ""
		allowRepeatOutputs = false
	case mapper.OpImport:
		coinAction = types.CoinSpent
		allowRepeatOutputs = false
	default:
		coinAction = types.CoinSpent
		allowRepeatOutputs = true
	}

	descriptions := &parser.Descriptions{
		OperationDescriptions: []*parser.OperationDescription{
			{
				Type: opType,
				Account: &parser.AccountDescription{
					Exists: true,
				},
				Amount: &parser.AmountDescription{
					Exists: true,
					Sign:   parser.NegativeAmountSign,
				},
				AllowRepeats: true,
				CoinAction:   coinAction,
			},
			{
				Type: opType,
				Account: &parser.AccountDescription{
					Exists: true,
				},
				Amount: &parser.AmountDescription{
					Exists: true,
					Sign:   parser.PositiveAmountSign,
				},
				AllowRepeats: allowRepeatOutputs,
			},
		},
		ErrUnmatched: true,
	}

	return parser.MatchOperations(descriptions, operations)
}

// TxBuilder implements backend specific transaction construction logic
type TxBuilder interface {
	BuildTx(matches []*types.Operation, rawMetadata map[string]interface{}) (AvaxTx, []*types.AccountIdentifier, *types.Error)
}

// BuildPayloads performs transaction construction in /construction/payloads call and returns the unsigned transaction as well as the signing payloads.
// Chain specific logic is abstracted using the TxBuilder interface's BuildTx method.
func BuildPayloads(
	txBuilder TxBuilder,
	req *types.ConstructionPayloadsRequest,
) (*types.ConstructionPayloadsResponse, *types.Error) {
	tx, signers, tErr := txBuilder.BuildTx(req.Operations, req.Metadata)
	if tErr != nil {
		return nil, tErr
	}

	accountIdentifierSigners := make([]Signer, 0, len(req.Operations))
	for _, o := range req.Operations {
		// Skip positive amounts
		if o.Amount.Value[0] != '-' {
			continue
		}

		var coinIdentifier string

		if o.CoinChange != nil && o.CoinChange.CoinIdentifier != nil {
			coinIdentifier = o.CoinChange.CoinIdentifier.Identifier
		}

		accountIdentifierSigners = append(accountIdentifierSigners, Signer{
			CoinIdentifier:    coinIdentifier,
			AccountIdentifier: o.Account,
		})
	}

	rosettaTx := &RosettaTx{
		Tx:                       tx,
		AccountIdentifierSigners: accountIdentifierSigners,
	}

	payloads := make([]*types.SigningPayload, len(signers))
	for i, signer := range signers {
		payloads[i] = &types.SigningPayload{
			AccountIdentifier: signer,
			Bytes:             tx.SigningPayload(),
			SignatureType:     types.EcdsaRecovery,
		}
	}

	var metadata pmapper.Metadata
	err := mapper.UnmarshalJSONMap(req.Metadata, &metadata)
	if err != nil {
		return nil, service.WrapError(service.ErrInternalError, err)
	}

	if metadata.ExportMetadata != nil {
		rosettaTx.DestinationChain = metadata.DestinationChain
		rosettaTx.DestinationChainID = &metadata.DestinationChainID
	}

	txJSON, err := json.Marshal(rosettaTx)
	if err != nil {
		return nil, service.WrapError(service.ErrInternalError, err)
	}

	return &types.ConstructionPayloadsResponse{
		UnsignedTransaction: string(txJSON),
		Payloads:            payloads,
	}, nil
}

// TxParser implements backend specific transaction parsing logic
type TxParser interface {
	ParseTx(tx *RosettaTx, inputAddresses map[string]*types.AccountIdentifier) ([]*types.Operation, error)
}

// Parse contains transaction parsing logic for /construction/parse endpoint
// Chain specific logic is abstracted using TxParser interface's ParseTx method
func Parse(parser TxParser, payloadsTx *RosettaTx, isSigned bool) (*types.ConstructionParseResponse, *types.Error) {
	// Convert input tx into operations
	inputAddresses := getInputAddresses(payloadsTx)
	operations, err := parser.ParseTx(payloadsTx, inputAddresses)
	if err != nil {
		return nil, service.WrapError(service.ErrInvalidInput, "incorrect transaction input")
	}

	// Generate AccountIdentifierSigners if request is signed
	var signers []*types.AccountIdentifier
	if isSigned {
		payloadSigners, err := payloadsTx.GetAccountIdentifiers(operations)
		if err != nil {
			return nil, service.WrapError(service.ErrInvalidInput, err)
		}

		signers = payloadSigners
	}

	return &types.ConstructionParseResponse{
		Operations:               operations,
		AccountIdentifierSigners: signers,
	}, nil
}

func getInputAddresses(tx *RosettaTx) map[string]*types.AccountIdentifier {
	addresses := make(map[string]*types.AccountIdentifier)

	for _, signer := range tx.AccountIdentifierSigners {
		addresses[signer.CoinIdentifier] = signer.AccountIdentifier
	}

	return addresses
}

// TxCombiner implements backend specific transaction submission logic
type TxCombiner interface {
	CombineTx(tx AvaxTx, signatures []*types.Signature) (AvaxTx, *types.Error)
}

// Combine combines unsigned transactions with the provided signatures as part of /construction/combine call.
// Chain spacific logic is abstracted in TxCombiner interface's CombineTx method.
func Combine(
	combiner TxCombiner,
	rosettaTx *RosettaTx,
	signatures []*types.Signature,
) (*types.ConstructionCombineResponse, *types.Error) {
	combinedTx, tErr := combiner.CombineTx(rosettaTx.Tx, signatures)
	if tErr != nil {
		return nil, tErr
	}

	signedTransaction, err := json.Marshal(&RosettaTx{
		Tx:                       combinedTx,
		AccountIdentifierSigners: rosettaTx.AccountIdentifierSigners,
		DestinationChain:         rosettaTx.DestinationChain,
		DestinationChainID:       rosettaTx.DestinationChainID,
	})
	if err != nil {
		return nil, service.WrapError(service.ErrInternalError, "unable to encode signed transaction")
	}

	return &types.ConstructionCombineResponse{
		SignedTransaction: string(signedTransaction),
	}, nil
}

// BuildCredentialList builds a list of *secp256k1fx.Credentials using the given signatures
//
// Based on tx inputs, we can determine the number of signatures
// required by each input and put correct number of signatures to
// construct the signed tx.
//
// See https://github.com/ava-labs/avalanchego/blob/v1.9.0/vms/platformvm/txs/tx.go#L104
// for more details.
func BuildCredentialList(ins []*avax.TransferableInput, signatures []*types.Signature) ([]verify.Verifiable, error) {
	creds := make([]verify.Verifiable, len(ins))
	sigOffset := 0
	for i, transferInput := range ins {
		input, ok := transferInput.In.(*secp256k1fx.TransferInput)
		if !ok {
			return nil, errInvalidInput
		}

		cred, err := buildCredential(len(input.SigIndices), &sigOffset, signatures)
		if err != nil {
			return nil, err
		}

		creds[i] = cred
	}

	if sigOffset != len(signatures) {
		return nil, errInvalidInputSignatureLen
	}

	return creds, nil
}

// BuildSingletonCredentialList builds a list of a single *secp256k1fx.Credential using the given signatures
func BuildSingletonCredentialList(signatures []*types.Signature) ([]verify.Verifiable, error) {
	offset := 0
	cred, err := buildCredential(1, &offset, signatures)
	if err != nil {
		return nil, err
	}

	return []verify.Verifiable{cred}, nil
}

func buildCredential(numSigs int, sigOffset *int, signatures []*types.Signature) (*secp256k1fx.Credential, error) {
	cred := &secp256k1fx.Credential{}
	cred.Sigs = make([][secp256k1.SignatureLen]byte, numSigs)
	for j := 0; j < numSigs; j++ {
		if *sigOffset >= len(signatures) {
			return nil, errInsufficientSignatures
		}

		if len(signatures[*sigOffset].Bytes) != secp256k1.SignatureLen {
			return nil, errInvalidSignatureLen
		}
		copy(cred.Sigs[j][:], signatures[*sigOffset].Bytes)
		*sigOffset++
	}
	return cred, nil
}

// HashTx generates a transaction id for the given RosettaTx
func HashTx(rosettaTx *RosettaTx) (*types.TransactionIdentifierResponse, *types.Error) {
	txID := rosettaTx.Tx.Hash()
	hash := txID.String()

	return &types.TransactionIdentifierResponse{
		TransactionIdentifier: &types.TransactionIdentifier{
			Hash: hash,
		},
	}, nil
}

// TransactionIssuer implements chain specific transaction submission logic
type TransactionIssuer interface {
	IssueTx(ctx context.Context, txByte []byte, options ...rpc.Option) (ids.ID, error)
}

// SubmitTx broadcasts given Rosetta tx on chain and returns the transaction id.
// Chain specific logic is abstracted using the TransactionIssuer interface's IssueTx method.
func SubmitTx(
	ctx context.Context,
	issuer TransactionIssuer,
	rosettaTx *RosettaTx,
) (*types.TransactionIdentifierResponse, *types.Error) {
	bytes, err := rosettaTx.Tx.Marshal()
	if err != nil {
		return nil, service.WrapError(service.ErrInvalidInput, err)
	}

	txID, err := issuer.IssueTx(ctx, bytes)
	if err != nil {
		return nil, service.WrapError(service.ErrClientError, err)
	}

	return &types.TransactionIdentifierResponse{
		TransactionIdentifier: &types.TransactionIdentifier{
			Hash: txID.String(),
		},
	}, nil
}

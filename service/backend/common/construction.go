package common

import (
	"encoding/json"
	"errors"

	"github.com/ava-labs/avalanchego/utils/crypto"
	"github.com/ava-labs/avalanchego/utils/formatting/address"
	"github.com/coinbase/rosetta-sdk-go/parser"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ava-labs/avalanche-rosetta/mapper"
	pmapper "github.com/ava-labs/avalanche-rosetta/mapper/pchain"
	"github.com/ava-labs/avalanche-rosetta/service"
)

var errNoOperationsToMatch = errors.New("no operations were passed to match")

func DeriveBech32Address(fac *crypto.FactorySECP256K1R, chainIDAlias string, req *types.ConstructionDeriveRequest) (*types.ConstructionDeriveResponse, *types.Error) {
	pub, err := fac.ToPublicKey(req.PublicKey.Bytes)
	if err != nil {
		return nil, service.WrapError(service.ErrInvalidInput, err)
	}

	hrp, getErr := mapper.GetHRP(req.NetworkIdentifier)
	if getErr != nil {
		return nil, service.WrapError(service.ErrInvalidInput, err)
	}

	addr, err := address.Format(chainIDAlias, hrp, pub.Address().Bytes())
	if err != nil {
		return nil, service.WrapError(service.ErrInvalidInput, err)
	}

	return &types.ConstructionDeriveResponse{
		AccountIdentifier: &types.AccountIdentifier{
			Address: addr,
		},
	}, nil
}

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

type TxBuilder interface {
	BuildTx(matches []*types.Operation, rawMetadata map[string]interface{}) (AvaxTx, []*types.AccountIdentifier, *types.Error)
}

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

	hash, err := tx.SigningPayload()
	if err != nil {
		return nil, service.WrapError(service.ErrInvalidInput, err)
	}

	payloads := make([]*types.SigningPayload, len(signers))

	for i, signer := range signers {
		payloads[i] = &types.SigningPayload{
			AccountIdentifier: signer,
			Bytes:             hash,
			SignatureType:     types.EcdsaRecovery,
		}
	}

	var metadata pmapper.Metadata
	err = mapper.UnmarshalJSONMap(req.Metadata, &metadata)
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

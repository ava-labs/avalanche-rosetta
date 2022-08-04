package common

import (
	"errors"

	"github.com/ava-labs/avalanchego/utils/crypto"
	"github.com/ava-labs/avalanchego/utils/formatting/address"
	"github.com/coinbase/rosetta-sdk-go/parser"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ava-labs/avalanche-rosetta/mapper"
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

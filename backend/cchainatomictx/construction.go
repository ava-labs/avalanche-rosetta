package cchainatomictx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/formatting/address"
	"github.com/ava-labs/avalanchego/vms/components/verify"
	"github.com/ava-labs/coreth/plugin/evm"
	"github.com/coinbase/rosetta-sdk-go/parser"
	"github.com/coinbase/rosetta-sdk-go/types"
	ethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/ava-labs/avalanche-rosetta/backend"
	"github.com/ava-labs/avalanche-rosetta/backend/common"
	"github.com/ava-labs/avalanche-rosetta/constants"
	cconstants "github.com/ava-labs/avalanche-rosetta/constants/cchain"
	"github.com/ava-labs/avalanche-rosetta/mapper"
	cmapper "github.com/ava-labs/avalanche-rosetta/mapper/cchain"
	camapper "github.com/ava-labs/avalanche-rosetta/mapper/cchainatomictx"
)

var (
	errUnknownTxType = errors.New("unknown tx type")
	errUndecodableTx = errors.New("undecodable transaction")
)

// ConstructionDerive implements /construction/derive endpoint for C-chain atomic transactions
func (b *Backend) ConstructionDerive(ctx context.Context, req *types.ConstructionDeriveRequest) (*types.ConstructionDeriveResponse, *types.Error) {
	return common.DeriveBech32Address(b.fac, constants.CChain, req)
}

// ConstructionPreprocess implements /construction/preprocess endpoint for C-chain atomic transactions
func (b *Backend) ConstructionPreprocess(
	ctx context.Context,
	req *types.ConstructionPreprocessRequest,
) (*types.ConstructionPreprocessResponse, *types.Error) {
	matches, err := common.MatchOperations(req.Operations)
	if err != nil {
		return nil, backend.WrapError(backend.ErrInvalidInput, err)
	}

	firstIn, _ := matches[0].First()
	firstOut, _ := matches[1].First()

	if firstIn == nil || firstOut == nil {
		return nil, backend.WrapError(backend.ErrInvalidInput, "both input and output operations must be specified")
	}

	gasUsed, err := b.estimateGasUsed(firstIn.Type, matches)
	if err != nil {
		return nil, backend.WrapError(backend.ErrInvalidInput, err)
	}

	preprocessOptions := camapper.Options{
		AtomicTxGas: new(big.Int).SetUint64(gasUsed),
	}

	switch firstIn.Type {
	case cconstants.Import.String():
		v, ok := req.Metadata[camapper.MetadataSourceChain]
		if !ok {
			return nil, backend.WrapError(backend.ErrInvalidInput, "source_chain metadata must be provided")
		}
		chainAlias, ok := v.(string)
		if !ok {
			return nil, backend.WrapError(backend.ErrInvalidInput, "invalid source_chain value")
		}

		preprocessOptions.SourceChain = chainAlias
	case cconstants.Export.String():
		chain, _, _, err := address.Parse(firstOut.Account.Address)
		if err != nil {
			return nil, backend.WrapError(backend.ErrInternalError, err)
		}

		preprocessOptions.From = firstIn.Account.Address
		preprocessOptions.DestinationChain = chain

		if v, ok := req.Metadata[camapper.MetadataNonce]; ok {
			stringObj, ok := v.(string)
			if !ok {
				return nil, backend.WrapError(backend.ErrInvalidInput, fmt.Errorf("%s is not a valid nonce string", v))
			}
			bigObj, ok := new(big.Int).SetString(stringObj, 10)
			if !ok {
				return nil, backend.WrapError(backend.ErrInvalidInput, fmt.Errorf("%s is not a valid nonce", v))
			}
			preprocessOptions.Nonce = bigObj
		}
	}

	optionsMap, err := mapper.MarshalJSONMap(preprocessOptions)
	if err != nil {
		return nil, backend.WrapError(backend.ErrInternalError, err)
	}

	return &types.ConstructionPreprocessResponse{
		Options: optionsMap,
	}, nil
}

func (b *Backend) estimateGasUsed(opType string, matches []*parser.Match) (uint64, error) {
	// building tx with dummy data to get byte size for fee estimate
	tx, _, err := camapper.BuildTx(
		opType,
		matches,
		camapper.Metadata{
			SourceChainID:      &ids.Empty,
			DestinationChainID: &ids.Empty,
		},
		b.codec,
		b.avaxAssetID,
	)
	if err != nil {
		return 0, err
	}

	return tx.GasUsed(true)
}

// ConstructionMetadata implements /construction/metadata endpoint for C-chain atomic transactions
func (b *Backend) ConstructionMetadata(
	ctx context.Context,
	req *types.ConstructionMetadataRequest,
) (*types.ConstructionMetadataResponse, *types.Error) {
	var input camapper.Options
	err := mapper.UnmarshalJSONMap(req.Options, &input)
	if err != nil {
		return nil, backend.WrapError(backend.ErrInvalidInput, err)
	}

	networkID, err := b.cClient.GetNetworkID(ctx)
	if err != nil {
		return nil, backend.WrapError(backend.ErrClientError, err)
	}

	cChainID, err := b.cClient.GetBlockchainID(ctx, constants.CChain.String())
	if err != nil {
		return nil, backend.WrapError(backend.ErrClientError, err)
	}

	metadata := camapper.Metadata{
		NetworkID: networkID,
		CChainID:  cChainID,
	}

	if input.SourceChain != "" {
		id, err := b.cClient.GetBlockchainID(ctx, input.SourceChain)
		if err != nil {
			return nil, backend.WrapError(backend.ErrClientError, err)
		}
		metadata.SourceChainID = &id
	}

	if input.DestinationChain != "" {
		id, err := b.cClient.GetBlockchainID(ctx, input.DestinationChain)
		if err != nil {
			return nil, backend.WrapError(backend.ErrClientError, err)
		}
		metadata.DestinationChain = input.DestinationChain
		metadata.DestinationChainID = &id
	}

	if input.From != "" {
		var nonce uint64
		if input.Nonce == nil {
			nonce, err = b.cClient.NonceAt(ctx, ethcommon.HexToAddress(input.From), nil)
			if err != nil {
				return nil, backend.WrapError(backend.ErrClientError, err)
			}
		} else {
			nonce = input.Nonce.Uint64()
		}
		metadata.Nonce = nonce
	}

	metadataMap, err := mapper.MarshalJSONMap(metadata)
	if err != nil {
		return nil, backend.WrapError(backend.ErrInternalError, err)
	}

	suggestedFeeAmount, err := b.calculateSuggestedFee(ctx, input.AtomicTxGas)
	if err != nil {
		return nil, backend.WrapError(backend.ErrClientError, err)
	}

	return &types.ConstructionMetadataResponse{
		Metadata: metadataMap,
		SuggestedFee: []*types.Amount{
			suggestedFeeAmount,
		},
	}, nil
}

func (b *Backend) calculateSuggestedFee(ctx context.Context, gasUsed *big.Int) (*types.Amount, error) {
	baseFee, err := b.cClient.EstimateBaseFee(ctx)
	if err != nil {
		return nil, err
	}

	suggestedFeeEth := new(big.Int).Mul(gasUsed, baseFee)
	suggestedFee := new(big.Int).Div(suggestedFeeEth, cmapper.X2crate)
	return camapper.AtomicAvaxAmount(suggestedFee), nil
}

// ConstructionPayloads implements /construction/payloads endpoint for C-chain atomic transactions
func (b *Backend) ConstructionPayloads(ctx context.Context, req *types.ConstructionPayloadsRequest) (*types.ConstructionPayloadsResponse, *types.Error) {
	builder := cAtomicTxBuilder{
		avaxAssetID:  b.avaxAssetID,
		codec:        b.codec,
		codecVersion: b.codecVersion,
	}
	return common.BuildPayloads(builder, req)
}

// ConstructionParse implements /construction/parse endpoint for C-chain atomic transactions
func (b *Backend) ConstructionParse(ctx context.Context, req *types.ConstructionParseRequest) (*types.ConstructionParseResponse, *types.Error) {
	rosettaTx, err := b.parsePayloadTxFromString(req.Transaction)
	if err != nil {
		return nil, backend.WrapError(backend.ErrInvalidInput, err)
	}

	hrp, err := mapper.GetHRP(req.NetworkIdentifier)
	if err != nil {
		return nil, backend.WrapError(backend.ErrInvalidInput, "incorrect network identifier")
	}

	chainIDs := map[ids.ID]string{}
	if rosettaTx.DestinationChainID != nil {
		chainIDs[*rosettaTx.DestinationChainID] = rosettaTx.DestinationChain
	}

	txParser := cAtomicTxParser{
		hrp:      hrp,
		chainIDs: chainIDs,
	}

	return common.Parse(txParser, rosettaTx, req.Signed)
}

func (b *Backend) parsePayloadTxFromString(transaction string) (*common.RosettaTx, error) {
	// Unmarshal input transaction
	payloadsTx := &common.RosettaTx{
		Tx: &cAtomicTx{
			Codec:        b.codec,
			CodecVersion: b.codecVersion,
		},
	}

	err := json.Unmarshal([]byte(transaction), payloadsTx)
	if err != nil {
		return nil, errUndecodableTx
	}

	return payloadsTx, payloadsTx.Tx.Initialize()
}

// ConstructionCombine implements /construction/combine endpoint for C-chain atomic transactions
func (b *Backend) ConstructionCombine(ctx context.Context, req *types.ConstructionCombineRequest) (*types.ConstructionCombineResponse, *types.Error) {
	rosettaTx, err := b.parsePayloadTxFromString(req.UnsignedTransaction)
	if err != nil {
		return nil, backend.WrapError(backend.ErrInvalidInput, err)
	}

	return common.Combine(b, rosettaTx, req.Signatures)
}

// CombineTx implements C-chain atomic transaction specific logic for combining unsigned transactions and signatures
func (b *Backend) CombineTx(tx common.AvaxTx, signatures []*types.Signature) (common.AvaxTx, *types.Error) {
	cTx, ok := tx.(*cAtomicTx)
	if !ok {
		return nil, backend.WrapError(backend.ErrInvalidInput, "invalid transaction")
	}

	creds, err := getTxCreds(cTx.Tx.UnsignedAtomicTx, signatures)
	if err != nil {
		return nil, backend.WrapError(backend.ErrInvalidInput, "unable attach signatures to transaction")
	}

	unsignedBytes, err := cTx.Marshal()
	if err != nil {
		return nil, backend.WrapError(backend.ErrInvalidInput, "unable to encode unsigned transaction")
	}

	cTx.Tx.Creds = creds

	signedBytes, err := cTx.Marshal()
	if err != nil {
		return nil, backend.WrapError(backend.ErrInternalError, "unable to marshal signed transaction")
	}

	cTx.Tx.Initialize(unsignedBytes, signedBytes)

	return cTx, nil
}

// getTxCreds fetches credentials based on the tx type
func getTxCreds(
	unsignedAtomicTx evm.UnsignedAtomicTx,
	signatures []*types.Signature,
) ([]verify.Verifiable, error) {
	switch uat := unsignedAtomicTx.(type) {
	case *evm.UnsignedImportTx:
		return common.BuildCredentialList(uat.ImportedInputs, signatures)
	case *evm.UnsignedExportTx:
		return common.BuildSingletonCredentialList(signatures)
	}

	return nil, errUnknownTxType
}

// ConstructionHash implements /construction/hash endpoint for C-chain atomic transactions
func (b *Backend) ConstructionHash(
	ctx context.Context,
	req *types.ConstructionHashRequest,
) (*types.TransactionIdentifierResponse, *types.Error) {
	rosettaTx, err := b.parsePayloadTxFromString(req.SignedTransaction)
	if err != nil {
		return nil, backend.WrapError(backend.ErrInvalidInput, err)
	}

	return common.HashTx(rosettaTx)
}

// ConstructionSubmit implements /construction/submit endpoint for C-chain atomic transactions
func (b *Backend) ConstructionSubmit(
	ctx context.Context,
	req *types.ConstructionSubmitRequest,
) (*types.TransactionIdentifierResponse, *types.Error) {
	rosettaTx, err := b.parsePayloadTxFromString(req.SignedTransaction)
	if err != nil {
		return nil, backend.WrapError(backend.ErrInvalidInput, err)
	}

	return common.SubmitTx(ctx, b.cClient, rosettaTx)
}

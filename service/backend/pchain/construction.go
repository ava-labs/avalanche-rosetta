package pchain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/rpc"
	"github.com/ava-labs/avalanchego/vms/components/avax"
	"github.com/ava-labs/avalanchego/vms/platformvm/txs"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ava-labs/avalanche-rosetta/constants"
	"github.com/ava-labs/avalanche-rosetta/mapper"
	pmapper "github.com/ava-labs/avalanche-rosetta/mapper/pchain"
	"github.com/ava-labs/avalanche-rosetta/service"
	"github.com/ava-labs/avalanche-rosetta/service/backend/common"
)

var (
	errUnknownTxType = errors.New("unknown tx type")
	errUndecodableTx = errors.New("undecodable transaction")
)

// ConstructionDerive implements /construction/derive endpoint for P-chain
func (b *Backend) ConstructionDerive(_ context.Context, req *types.ConstructionDeriveRequest) (*types.ConstructionDeriveResponse, *types.Error) {
	return common.DeriveBech32Address(b.fac, constants.PChain, req)
}

// ConstructionPreprocess implements /construction/preprocess endpoint for P-chain
func (b *Backend) ConstructionPreprocess(
	_ context.Context,
	req *types.ConstructionPreprocessRequest,
) (*types.ConstructionPreprocessResponse, *types.Error) {
	matches, err := common.MatchOperations(req.Operations)
	if err != nil {
		return nil, service.WrapError(service.ErrInvalidInput, err)
	}

	reqMetadata := req.Metadata
	if reqMetadata == nil {
		reqMetadata = make(map[string]interface{})
	}
	reqMetadata[pmapper.MetadataOpType] = matches[0].Operations[0].Type

	return &types.ConstructionPreprocessResponse{
		Options: reqMetadata,
	}, nil
}

// ConstructionMetadata implements /construction/metadata endpoint for P-chain
func (b *Backend) ConstructionMetadata(
	ctx context.Context,
	req *types.ConstructionMetadataRequest,
) (*types.ConstructionMetadataResponse, *types.Error) {
	opMetadata, err := pmapper.ParseOpMetadata(req.Options)
	if err != nil {
		return nil, service.WrapError(service.ErrInvalidInput, err)
	}

	var suggestedFee *types.Amount
	var metadata *pmapper.Metadata
	switch opMetadata.Type {
	case pmapper.OpImportAvax:
		metadata, suggestedFee, err = b.buildImportMetadata(ctx, req.Options)
	case pmapper.OpExportAvax:
		metadata, suggestedFee, err = b.buildExportMetadata(ctx, req.Options)
	case pmapper.OpAddValidator, pmapper.OpAddDelegator:
		metadata, suggestedFee, err = b.buildStakingMetadata(req.Options)
		metadata.Threshold = opMetadata.Threshold
		metadata.Locktime = opMetadata.Locktime

	default:
		return nil, service.WrapError(
			service.ErrInternalError,
			fmt.Errorf("invalid tx type for building metadata: %s", opMetadata.Type),
		)
	}
	if err != nil {
		return nil, service.WrapError(service.ErrInternalError, err)
	}

	pChainID, err := b.pClient.GetBlockchainID(ctx, constants.PChain.String())
	if err != nil {
		return nil, service.WrapError(service.ErrInvalidInput, err)
	}

	metadata.NetworkID = b.avalancheNetworkID
	metadata.BlockchainID = pChainID

	metadataMap, err := mapper.MarshalJSONMap(metadata)
	if err != nil {
		return nil, service.WrapError(service.ErrInternalError, err)
	}

	return &types.ConstructionMetadataResponse{
		Metadata:     metadataMap,
		SuggestedFee: []*types.Amount{suggestedFee},
	}, nil
}

func (b *Backend) buildImportMetadata(ctx context.Context, options map[string]interface{}) (*pmapper.Metadata, *types.Amount, error) {
	var preprocessOptions pmapper.ImportExportOptions
	if err := mapper.UnmarshalJSONMap(options, &preprocessOptions); err != nil {
		return nil, nil, err
	}

	sourceChainID, err := b.pClient.GetBlockchainID(ctx, preprocessOptions.SourceChain)
	if err != nil {
		return nil, nil, err
	}

	suggestedFee, err := b.getBaseTxFee(ctx)
	if err != nil {
		return nil, nil, err
	}

	importMetadata := &pmapper.ImportMetadata{
		SourceChainID: sourceChainID,
	}

	return &pmapper.Metadata{ImportMetadata: importMetadata}, suggestedFee, nil
}

func (b *Backend) buildExportMetadata(ctx context.Context, options map[string]interface{}) (*pmapper.Metadata, *types.Amount, error) {
	var preprocessOptions pmapper.ImportExportOptions
	if err := mapper.UnmarshalJSONMap(options, &preprocessOptions); err != nil {
		return nil, nil, err
	}

	destinationChainID, err := b.pClient.GetBlockchainID(ctx, preprocessOptions.DestinationChain)
	if err != nil {
		return nil, nil, err
	}

	suggestedFee, err := b.getBaseTxFee(ctx)
	if err != nil {
		return nil, nil, err
	}

	exportMetadata := &pmapper.ExportMetadata{
		DestinationChain:   preprocessOptions.DestinationChain,
		DestinationChainID: destinationChainID,
	}

	return &pmapper.Metadata{ExportMetadata: exportMetadata}, suggestedFee, nil
}

func (b *Backend) buildStakingMetadata(options map[string]interface{}) (*pmapper.Metadata, *types.Amount, error) {
	var preprocessOptions pmapper.StakingOptions
	if err := mapper.UnmarshalJSONMap(options, &preprocessOptions); err != nil {
		return nil, nil, err
	}

	stakingMetadata := &pmapper.StakingMetadata{
		NodeID:          preprocessOptions.NodeID,
		Start:           preprocessOptions.Start,
		End:             preprocessOptions.End,
		Memo:            preprocessOptions.Memo,
		Locktime:        preprocessOptions.Locktime,
		Threshold:       preprocessOptions.Threshold,
		RewardAddresses: preprocessOptions.RewardAddresses,
		Shares:          preprocessOptions.Shares,
	}

	zeroAvax := mapper.AtomicAvaxAmount(big.NewInt(0))

	return &pmapper.Metadata{StakingMetadata: stakingMetadata}, zeroAvax, nil
}

func (b *Backend) getBaseTxFee(ctx context.Context) (*types.Amount, error) {
	fees, err := b.pClient.GetTxFee(ctx)
	if err != nil {
		return nil, err
	}

	feeAmount := new(big.Int).SetUint64(uint64(fees.TxFee))
	suggestedFee := mapper.AtomicAvaxAmount(feeAmount)
	return suggestedFee, nil
}

// ConstructionPayloads implements /construction/payloads endpoint for P-chain
func (b *Backend) ConstructionPayloads(_ context.Context, req *types.ConstructionPayloadsRequest) (*types.ConstructionPayloadsResponse, *types.Error) {
	builder := pTxBuilder{
		avaxAssetID:  b.avaxAssetID,
		codec:        b.codec,
		codecVersion: b.codecVersion,
	}
	return common.BuildPayloads(builder, req)
}

// ConstructionParse implements /construction/parse endpoint for P-chain
func (b *Backend) ConstructionParse(_ context.Context, req *types.ConstructionParseRequest) (*types.ConstructionParseResponse, *types.Error) {
	rosettaTx, err := b.parsePayloadTxFromString(req.Transaction)
	if err != nil {
		return nil, service.WrapError(service.ErrInvalidInput, err)
	}

	netID, _ := constants.FromString(rosettaTx.DestinationChain)
	chainIDs := map[ids.ID]constants.ChainIDAlias{}
	if rosettaTx.DestinationChainID != nil {
		chainIDs[*rosettaTx.DestinationChainID] = netID
	}

	txParser := pTxParser{
		hrp:         b.networkHRP,
		chainIDs:    chainIDs,
		avaxAssetID: b.avaxAssetID,
	}

	return common.Parse(txParser, rosettaTx, req.Signed)
}

// ConstructionCombine implements /construction/combine endpoint for P-chain
func (b *Backend) ConstructionCombine(_ context.Context, req *types.ConstructionCombineRequest) (*types.ConstructionCombineResponse, *types.Error) {
	rosettaTx, err := b.parsePayloadTxFromString(req.UnsignedTransaction)
	if err != nil {
		return nil, service.WrapError(service.ErrInvalidInput, err)
	}

	return common.Combine(b, rosettaTx, req.Signatures)
}

// CombineTx implements P-chain specific logic for combining unsigned transactions and signatures
func (b *Backend) CombineTx(tx common.AvaxTx, signatures []*types.Signature) (common.AvaxTx, *types.Error) {
	pTx, ok := tx.(*pTx)
	if !ok {
		return nil, service.WrapError(service.ErrInvalidInput, "invalid transaction")
	}

	ins, err := getTxInputs(pTx.Tx.Unsigned)
	if err != nil {
		return nil, service.WrapError(service.ErrInvalidInput, err)
	}

	creds, err := common.BuildCredentialList(ins, signatures)
	if err != nil {
		return nil, service.WrapError(service.ErrInvalidInput, err)
	}

	unsignedBytes, err := pTx.Marshal()
	if err != nil {
		return nil, service.WrapError(service.ErrInvalidInput, err)
	}

	pTx.Tx.Creds = creds

	signedBytes, err := pTx.Marshal()
	if err != nil {
		return nil, service.WrapError(service.ErrInvalidInput, err)
	}

	pTx.Tx.SetBytes(unsignedBytes, signedBytes)

	return pTx, nil
}

// getTxInputs fetches tx inputs based on the tx type.
func getTxInputs(
	unsignedTx txs.UnsignedTx,
) ([]*avax.TransferableInput, error) {
	switch utx := unsignedTx.(type) {
	case *txs.AddValidatorTx:
		return utx.Ins, nil
	case *txs.AddSubnetValidatorTx:
		return utx.Ins, nil
	case *txs.AddDelegatorTx:
		return utx.Ins, nil
	case *txs.CreateChainTx:
		return utx.Ins, nil
	case *txs.CreateSubnetTx:
		return utx.Ins, nil
	case *txs.ImportTx:
		return utx.ImportedInputs, nil
	case *txs.ExportTx:
		return utx.Ins, nil
	default:
		return nil, errUnknownTxType
	}
}

// ConstructionHash implements /construction/hash endpoint for P-chain
func (b *Backend) ConstructionHash(
	_ context.Context,
	req *types.ConstructionHashRequest,
) (*types.TransactionIdentifierResponse, *types.Error) {
	rosettaTx, err := b.parsePayloadTxFromString(req.SignedTransaction)
	if err != nil {
		return nil, service.WrapError(service.ErrInvalidInput, err)
	}

	return common.HashTx(rosettaTx)
}

// ConstructionSubmit implements /construction/submit endpoint for P-chain
func (b *Backend) ConstructionSubmit(
	ctx context.Context,
	req *types.ConstructionSubmitRequest,
) (*types.TransactionIdentifierResponse, *types.Error) {
	rosettaTx, err := b.parsePayloadTxFromString(req.SignedTransaction)
	if err != nil {
		return nil, service.WrapError(service.ErrInvalidInput, err)
	}

	return common.SubmitTx(ctx, b, rosettaTx)
}

// IssueTx broadcasts given transaction on P-chain
func (b *Backend) IssueTx(ctx context.Context, txByte []byte, options ...rpc.Option) (ids.ID, error) {
	return b.pClient.IssueTx(ctx, txByte, options...)
}

func (b *Backend) parsePayloadTxFromString(transaction string) (*common.RosettaTx, error) {
	// Unmarshal input transaction
	payloadsTx := &common.RosettaTx{
		Tx: &pTx{
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

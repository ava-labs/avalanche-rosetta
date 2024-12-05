package pchain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/rpc"
	"github.com/ava-labs/avalanchego/vms/components/avax"
	"github.com/ava-labs/avalanchego/vms/platformvm/txs"
	"github.com/coinbase/rosetta-sdk-go/parser"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ava-labs/avalanche-rosetta/constants"
	"github.com/ava-labs/avalanche-rosetta/mapper"
	"github.com/ava-labs/avalanche-rosetta/service"
	"github.com/ava-labs/avalanche-rosetta/service/backend/common"

	pmapper "github.com/ava-labs/avalanche-rosetta/mapper/pchain"
	txfee "github.com/ava-labs/avalanchego/vms/platformvm/txs/fee"
)

var (
	errUnknownTxType = errors.New("unknown tx type")
	errUndecodableTx = errors.New("undecodable transaction")
)

// ConstructionDerive implements /construction/derive endpoint for P-chain
func (*Backend) ConstructionDerive(_ context.Context, req *types.ConstructionDeriveRequest) (*types.ConstructionDeriveResponse, *types.Error) {
	return common.DeriveBech32Address(constants.PChain, req)
}

// ConstructionPreprocess implements /construction/preprocess endpoint for P-chain
func (*Backend) ConstructionPreprocess(
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
	reqMetadata[pmapper.MetadataMatches] = matches

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

	if opMetadata.Matches == nil {
		return nil, service.WrapError(service.ErrInvalidInput, errors.New("matches not found in options"))
	}

	// Build a dummy base tx to calculate the base fee
	dummyBaseTx, _, err := pmapper.BuildTx(
		pmapper.OpDummyBase,
		opMetadata.Matches,
		pmapper.Metadata{BlockchainID: ids.Empty},
		b.codec,
		b.avaxAssetID,
	)
	if err != nil {
		return nil, service.WrapError(service.ErrInternalError, err)
	}
	suggestedFee, err := b.calculateFee(ctx, dummyBaseTx)
	if err != nil {
		return nil, service.WrapError(service.ErrInternalError, err)
	}

	var metadata *pmapper.Metadata
	switch opMetadata.Type {
	case pmapper.OpImportAvax:
		metadata, err = b.buildImportMetadata(ctx, req.Options)
	case pmapper.OpExportAvax:
		metadata, err = b.buildExportMetadata(ctx, req.Options)
	case pmapper.OpAddValidator, pmapper.OpAddDelegator, pmapper.OpAddPermissionlessDelegator, pmapper.OpAddPermissionlessValidator:
		metadata, suggestedFee, err = b.buildStakingMetadata(ctx, req.Options, opMetadata.Matches, opMetadata.Type)
		if err != nil {
			return nil, service.WrapError(service.ErrInternalError, err)
		}
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

	suggestedFeeAvax := mapper.AtomicAvaxAmount(big.NewInt(int64(suggestedFee)))
	return &types.ConstructionMetadataResponse{
		Metadata:     metadataMap,
		SuggestedFee: []*types.Amount{suggestedFeeAvax},
	}, nil
}

func (b *Backend) buildImportMetadata(
	ctx context.Context,
	options map[string]interface{},
) (*pmapper.Metadata, error) {
	var preprocessOptions pmapper.ImportExportOptions
	if err := mapper.UnmarshalJSONMap(options, &preprocessOptions); err != nil {
		return nil, err
	}

	sourceChainID, err := b.pClient.GetBlockchainID(ctx, preprocessOptions.SourceChain)
	if err != nil {
		return nil, err
	}

	importMetadata := &pmapper.ImportMetadata{
		SourceChainID: sourceChainID,
	}

	return &pmapper.Metadata{ImportMetadata: importMetadata}, nil
}

func (b *Backend) buildExportMetadata(
	ctx context.Context,
	options map[string]interface{},
) (*pmapper.Metadata, error) {
	var preprocessOptions pmapper.ImportExportOptions
	if err := mapper.UnmarshalJSONMap(options, &preprocessOptions); err != nil {
		return nil, err
	}

	destinationChainID, err := b.pClient.GetBlockchainID(ctx, preprocessOptions.DestinationChain)
	if err != nil {
		return nil, err
	}

	exportMetadata := &pmapper.ExportMetadata{
		DestinationChain:   preprocessOptions.DestinationChain,
		DestinationChainID: destinationChainID,
	}

	return &pmapper.Metadata{ExportMetadata: exportMetadata}, nil
}

func (b *Backend) buildStakingMetadata(
	ctx context.Context,
	options map[string]interface{},
	matches []*parser.Match,
	opType string,
) (*pmapper.Metadata, uint64, error) {
	var preprocessOptions pmapper.StakingOptions
	if err := mapper.UnmarshalJSONMap(options, &preprocessOptions); err != nil {
		return nil, 0, err
	}

	stakingMetadata := &pmapper.Metadata{
		StakingMetadata: &pmapper.StakingMetadata{
			NodeID:                  preprocessOptions.NodeID,
			BLSPublicKey:            preprocessOptions.BLSPublicKey,
			BLSProofOfPossession:    preprocessOptions.BLSProofOfPossession,
			ValidationRewardsOwners: preprocessOptions.ValidationRewardsOwners,
			DelegationRewardsOwners: preprocessOptions.DelegationRewardsOwners,
			Start:                   preprocessOptions.Start,
			End:                     preprocessOptions.End,
			Subnet:                  preprocessOptions.Subnet,
			Shares:                  preprocessOptions.Shares,
			Locktime:                preprocessOptions.Locktime,
			Threshold:               preprocessOptions.Threshold,
		},
	}

	// Build a dummy staking tx to calculate the staking related fee
	dummyStakingTx, _, err := pmapper.BuildTx(
		opType,
		matches,
		*stakingMetadata,
		b.codec,
		b.avaxAssetID,
	)
	if err != nil {
		return nil, 0, err
	}
	suggestedFee, err := b.calculateFee(ctx, dummyStakingTx)
	if err != nil {
		return nil, 0, err
	}

	return stakingMetadata, suggestedFee, nil
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
func (*Backend) CombineTx(tx common.AvaxTx, signatures []*types.Signature) (common.AvaxTx, *types.Error) {
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

func (b *Backend) calculateFee(ctx context.Context, tx *txs.Tx) (uint64, error) {
	feeCalculator, err := b.PickFeeCalculator(ctx, time.Now())
	if err != nil {
		return 0, err
	}
	fee, err := feeCalculator.CalculateFee(tx.Unsigned)
	if err != nil {
		return 0, err
	}
	return fee, nil
}

func (b *Backend) PickFeeCalculator(ctx context.Context, timestamp time.Time) (txfee.Calculator, error) {
	if !b.upgradeConfig.IsEtnaActivated(timestamp) {
		return b.NewStaticFeeCalculator(timestamp), nil
	}

	_, gasPrice, _, err := b.pClient.GetFeeState(ctx)
	if err != nil {
		return nil, err
	}
	return txfee.NewDynamicCalculator(
		b.feeConfig.DynamicFeeConfig.Weights,
		gasPrice,
	), nil
}

func (b *Backend) NewStaticFeeCalculator(timestamp time.Time) txfee.Calculator {
	feeConfig := b.feeConfig.StaticFeeConfig
	if !b.upgradeConfig.IsApricotPhase3Activated(timestamp) {
		feeConfig.CreateSubnetTxFee = b.feeConfig.CreateAssetTxFee
		feeConfig.CreateBlockchainTxFee = b.feeConfig.CreateAssetTxFee
	}
	return txfee.NewStaticCalculator(feeConfig)
}

// getTxInputs fetches tx inputs based on the tx type.
func getTxInputs(
	unsignedTx txs.UnsignedTx,
) ([]*avax.TransferableInput, error) {
	// TODO: Move to using [txs.Visitor] from AvalancheGo
	// Ref: https://github.com/ava-labs/avalanchego/blob/master/vms/platformvm/txs/visitor.go
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
	case *txs.AdvanceTimeTx:
		return nil, nil
	case *txs.RewardValidatorTx:
		return nil, nil
	case *txs.TransformSubnetTx:
		return utx.Ins, nil
	case *txs.AddPermissionlessValidatorTx:
		return utx.Ins, nil
	case *txs.AddPermissionlessDelegatorTx:
		return utx.Ins, nil
	case *txs.TransferSubnetOwnershipTx:
		return utx.Ins, nil
	case *txs.ConvertSubnetToL1Tx:
		return utx.Ins, nil
	case *txs.RegisterL1ValidatorTx:
		return utx.Ins, nil
	case *txs.IncreaseL1ValidatorBalanceTx:
		return utx.Ins, nil
	case *txs.SetL1ValidatorWeightTx:
		return utx.Ins, nil
	case *txs.DisableL1ValidatorTx:
		return utx.Ins, nil
	case *txs.BaseTx:
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

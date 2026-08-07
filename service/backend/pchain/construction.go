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
	"github.com/ava-labs/avalanchego/vms/platformvm/reward"
	"github.com/ava-labs/avalanchego/vms/platformvm/txs"
	"github.com/ava-labs/avalanchego/vms/secp256k1fx"
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

	var metadata *pmapper.Metadata
	switch opMetadata.Type {
	case pmapper.OpImportAvax:
		metadata, err = b.buildImportMetadata(ctx, req.Options)
	case pmapper.OpExportAvax:
		metadata, err = b.buildExportMetadata(ctx, req.Options)
	case pmapper.OpAddValidator, pmapper.OpAddDelegator, pmapper.OpAddPermissionlessDelegator, pmapper.OpAddPermissionlessValidator:
		metadata, err = b.buildStakingMetadata(ctx, req.Options)
		if err != nil {
			return nil, service.WrapError(service.ErrInternalError, err)
		}
		metadata.Threshold = opMetadata.Threshold
		metadata.Locktime = opMetadata.Locktime
	case pmapper.OpAddAutoRenewedValidator:
		metadata, err = b.buildAutoRenewedValidatorMetadata(ctx, req.Options)
	case pmapper.OpSetAutoRenewedValidatorConfig:
		metadata, err = b.buildAutoRenewedValidatorConfigMetadata(ctx, req.Options)
	default:
		return nil, service.WrapError(
			service.ErrInternalError,
			fmt.Errorf("invalid tx type for building metadata: %s", opMetadata.Type),
		)
	}
	if err != nil {
		return nil, service.WrapError(service.ErrInternalError, err)
	}

	// Suggested fee calculation
	tx, _, err := pmapper.BuildTx(
		opMetadata.Type,
		opMetadata.Matches,
		*metadata,
		b.codec,
		b.avaxAssetID,
	)
	if err != nil {
		return nil, service.WrapError(service.ErrInternalError, err)
	}
	suggestedFee, err := b.calculateFee(ctx, tx)
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

func (*Backend) buildAutoRenewedValidatorMetadata(
	_ context.Context,
	options map[string]interface{},
) (*pmapper.Metadata, error) {
	var opts pmapper.AutoRenewedValidatorOptions
	if err := mapper.UnmarshalJSONMap(options, &opts); err != nil {
		return nil, err
	}

	// buildOutputOwner returns an empty (unspendable) owner for an empty address
	// list without erroring, so guard both owner lists here where the metadata is
	// constructed. validator_authority_addresses must be explicit: silently falling
	// back to the reward addresses would grant the reward key permanent authority to
	// reconfigure or stop the validator, which the caller never asked for.
	if len(opts.ValidationRewardsOwners) == 0 {
		return nil, errors.New("reward_addresses must be non-empty")
	}
	if len(opts.ValidatorAuthorityOwners) == 0 {
		return nil, errors.New("validator_authority_addresses must be non-empty")
	}

	// Fail fast on the network-independent syntactic constraints
	// AddAutoRenewedValidatorTx enforces, so the caller learns at /metadata rather
	// than hitting an opaque /submit failure.
	if opts.Period == 0 {
		return nil, errors.New("period must be non-zero")
	}
	if opts.Shares > reward.PercentDenominator {
		return nil, fmt.Errorf("shares must be <= %d", reward.PercentDenominator)
	}
	if opts.AutoCompoundRewardShares > reward.PercentDenominator {
		return nil, fmt.Errorf("auto_compound_reward_shares must be <= %d", reward.PercentDenominator)
	}

	// A zero threshold on a non-empty owner is rejected by the P-chain on issue
	// (OutputOwners.Verify returns ErrOutputUnoptimized), so default each to 1 when
	// the client does not specify one.
	if opts.Threshold == 0 {
		opts.Threshold = 1
	}
	if opts.ValidatorAuthorityThreshold == 0 {
		opts.ValidatorAuthorityThreshold = 1
	}

	// AutoRenewedValidatorMetadata is an alias of AutoRenewedValidatorOptions, so the
	// validated options are returned directly rather than copied field by field.
	return &pmapper.Metadata{AutoRenewedValidator: &opts}, nil
}

func (*Backend) buildAutoRenewedValidatorConfigMetadata(
	_ context.Context,
	options map[string]interface{},
) (*pmapper.Metadata, error) {
	var opts pmapper.AutoRenewedValidatorConfigOptions
	if err := mapper.UnmarshalJSONMap(options, &opts); err != nil {
		return nil, err
	}

	// Each authority key produces an extra signing payload in the construction flow,
	// so at least one is required. AuthAddresses[i] pairs with AuthSigIndices[i], and
	// the indices must be strictly ascending, as secp256k1fx.Input requires.
	if len(opts.AuthAddresses) == 0 {
		return nil, errors.New("auth_addresses must be non-empty")
	}
	if len(opts.AuthAddresses) != len(opts.AuthSigIndices) {
		return nil, errors.New("auth_addresses and auth_sig_indices must have equal length")
	}
	for i := 1; i < len(opts.AuthSigIndices); i++ {
		if opts.AuthSigIndices[i] <= opts.AuthSigIndices[i-1] {
			return nil, errors.New("auth_sig_indices must be strictly ascending")
		}
	}

	stakingTxID, err := ids.FromString(opts.StakingTxID)
	if err != nil {
		return nil, fmt.Errorf("invalid staking_tx_id: %w", err)
	}

	return &pmapper.Metadata{
		AutoRenewedValidatorConfig: &pmapper.AutoRenewedValidatorConfigMetadata{
			StakingTxID:              stakingTxID,
			AutoCompoundRewardShares: opts.AutoCompoundRewardShares,
			Period:                   opts.Period,
			AuthAddresses:            opts.AuthAddresses,
			AuthSigIndices:           opts.AuthSigIndices,
		},
	}, nil
}

func (*Backend) buildStakingMetadata(
	_ context.Context,
	options map[string]interface{},
) (*pmapper.Metadata, error) {
	var preprocessOptions pmapper.StakingOptions
	if err := mapper.UnmarshalJSONMap(options, &preprocessOptions); err != nil {
		return nil, err
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

	return stakingMetadata, nil
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

	numAuthSigs, err := getTxAuthSigCount(pTx.Tx.Unsigned)
	if err != nil {
		return nil, service.WrapError(service.ErrInvalidInput, err)
	}

	wantTotal := len(ins) + numAuthSigs
	if len(signatures) < wantTotal {
		return nil, service.WrapError(service.ErrInvalidInput, fmt.Errorf(
			"need %d signature(s) (%d for inputs, %d for auth), got %d",
			wantTotal, len(ins), numAuthSigs, len(signatures),
		))
	}

	inputSigs := signatures[:len(signatures)-numAuthSigs]
	authSigs := signatures[len(signatures)-numAuthSigs:]

	creds, err := common.BuildCredentialList(ins, inputSigs)
	if err != nil {
		return nil, service.WrapError(service.ErrInvalidInput, err)
	}

	if numAuthSigs > 0 {
		authCreds, err := common.BuildSingletonCredentialList(authSigs)
		if err != nil {
			return nil, service.WrapError(service.ErrInvalidInput, err)
		}
		creds = append(creds, authCreds...)
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

// getTxAuthSigCount returns the number of trailing signatures required for a
// tx-level authorization credential (e.g. SetAutoRenewedValidatorConfigTx.Auth).
// Returns 0 for tx types that have no such credential.
func getTxAuthSigCount(unsignedTx txs.UnsignedTx) (int, error) {
	utx, ok := unsignedTx.(*txs.SetAutoRenewedValidatorConfigTx)
	if !ok {
		return 0, nil
	}
	authInput, ok := utx.Auth.(*secp256k1fx.Input)
	if !ok {
		return 0, errors.New("unsupported auth type in SetAutoRenewedValidatorConfigTx")
	}
	return len(authInput.SigIndices), nil
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
	_, gasPrice, _, err := b.pClient.GetFeeState(ctx)
	if err != nil {
		return nil, err
	}
	return txfee.NewDynamicCalculator(
		b.feeConfig.DynamicFeeConfig.Weights,
		gasPrice,
	), nil
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
	case *txs.AddAutoRenewedValidatorTx:
		return utx.Ins, nil
	case *txs.SetAutoRenewedValidatorConfigTx:
		return utx.Ins, nil
	case *txs.RewardAutoRenewedValidatorTx:
		return nil, nil
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

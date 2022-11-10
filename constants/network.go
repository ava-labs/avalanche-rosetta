package constants

import (
	"github.com/ava-labs/avalanchego/utils/constants"
	"github.com/ava-labs/coreth/params"
	"github.com/coinbase/rosetta-sdk-go/types"
)

const (
	MainnetCChainID = 43114
	MainnetCAssetID = "FvwEAhmxKfeiG8SnEvq42hc6whRyY3EFYAvebMqDNDGCgxN5Z"
	MainnetNetwork  = constants.MainnetName

	FujiCChainID = 43113
	FujiCAssetID = "U8iRqJoiJm8xZHAacmvYyZVwqQx6uDNtQeP3CQ6fcgQk3JqnK"
	FujiNetwork  = constants.FujiName

	StatusSuccess = "SUCCESS"
	StatusFailure = "FAILURE"
)

var (
	MainnetAP5Activation = params.AvalancheMainnetChainConfig.ApricotPhase5BlockTimestamp
	FujiAP5Activation    = params.AvalancheFujiChainConfig.ApricotPhase5BlockTimestamp

	StageBootstrap = &types.SyncStatus{
		Synced: types.Bool(false),
		Stage:  types.String("BOOTSTRAP"),
	}

	StageSynced = &types.SyncStatus{
		Synced: types.Bool(true),
		Stage:  types.String("SYNCED"),
	}

	OperationStatuses = []*types.OperationStatus{
		{
			Status:     StatusSuccess,
			Successful: true,
		},
		{
			Status:     StatusFailure,
			Successful: false,
		},
	}
)

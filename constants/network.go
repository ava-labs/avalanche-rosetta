package constants

import (
	"github.com/ava-labs/avalanchego/utils/constants"
	"github.com/ava-labs/coreth/params"
)

const (
	MainnetChainID = 43114
	MainnetAssetID = "FvwEAhmxKfeiG8SnEvq42hc6whRyY3EFYAvebMqDNDGCgxN5Z"
	MainnetNetwork = constants.MainnetName

	FujiChainID = 43113
	FujiAssetID = "U8iRqJoiJm8xZHAacmvYyZVwqQx6uDNtQeP3CQ6fcgQk3JqnK"
	FujiNetwork = constants.FujiName
)

var (
	MainnetAP5Activation = *params.AvalancheMainnetChainConfig.ApricotPhase5BlockTimestamp
	FujiAP5Activation    = *params.AvalancheFujiChainConfig.ApricotPhase5BlockTimestamp
)

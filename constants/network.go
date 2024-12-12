package constants

import (
	"github.com/ava-labs/avalanchego/upgrade"
	"github.com/ava-labs/avalanchego/utils/constants"
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
	mainnetUpgrades = upgrade.GetConfig(constants.MainnetID)
	fujiUpgrades    = upgrade.GetConfig(constants.FujiID)
)

var (
	MainnetAP5Activation = uint64(mainnetUpgrades.ApricotPhase5Time.Unix())
	FujiAP5Activation    = uint64(fujiUpgrades.ApricotPhase5Time.Unix())
)

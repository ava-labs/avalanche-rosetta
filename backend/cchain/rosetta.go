package cchain

import (
	"fmt"

	"github.com/ava-labs/avalanchego/version"
)

// TODO: move to constants

var NodeVersion = fmt.Sprintf(
	"%d.%d.%d",
	version.Current.Major,
	version.Current.Minor,
	version.Current.Patch,
)

const (
	MiddlewareVersion = "0.1.20"
	BlockchainName    = "Avalanche"
)

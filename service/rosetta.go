package service

import (
	"fmt"

	"github.com/ava-labs/avalanchego/version"
)

var NodeVersion = fmt.Sprintf(
	"%d.%d.%d",
	version.Current.Major,
	version.Current.Minor,
	version.Current.Patch,
)

const (
	MiddlewareVersion = "0.1.34"
	BlockchainName    = "Avalanche"
)

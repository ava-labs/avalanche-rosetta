package service

import (
	"fmt"

	"github.com/chain4travel/caminogo/version"
)

var NodeVersion = fmt.Sprintf(
	"%d.%d.%d",
	version.Current.Major,
	version.Current.Minor,
	version.Current.Patch,
)

const (
	MiddlewareVersion = "0.1.22"
	BlockchainName    = "Camino"
)

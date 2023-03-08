package mapper

import (
	"testing"

	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/coinbase/rosetta-sdk-go/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

var USDC = &types.Currency{
	Symbol:   "USDC",
	Decimals: 6,
	Metadata: map[string]interface{}{
		"contractAddress": "0xB97EF9Ef8734C71904D8002F8b6Bc66Dd9c48a6E",
	},
}

func TestMixedCaseAddress(t *testing.T) {
	require := require.New(t)

	parsedCurrency := ToCurrency("USDC", 6, common.HexToAddress("0xB97EF9Ef8734C71904D8002F8b6Bc66Dd9c48a6E"))
	require.True(utils.Equal(USDC, parsedCurrency))
}

package mapper

import (
	"testing"

	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/coinbase/rosetta-sdk-go/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

var USDC = &types.Currency{
	Symbol:   "USDC",
	Decimals: 6,
	Metadata: map[string]interface{}{
		"contractAddress": common.HexToAddress("0xB97EF9Ef8734C71904D8002F8b6Bc66Dd9c48a6E"),
	},
}

func TestMixedCaseAddress(t *testing.T) {
	t.Run("correct address", func(t *testing.T) {
		assert.True(t, utils.Equal(USDC, ToCurrency("USDC", 6, common.HexToAddress("0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e"))))
	})
}

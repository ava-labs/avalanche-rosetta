package mapper

import (
	"strings"
	"testing"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func TestAdress(t *testing.T) {
	t.Run("ConvertEVMTopicHashToAddress", func(t *testing.T) {
		addressString := "0x54761841b2005ee456ba5a5a46ee78dded90b16d"
		hash := ethcommon.HexToHash(addressString)
		convertedAddress := ConvertEVMTopicHashToAddress(&hash)

		assert.Equal(t, strings.ToLower(addressString), strings.ToLower(convertedAddress.String()))
	})
}

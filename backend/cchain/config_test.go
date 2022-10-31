package cchain

import (
	"math/big"
	"testing"

	"github.com/ava-labs/avalanche-rosetta/constants"
	ethtypes "github.com/ava-labs/coreth/core/types"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	t.Run("online", func(t *testing.T) {
		cfg := Config{
			Mode:      constants.Online,
			ChainID:   big.NewInt(1),
			NetworkID: &types.NetworkIdentifier{},
		}

		assert.Equal(t, false, cfg.IsOfflineMode())
		assert.Equal(t, true, cfg.IsOnlineMode())
	})

	t.Run("offline", func(t *testing.T) {
		cfg := Config{
			Mode:      constants.Offline,
			ChainID:   big.NewInt(1),
			NetworkID: &types.NetworkIdentifier{},
		}

		assert.Equal(t, true, cfg.IsOfflineMode())
		assert.Equal(t, false, cfg.IsOnlineMode())
	})

	t.Run("signer", func(t *testing.T) {
		cfg := Config{
			ChainID: big.NewInt(1),
		}
		assert.IsType(t, ethtypes.NewLondonSigner(big.NewInt(1)), cfg.Signer())
	})
}

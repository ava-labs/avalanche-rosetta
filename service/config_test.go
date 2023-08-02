package service

import (
	"testing"

	ethtypes "github.com/ava-labs/coreth/core/types"
	"github.com/ava-labs/coreth/params"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	t.Run("online", func(t *testing.T) {
		cfg := Config{
			Mode:      "online",
			ChainID:   params.AvalancheMainnetChainID,
			NetworkID: &types.NetworkIdentifier{},
		}

		assert.Equal(t, false, cfg.IsOfflineMode())
		assert.Equal(t, true, cfg.IsOnlineMode())
	})

	t.Run("offline", func(t *testing.T) {
		cfg := Config{
			Mode:      "offline",
			ChainID:   params.AvalancheMainnetChainID,
			NetworkID: &types.NetworkIdentifier{},
		}

		assert.Equal(t, true, cfg.IsOfflineMode())
		assert.Equal(t, false, cfg.IsOnlineMode())
	})

	t.Run("signer", func(t *testing.T) {
		cfg := Config{
			ChainID: params.AvalancheMainnetChainID,
		}
		assert.IsType(t, ethtypes.NewLondonSigner(params.AvalancheMainnetChainID), cfg.Signer())
	})
}

package service

import (
	"testing"

	ethtypes "github.com/ava-labs/coreth/core/types"
	"github.com/ava-labs/coreth/params"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/stretchr/testify/require"
)

func TestConfig(t *testing.T) {
	t.Run("online", func(t *testing.T) {
		cfg := Config{
			Mode:      "online",
			ChainID:   params.AvalancheMainnetChainID,
			NetworkID: &types.NetworkIdentifier{},
		}

		require.Equal(t, false, cfg.IsOfflineMode())
		require.Equal(t, true, cfg.IsOnlineMode())
	})

	t.Run("offline", func(t *testing.T) {
		cfg := Config{
			Mode:      "offline",
			ChainID:   params.AvalancheMainnetChainID,
			NetworkID: &types.NetworkIdentifier{},
		}

		require.Equal(t, true, cfg.IsOfflineMode())
		require.Equal(t, false, cfg.IsOnlineMode())
	})

	t.Run("signer", func(t *testing.T) {
		cfg := Config{
			ChainID: params.AvalancheMainnetChainID,
		}
		require.IsType(t, ethtypes.NewLondonSigner(params.AvalancheMainnetChainID), cfg.Signer())
	})
}

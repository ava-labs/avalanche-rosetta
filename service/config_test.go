package service

import (
	"testing"

	"github.com/ava-labs/avalanchego/vms/saevm/cchain/libevm"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/stretchr/testify/require"

	ethtypes "github.com/ava-labs/libevm/core/types"
)

func TestConfig(t *testing.T) {
	libevm.RegisterExtras()
	t.Run("online", func(t *testing.T) {
		cfg := Config{
			Mode:      "online",
			ChainID:   avalancheMainnetChainID,
			NetworkID: &types.NetworkIdentifier{},
		}

		require.False(t, cfg.IsOfflineMode())
		require.True(t, cfg.IsOnlineMode())
	})

	t.Run("offline", func(t *testing.T) {
		cfg := Config{
			Mode:      "offline",
			ChainID:   avalancheMainnetChainID,
			NetworkID: &types.NetworkIdentifier{},
		}

		require.True(t, cfg.IsOfflineMode())
		require.False(t, cfg.IsOnlineMode())
	})

	t.Run("signer", func(t *testing.T) {
		cfg := Config{
			ChainID: avalancheMainnetChainID,
		}
		require.IsType(t, ethtypes.NewLondonSigner(avalancheMainnetChainID), cfg.Signer())
	})
}

package mapper

import (
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ava-labs/avalanchego/api/info"
)

func Peers(peers []info.Peer) []*types.Peer {
	result := make([]*types.Peer, len(peers))

	for idx, peer := range peers {
		result[idx] = &types.Peer{
			PeerID: peer.ID.String(),
			Metadata: map[string]interface{}{
				"ip":              peer.IP,
				"public_ip":       peer.PublicIP,
				"version":         peer.Version,
				"last_sent":       peer.LastSent,
				"last_received":   peer.LastReceived,
				"benched":         peer.Benched,
				"observed_uptime": peer.ObservedUptime,
				"tracked_subnets": peer.TrackedSubnets,
			},
		}
	}

	return result
}

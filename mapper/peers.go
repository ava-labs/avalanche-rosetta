package mapper

import (
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ava-labs/avalanche-rosetta/client"
)

func Peers(peers []client.Peer) []*types.Peer {
	result := make([]*types.Peer, len(peers))

	for idx, peer := range peers {
		result[idx] = &types.Peer{
			PeerID:   peer.ID,
			Metadata: peer.Metadata(),
		}
	}

	return result
}

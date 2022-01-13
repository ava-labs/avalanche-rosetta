package mapper

import (
	"encoding/json"

	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ava-labs/avalanchego/network"
)

func Peers(peers []network.PeerInfo) []*types.Peer {
	result := make([]*types.Peer, len(peers))

	var metadata map[string]interface{}
	j, _ := json.Marshal(metadata)
	json.Unmarshal(j, &metadata)

	for idx, peer := range peers {
		result[idx] = &types.Peer{
			PeerID:   peer.ID,
			Metadata: metadata,
		}
	}

	return result
}

package mapper

import (
	"encoding/json"

	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ava-labs/avalanchego/network"
	"github.com/ava-labs/avalanchego/utils/wrappers"
)

func Peers(peers []network.PeerInfo) []*types.Peer {
	var errs wrappers.Errs
	result := make([]*types.Peer, len(peers))

	for idx, peer := range peers {
		var metadata map[string]interface{}
		j, err := json.Marshal(peer)
		errs.Add(err)
		errs.Add(json.Unmarshal(j, &metadata))
		delete(metadata, "nodeID")

		result[idx] = &types.Peer{
			PeerID:   peer.ID,
			Metadata: metadata,
		}
	}

	if errs.Err != nil {
		return []*types.Peer{}
	}
	return result
}

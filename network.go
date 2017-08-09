package ppsppnet

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"

	"github.com/golang/glog"

	"github.com/livepeer/go-PPSPP/core"
	lpnet "github.com/livepeer/go-livepeer/net"
)

const (
	// PpsppChunkSize is the chunk size used by Swarms in the Livepeer
	// Video Network.
	// TODO: make the chunk size configurable.
	PpsppChunkSize = 1024
)

// PpsppVideoNetwork implements VideoNetwork using PPSPP
type PpsppVideoNetwork struct {
	self    *core.Peer
	tracker Tracker
}

// NewPpsppVideoNetwork creates a Livepeer Video Network on the given port.
func NewPpsppVideoNetwork(port uint16, tracker Tracker) (*PpsppVideoNetwork, error) {
	self, err := core.NewLibp2pPeer(int(port), core.NewPpspp())
	if err != nil {
		return nil, fmt.Errorf("error creating libp2p peer: %v", err)
	}
	return &PpsppVideoNetwork{
		self:    self,
		tracker: tracker,
	}, nil
}

func (n *PpsppVideoNetwork) GetNodeID() string {
	return n.self.ID().String()
}

func (n *PpsppVideoNetwork) GetBroadcaster(streamID string) (lpnet.Broadcaster, error) {
	// Register self with the tracker
	if err := n.tracker.Register(streamID, Peer{
		ID:    n.self.ID(),
		Addrs: n.self.Addrs(),
	}); err != nil {
		return nil, fmt.Errorf("error registering for stream %v with the tracker: %v", streamID, err)
	}

	// Get peers currently in the Swarm
	peers, err := n.tracker.GetPeers(streamID)
	if err != nil {
		return nil, fmt.Errorf("error getting peers for stream %v: %v", streamID, err)
	}

	// Create a local swarm
	swarmID := streamToSwarmID(streamID)
	if err := n.self.P.AddSwarm(core.SwarmConfig{
		Metadata: core.SwarmMetadata{
			ID:        swarmID,
			ChunkSize: PpsppChunkSize,
		},
	}); err != nil {
		return nil, fmt.Errorf("error adding swarm: %v", err)
	}

	// Connect to those peers
	// TODO: for a large Swarm, we only want to connect to a subset of peers.
	var connectedPeers []Peer
	for _, peer := range peers {
		n.self.AddAddrs(peer.ID, peer.Addrs)
		if err := n.self.Connect(peer.ID); err != nil {
			glog.Errorf("error connecting to peer %s", peer.ID)
			continue
		}
		if err := n.self.P.StartHandshake(peer.ID, swarmID); err != nil {
			glog.Fatalf("error starting handshake with %s: %v", peer.ID, err)
			continue
		}
		connectedPeers = append(connectedPeers, peer)
	}

	swarm, err := n.self.P.Swarm(swarmID)
	if err != nil {
		return nil, fmt.Errorf("error retrieving swarm: %v", err)
	}

	return &ppsppBroadcaster{
		p:       n.self.P,
		swarmID: swarmID,
		swarm:   swarm,
		peers:   connectedPeers,
	}, nil
}

// streamToSwarmID converts a stream ID into a Swarm ID.
// TODO: it's unclear why SwarmID has to be a uint32 in the first place.
// Maybe the right thing to do here is to change go-PPSPP to use strings
// for SwarmIDs.
func streamToSwarmID(streamID string) core.SwarmID {
	hash := sha256.Sum256([]byte(streamID))
	return core.SwarmID(binary.BigEndian.Uint32(hash[:4]))
}

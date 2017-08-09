package ppsppnet

import (
	"sync"

	"github.com/livepeer/go-PPSPP/core"
	ma "github.com/multiformats/go-multiaddr"
)

type Peer struct {
	ID    core.PeerID
	Addrs []ma.Multiaddr
}

// Tracker keeps track of the peers in a Swarm.
type Tracker interface {
	// GetPeers returns a list of peers that are currently in the Swarm.
	// Note that it's up to the implementation to decide whether the list
	// is complete and which peers are included in the list.
	GetPeers(string) ([]Peer, error)
	// Register signals to the Tracker the existence of a peer in the
	// given Swarm.
	Register(string, Peer) error
	// Deregister signals to the tracker that a given peer has left the Swarm.
	Deregister(string, core.PeerID) error
}

// NewMemTracker creates a local, in-memory tracker that's primarily used
// for testing purposes.
func NewMemTracker() Tracker {
	return &memTracker{
		peers: make(map[string]map[string]Peer),
	}
}

type memTracker struct {
	sync.RWMutex
	// peers is a map from a Swarm ID to a set of peers.
	peers map[string]map[string]Peer
}

func (m *memTracker) GetPeers(swarmID string) ([]Peer, error) {
	m.RLock()
	defer m.RUnlock()
	var peers []Peer
	for _, peer := range m.peers[swarmID] {
		peers = append(peers, peer)
	}
	return peers, nil
}

func (m *memTracker) Register(swarmID string, peer Peer) error {
	m.Lock()
	defer m.Unlock()
	if m.peers[swarmID] == nil {
		m.peers[swarmID] = make(map[string]Peer)
	}
	m.peers[swarmID][peer.ID.String()] = peer
	return nil
}

func (m *memTracker) Deregister(swarmID string, peerID core.PeerID) error {
	m.Lock()
	defer m.Unlock()
	delete(m.peers[swarmID], peerID.String())
	return nil
}

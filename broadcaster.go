package ppsppnet

import "github.com/livepeer/go-PPSPP/core"

type ppsppBroadcaster struct {
	swarm *core.Swarm
}

func (b *ppsppBroadcaster) Broadcast(seqNo uint64, data []byte) error {
	return nil
}

func (b *ppsppBroadcaster) Finish() error {
	return nil
}

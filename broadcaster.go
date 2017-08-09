package ppsppnet

import (
	"fmt"
	"sync"

	"github.com/golang/glog"
	"github.com/livepeer/go-PPSPP/core"
)

type ppsppBroadcaster struct {
	sync.Mutex

	swarmID core.SwarmID
	swarm   *core.Swarm
	peers   []Peer
	p       core.Protocol

	nextSeq   uint64       // the next segment to broadcast
	nextChunk core.ChunkID // the next chunk to broadcast
}

func (b *ppsppBroadcaster) Broadcast(seqNo uint64, data []byte) error {
	b.Lock()
	defer b.Unlock()

	if seqNo != b.nextSeq {
		return fmt.Errorf("expected seqNo %v, got %v", b.nextSeq, seqNo)
	}

	// Divide data into chunks and add them to local swarm
	var numChunks int
	var err error
	if numChunks, err = chunk(data, func(chunkNo int, chunk []byte) error {
		chunkID := b.nextChunk + core.ChunkID(chunkNo)
		if err := b.swarm.AddLocalChunk(chunkID, &core.Chunk{
			ID: chunkID,
			B:  chunk,
		}); err != nil {
			return fmt.Errorf("error adding local chunk: %v", err)
		}
		return nil
	}); err != nil {
		return err
	}

	// At this point, we consider the broadcast to have succeeded.
	// We then proceed to send Have to our peers in a best-effort basis.
	// TODO: investigate if Protocol.SendHave is thread-safe.
	go b.sendHaves(b.nextChunk, b.nextChunk+core.ChunkID(numChunks))

	b.nextSeq++
	b.nextChunk += core.ChunkID(numChunks)

	return nil
}

func (b *ppsppBroadcaster) Finish() error {
	return nil
}

func (b *ppsppBroadcaster) sendHaves(start core.ChunkID, end core.ChunkID) {
	for _, peer := range b.peers {
		if err := b.p.SendHave(start, end, peer.ID, b.swarmID); err != nil {
			glog.Fatalf("error sending HAVE to %s: %v", peer.ID, err)
		}
	}
}

// chunk divides a byte slice into many fixed-size chunks, and invoke a
// callback on each chunk.
// If the callback returns an error, this function terminates and returns
// the error.
// The function also returns the number of chunks processed so far.
func chunk(data []byte, callback func(chunkNo int, chunk []byte) error) (int, error) {
	var n int
	var chunkNo int
	for n < len(data) {
		if err := callback(chunkNo, data[n:n+PpsppChunkSize]); err != nil {
			return chunkNo, err
		}
		n += PpsppChunkSize
		chunkNo++
	}
	return chunkNo, nil
}

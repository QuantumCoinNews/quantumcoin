package p2p

import (
	"bytes"
	"crypto/rand"
	"encoding/gob"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"quantumcoin/blockchain"
)

const (
	MsgSyncRequest     MessageType = "sync-range"
	MsgSyncChunk       MessageType = "sync-chunk"
	MsgSyncReset       MessageType = "sync-reset"
	syncBlocksPerChunk             = 8
	syncChunkBytes                 = 896 << 10
	syncResumeTTL                  = 2 * time.Minute
	maxSyncDownloads               = 4
)

type syncRequest struct {
	ID        [16]byte
	Start     int
	TipHeight int
	TipHash   []byte
	Previous  []byte
}

type syncChunk struct {
	ID        [16]byte
	Start     int
	TipHeight int
	TipHash   []byte
	Blocks    []*blockchain.Block
}

type syncDownload struct {
	mu         sync.Mutex
	owner      *peer
	candidate  *blockchain.Blockchain
	request    syncRequest
	last       time.Time
	resets     int
	incomplete bool
}

var syncDownloads = struct {
	sync.Mutex
	chains map[*blockchain.Blockchain]*syncDownload
}{chains: make(map[*blockchain.Blockchain]*syncDownload)}

func getSyncDownload(chain *blockchain.Blockchain, create bool) *syncDownload {
	syncDownloads.Lock()
	defer syncDownloads.Unlock()
	if d := syncDownloads.chains[chain]; d != nil {
		return d
	}
	if !create {
		return nil
	}
	for key, d := range syncDownloads.chains {
		d.mu.Lock()
		idle := d.owner == nil && (d.candidate == nil || time.Since(d.last) > syncResumeTTL)
		d.mu.Unlock()
		if idle {
			delete(syncDownloads.chains, key)
		}
	}
	if len(syncDownloads.chains) >= maxSyncDownloads {
		return nil
	}
	d := &syncDownload{}
	syncDownloads.chains[chain] = d
	return d
}

func registeredPeer(conn net.Conn) *peer {
	if conn == nil {
		return nil
	}
	peersMu.Lock()
	defer peersMu.Unlock()
	p := peers[conn.RemoteAddr().String()]
	if p == nil || p.conn != conn {
		return nil
	}
	return p
}

func releaseSyncOwner(p *peer) {
	if p == nil {
		return
	}
	syncDownloads.Lock()
	defer syncDownloads.Unlock()
	for _, d := range syncDownloads.chains {
		d.mu.Lock()
		if d.owner == p {
			d.owner = nil
		}
		d.mu.Unlock()
	}
}

func encodeSync(kind MessageType, value any) (Message, error) {
	var buffer bytes.Buffer
	if err := gob.NewEncoder(&buffer).Encode(value); err != nil {
		return Message{}, err
	}
	msg := Message{Type: kind, Data: buffer.Bytes()}
	return msg, ValidateP2PMessageSizeStrictV1(msg)
}

func startChainSync(chain *blockchain.Blockchain, conn net.Conn) error {
	p := registeredPeer(conn)
	if chain == nil || p == nil {
		return fmt.Errorf("sync requires a registered local chain/peer")
	}
	d := getSyncDownload(chain, true)
	if d == nil {
		return fmt.Errorf("sync download slots are busy")
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.owner != nil {
		if registeredPeer(d.owner.conn) == d.owner {
			return nil
		}
		d.owner = nil
	}
	if time.Since(d.last) > syncResumeTTL {
		d.candidate = nil
	}
	d.owner, d.last, d.resets = p, time.Now(), 0
	d.incomplete = true
	p.syncAttempted.Store(true)
	d.request.ID = [16]byte{}
	if _, err := rand.Read(d.request.ID[:]); err != nil {
		d.owner = nil
		return err
	}
	if d.candidate == nil {
		d.request = syncRequest{ID: d.request.ID, TipHeight: -1}
	}
	return sendSyncRequestLocked(d)
}

func sendSyncRequestLocked(d *syncDownload) error {
	msg, err := encodeSync(MsgSyncRequest, d.request)
	if err != nil {
		return err
	}
	return d.owner.send(msg)
}

func respondSyncRange(chain *blockchain.Blockchain, conn net.Conn, data []byte) error {
	var request syncRequest
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&request); err != nil {
		return err
	}
	if request.ID == ([16]byte{}) || len(request.TipHash) > 32 || len(request.Previous) > 32 {
		return fmt.Errorf("invalid sync request")
	}
	blocks, height, hash, err := chain.PeerSyncRange(request.Start, syncBlocksPerChunk, request.TipHeight, request.TipHash, request.Previous)
	if err != nil {
		msg, encErr := encodeSync(MsgSyncReset, request.ID)
		if encErr != nil {
			return encErr
		}
		sendToPeer(conn, msg)
		return nil
	}
	chunk := syncChunk{ID: request.ID, Start: request.Start, TipHeight: height, TipHash: hash}
	var result Message
	for _, block := range blocks {
		if len(block.Serialize()) > MaxP2PBlockDataBytesV1 {
			return fmt.Errorf("sync block exceeds block payload limit")
		}
		chunk.Blocks = append(chunk.Blocks, block)
		encoded, err := encodeSync(MsgSyncChunk, chunk)
		if err != nil || len(encoded.Data) > syncChunkBytes {
			if len(chunk.Blocks) == 1 {
				return fmt.Errorf("sync block does not fit chunk")
			}
			break
		}
		result = encoded
	}
	if result.Type != MsgSyncChunk {
		return fmt.Errorf("empty sync range")
	}
	sendToPeer(conn, result)
	return nil
}

func acceptSyncChunk(chain *blockchain.Blockchain, conn net.Conn, data []byte) error {
	p := registeredPeer(conn)
	d := getSyncDownload(chain, false)
	if p == nil || d == nil {
		return fmt.Errorf("unsolicited sync chunk")
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.owner != p || time.Since(d.last) > syncResumeTTL {
		return fmt.Errorf("expired sync owner")
	}
	var chunk syncChunk
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&chunk); err != nil {
		return err
	}
	if chunk.ID != d.request.ID || chunk.Start != d.request.Start || len(chunk.Blocks) < 1 || len(chunk.Blocks) > syncBlocksPerChunk ||
		chunk.TipHeight < chunk.Start || len(chunk.TipHash) != 32 || len(chunk.Blocks)-1 > chunk.TipHeight-chunk.Start {
		return fmt.Errorf("invalid or replayed sync chunk")
	}
	if d.request.TipHeight >= 0 && (chunk.TipHeight != d.request.TipHeight || !bytes.Equal(chunk.TipHash, d.request.TipHash)) {
		return fmt.Errorf("sync tip changed during continuation")
	}
	for index, block := range chunk.Blocks {
		if block == nil || block.Index != chunk.Start+index || len(block.Serialize()) > MaxP2PBlockDataBytesV1 {
			return fmt.Errorf("invalid sync block position/size")
		}
	}
	for _, block := range chunk.Blocks {
		if block.Index == 0 {
			candidate, err := chain.NewPeerSyncCandidate(block)
			if err != nil {
				d.candidate = nil
				return err
			}
			d.candidate = candidate
		} else {
			if d.candidate == nil {
				return fmt.Errorf("sync candidate genesis missing")
			}
			if err := d.candidate.AddBlockFromPeerSynchronizedV2(block); err != nil {
				d.candidate = nil
				return err
			}
		}
	}
	last := chunk.Blocks[len(chunk.Blocks)-1]
	d.request.Start, d.request.Previous = last.Index+1, append([]byte(nil), last.Hash...)
	d.request.TipHeight, d.request.TipHash, d.last = chunk.TipHeight, append([]byte(nil), chunk.TipHash...), time.Now()
	if last.Index < chunk.TipHeight {
		return sendSyncRequestLocked(d)
	}
	if !bytes.Equal(last.Hash, chunk.TipHash) {
		d.candidate = nil
		return fmt.Errorf("sync completed at wrong tip")
	}
	err := chain.ReplaceChainFromPeerSynchronizedV1(d.candidate.BlocksSnapshot())
	if err != nil && !errors.Is(err, blockchain.ErrIncomingChainNotLonger) {
		d.candidate = nil
		return err
	}
	recordValidatedPeerHeight(conn.RemoteAddr().String(), last.Index)
	d.candidate, d.owner = nil, nil
	d.incomplete = false
	return nil
}

// Retry a deferred first attempt or an interrupted download, never repeatedly
// download a completed history on heartbeat. Only one replay is active per node.
func retryChainSync(chain *blockchain.Blockchain, p *peer) {
	if chain == nil || p == nil || !p.rangeSync.Load() {
		return
	}
	retry := !p.syncAttempted.Load()
	if d := getSyncDownload(chain, false); d != nil {
		d.mu.Lock()
		retry = retry || (d.incomplete && d.owner == nil)
		d.mu.Unlock()
	}
	if retry {
		_ = startChainSync(chain, p.conn)
	}
}

func resetChainSync(chain *blockchain.Blockchain, conn net.Conn, data []byte) error {
	p := registeredPeer(conn)
	d := getSyncDownload(chain, false)
	if p == nil || d == nil {
		return fmt.Errorf("unsolicited sync reset")
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	var id [16]byte
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&id); err != nil {
		return err
	}
	if d.owner != p || id != d.request.ID || d.resets >= 1 {
		return fmt.Errorf("invalid/repeated sync reset")
	}
	d.resets++
	d.candidate = nil
	d.request = syncRequest{TipHeight: -1}
	if _, err := rand.Read(d.request.ID[:]); err != nil {
		return err
	}
	d.last = time.Now()
	return sendSyncRequestLocked(d)
}

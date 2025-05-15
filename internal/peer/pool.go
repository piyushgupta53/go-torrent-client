package peer

import (
	"fmt"
	"sync"
	"time"

	"github.com/piyushgupta53/go-torrent/internal/tracker"
)

// Pool manages multiple peer connections
type Pool struct {
	InfoHash  [20]byte
	OurPeerID [20]byte
	Peers     map[string]*Client
	mu        sync.Mutex
}

// NewPool creates a new peer connection pool
func NewPool(infoHash, ourPeerID [20]byte) *Pool {
	return &Pool{
		InfoHash:  infoHash,
		OurPeerID: ourPeerID,
		Peers:     map[string]*Client{},
	}
}

// Connect attempts to connect to a list of peers
func (p *Pool) Connect(peers []tracker.Peer, maxConnections int) {
	connected := 0

	for _, peer := range peers {
		if connected >= maxConnections {
			break
		}

		peerAddr := peer.String()

		// Skip if already connected
		p.mu.Lock()
		if _, exists := p.Peers[peerAddr]; exists {
			p.mu.Unlock()
			continue
		}
		p.mu.Unlock()

		// Try to connect
		go func(addr string) {
			client, err := NewClient(peerAddr, p.InfoHash, p.OurPeerID)
			if err != nil {
				fmt.Printf("Failed to connect to peer %s: %v\n", addr, err)
				return
			}

			p.mu.Lock()
			p.Peers[addr] = client
			p.mu.Unlock()

			fmt.Printf("Successfully connected to peer %s\n", addr)
		}(peerAddr)

		connected++

		// Small delay between connection attempts
		time.Sleep(100 * time.Millisecond)
	}
}

// GetConnectedPeers returns the number of connected peers
func (p *Pool) GetConnectedPeers() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.Peers)
}

// GetPeer returns a specific peer client
func (p *Pool) GetPeer(addr string) (*Client, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if client, exists := p.Peers[addr]; exists {
		return client, nil
	}
	return nil, fmt.Errorf("peer %s not found", addr)
}

// ClosePeer closes a connection to a specific peer
func (p *Pool) ClosePeer(addr string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if client, exists := p.Peers[addr]; exists {
		client.Close()
		delete(p.Peers, addr)
	}
}

// CloseAll closes all peer connections
func (p *Pool) CloseAll() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for addr, client := range p.Peers {
		client.Close()
		delete(p.Peers, addr)
	}
}

// GetPeers returns a copy of the peers map
func (p *Pool) GetPeers() map[string]*Client {
	p.mu.Lock()
	defer p.mu.Unlock()
	peers := make(map[string]*Client)
	for k, v := range p.Peers {
		peers[k] = v
	}
	return peers
}

package peer

import (
	"fmt"
	"sync"
	"time"

	"github.com/piyushgupta53/go-torrent/internal/tracker"
)

// Pool manages multiple peer sessions
type Pool struct {
	InfoHash  [20]byte
	OurPeerID [20]byte
	Sessions  map[string]*Session
	mu        sync.Mutex
}

// NewPool creates a new peer connection pool
func NewPool(infoHash, ourPeerID [20]byte) *Pool {
	return &Pool{
		InfoHash:  infoHash,
		OurPeerID: ourPeerID,
		Sessions:  make(map[string]*Session),
	}
}

// Connect attempts to connect to a list of peers
func (p *Pool) Connect(peers []tracker.Peer, maxConnections int) int {
	connected := 0

	for _, peer := range peers {
		if connected >= maxConnections {
			break
		}

		peerAddr := peer.String()

		// Skip if already connected
		p.mu.Lock()
		if _, exists := p.Sessions[peerAddr]; exists {
			p.mu.Unlock()
			continue
		}
		p.mu.Unlock()

		// Try to connect
		session, err := NewSession(peerAddr, p.InfoHash, p.OurPeerID)
		if err != nil {
			fmt.Printf("Failed to connect to peer %s: %v\n", peerAddr, err)
			continue
		}

		// Start the session
		if err := session.Start(); err != nil {
			fmt.Printf("Failed to start session with %s: %v\n", peerAddr, err)
			session.Close()
			continue
		}

		p.mu.Lock()
		p.Sessions[peerAddr] = session
		p.mu.Unlock()

		fmt.Printf("Successfully connected to peer %s\n", peerAddr)
		connected++

		// Small delay between connection attempts
		time.Sleep(100 * time.Millisecond)
	}

	return connected
}

// GetConnectedPeers returns the number of connected peers
func (p *Pool) GetConnectedPeers() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.Sessions)
}

// GetSession returns a specific peer session
func (p *Pool) GetSession(addr string) (*Session, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	session, exists := p.Sessions[addr]
	return session, exists
}

// GetUnchokedSessions returns all sessions that are not choked
func (p *Pool) GetUnchokedSessions() []*Session {
	p.mu.Lock()
	defer p.mu.Unlock()

	var unchoked []*Session
	for _, session := range p.Sessions {
		if !session.IsChoked() {
			unchoked = append(unchoked, session)
		}
	}

	return unchoked
}

// GetSessionsWithPiece returns all sessions that have a specific piece
func (p *Pool) GetSessionsWithPiece(pieceIndex int) []*Session {
	p.mu.Lock()
	defer p.mu.Unlock()

	var sessions []*Session
	for _, session := range p.Sessions {
		if session.HasPiece(pieceIndex) {
			sessions = append(sessions, session)
		}
	}

	return sessions
}

// CloseSession closes a connection to a specific peer
func (p *Pool) CloseSession(addr string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if session, exists := p.Sessions[addr]; exists {
		session.Close()
		delete(p.Sessions, addr)
	}
}

// CloseAll closes all peer connections
func (p *Pool) CloseAll() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for addr, session := range p.Sessions {
		session.Close()
		delete(p.Sessions, addr)
	}
}

// GetPeers returns all peer sessions
func (p *Pool) GetPeers() map[string]*Session {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.Sessions
}

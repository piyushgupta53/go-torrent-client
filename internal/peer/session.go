package peer

import (
	"fmt"
	"sync"
	"time"
)

// Session represents an active session with a peer
type Session struct {
	client  *Client
	handler *MessageHandler
	addr    string
	mu      sync.Mutex
}

// NewSession creates a new peer session
func NewSession(peerAdrr string, infoHash, ourPeerID [20]byte) (*Session, error) {
	client, err := NewClient(peerAdrr, infoHash, ourPeerID)
	if err != nil {
		return nil, err
	}

	handler := NewMessageHandler(client)

	return &Session{
		client:  client,
		handler: handler,
		addr:    peerAdrr,
	}, nil
}

// Start begins the session
func (s *Session) Start() error {
	// Send interested message
	if err := s.client.SendInterested(); err != nil {
		return fmt.Errorf("failed to send interested: %w", err)
	}

	// Start the message handler's processing loop
	s.handler.Start()

	// Start a goroutine to keep the connection alive
	go s.keepAliveRoutine()

	return nil
}

// keepAliveRoutine sends periodic keep-alive messages
func (s *Session) keepAliveRoutine() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		if err := s.client.SendKeepAlive(); err != nil {
			fmt.Printf("Failed to send keep-alive to %s: %v\n", s.addr, err)
			s.mu.Unlock()
			return
		}
		s.mu.Unlock()
	}
}

// IsChoked returns whether we're choked by this peer
func (s *Session) IsChoked() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.client.Choked
}

// HasPiece returns whether the peer has a specific piece
func (s *Session) HasPiece(index int) bool {
	return s.handler.HasPiece(index)
}

// RequestBlock requests a block from the peer
func (s *Session) RequestBlock(index, begin, length int) error {
	return s.handler.RequestPiece(index, begin, length)
}

// SetOnUnchoke sets the callback for when we're unchoked
func (s *Session) SetOnUnchoke(callback func()) {
	s.handler.SetOnUnchoke(callback)
}

// SetOnPiece sets the callback for when we receive a piece
func (s *Session) SetOnPiece(callback func(*Piece)) {
	s.handler.SetOnPiece(callback)
}

// Close closes the session
func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.client.Close()
}

// String returns a string representation of the session
func (s *Session) String() string {
	return fmt.Sprintf("Session{addr=%s, choked=%v}", s.addr, s.IsChoked())
}

// SendInterested sends an interested message to the peer
func (s *Session) SendInterested() error {
	return s.client.SendInterested()
}

// Read reads a message from the peer
func (s *Session) Read() (*Message, error) {
	return s.client.Read()
}

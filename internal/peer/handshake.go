package peer

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"time"
)

// Handshake represents a BitTorrent handshake message
type Handshake struct {
	ProtocolLen byte
	Protocol    [19]byte
	Reserved    [8]byte
	InfoHash    [20]byte
	PeerID      [20]byte
}

// New creates a new handshake message
func NewHandshake(infoHash, peerID [20]byte) *Handshake {
	return &Handshake{
		ProtocolLen: 19,
		Protocol:    [19]byte{'B', 'i', 't', 'T', 'o', 'r', 'r', 'e', 'n', 't', ' ', 'p', 'r', 'o', 't', 'o', 'c', 'o', 'l'},
		Reserved:    [8]byte{0, 0, 0, 0, 0, 0, 0, 0}, // No extensions for now
		InfoHash:    infoHash,
		PeerID:      peerID,
	}
}

// Serialize converts the handshake to bytes for sending
func (h *Handshake) Serialize() []byte {
	buf := make([]byte, 68)

	buf[0] = h.ProtocolLen
	copy(buf[1:20], h.Protocol[:])
	copy(buf[20:28], h.Reserved[:])
	copy(buf[28:48], h.InfoHash[:])
	copy(buf[48:68], h.PeerID[:])

	return buf
}

// Read reads a handshake from an io.Reader
func Read(r io.Reader) (*Handshake, error) {
	lengthBuf := make([]byte, 1)

	_, err := io.ReadFull(r, lengthBuf)
	if err != nil {
		return nil, err
	}

	protocolLen := lengthBuf[0]
	if protocolLen != 19 {
		return nil, fmt.Errorf("invalid protocol length: %d", protocolLen)
	}

	// Read the rest of the handshake
	buf := make([]byte, 67)
	_, err = io.ReadFull(r, buf)
	if err != nil {
		return nil, err
	}

	handshake := &Handshake{
		ProtocolLen: protocolLen,
	}

	copy(handshake.Protocol[:], buf[0:19])
	copy(handshake.Reserved[:], buf[19:27])
	copy(handshake.InfoHash[:], buf[27:47])
	copy(handshake.PeerID[:], buf[47:67])

	// Verify protocol string
	expectedProtocol := "BitTorrent protocol"
	if string(handshake.Protocol[:]) != expectedProtocol {
		return nil, fmt.Errorf("invalid protocol: %s", string(handshake.Protocol[:]))
	}

	return handshake, nil
}

// Validate checks if the handshake is valid for our torrent
func (h *Handshake) Validate(expectedInfoHash [20]byte) error {
	if !bytes.Equal(h.InfoHash[:], expectedInfoHash[:]) {
		return fmt.Errorf("info hash mismatch: got %x, want %x", h.InfoHash, expectedInfoHash)
	}

	return nil
}

// DoHandshake performs a complete handshake with a peer
func DoHandshake(conn net.Conn, infoHash, peerID [20]byte) (*Handshake, error) {
	// Set a timeout for handshake
	conn.SetDeadline(time.Now().Add(30 * time.Second))
	defer conn.SetDeadline(time.Time{}) // remove deadline after handshake

	// Create and send our handshake
	handshake := NewHandshake(infoHash, peerID)
	_, err := conn.Write(handshake.Serialize())
	if err != nil {
		return nil, fmt.Errorf("failed to send handshake: %w", err)
	}

	// Read the peer's handshake
	peerHandshake, err := Read(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to read handshake: %w", err)
	}

	// Validate the handshake
	if err := peerHandshake.Validate(infoHash); err != nil {
		return nil, fmt.Errorf("handshake validation failed: %w", err)
	}

	return peerHandshake, nil
}

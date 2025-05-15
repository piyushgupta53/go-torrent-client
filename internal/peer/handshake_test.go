// internal/peer/handshake_test.go
package peer

import (
	"bytes"
	"testing"
)

func TestHandshake(t *testing.T) {
	infoHash := [20]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	peerID := [20]byte{20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1}

	// Create a handshake
	handshake := NewHandshake(infoHash, peerID)

	// Serialize it
	data := handshake.Serialize()

	// Expected length: 1 + 19 + 8 + 20 + 20 = 68
	if len(data) != 68 {
		t.Errorf("Serialized handshake length = %d, want 68", len(data))
	}

	// Read it back
	reader := bytes.NewReader(data)
	readHandshake, err := Read(reader)
	if err != nil {
		t.Fatalf("Failed to read handshake: %v", err)
	}

	// Verify fields
	if readHandshake.ProtocolLen != 19 {
		t.Errorf("Protocol length = %d, want 19", readHandshake.ProtocolLen)
	}

	expectedProtocol := "BitTorrent protocol"
	if string(readHandshake.Protocol[:]) != expectedProtocol {
		t.Errorf("Protocol = %s, want %s", string(readHandshake.Protocol[:]), expectedProtocol)
	}

	if !bytes.Equal(readHandshake.InfoHash[:], infoHash[:]) {
		t.Errorf("InfoHash mismatch")
	}

	if !bytes.Equal(readHandshake.PeerID[:], peerID[:]) {
		t.Errorf("PeerID mismatch")
	}
}

func TestHandshakeValidation(t *testing.T) {
	infoHash := [20]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	wrongInfoHash := [20]byte{0, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	peerID := [20]byte{20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1}

	handshake := NewHandshake(infoHash, peerID)

	// Valid validation
	if err := handshake.Validate(infoHash); err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}

	// Invalid validation
	if err := handshake.Validate(wrongInfoHash); err == nil {
		t.Errorf("Validate() error = nil, want error")
	}
}

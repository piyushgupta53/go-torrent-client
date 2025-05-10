package tracker

import (
	"crypto/rand"
	"fmt"
)

// GeneratePeerID generates a unique peer ID for our client
// Format: -GT0001-[12 random bytes]
// GT = GoTorrent, 0001 = version
func GeneratePeerID() ([20]byte, error) {
	peerID := [20]byte{}

	// Client identifier prefix
	prefix := "-GT0001-"
	copy(peerID[:], []byte(prefix))

	// Generate random bytes for the rest
	_, err := rand.Read(peerID[len(prefix):])
	if err != nil {
		return peerID, fmt.Errorf("failed to generate peer ID: %w", err)
	}

	return peerID, nil
}

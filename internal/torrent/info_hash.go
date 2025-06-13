package torrent

import (
	"bytes"
	"crypto/sha1"
	"fmt"

	"github.com/piyushgupta53/go-torrent/internal/bencode"
)

// calculateHashInfo computes the SHA-1 hash of the bencoded info dictionary
func calculateHashInfo(info map[string]any) ([20]byte, error) {
	var buf bytes.Buffer

	// Re-encode the info dictionary to get its exact bencoded representation
	err := bencode.Encode(&buf, info)
	if err != nil {
		return [20]byte{}, fmt.Errorf("failed to encode info dictionary: %w", err)
	}

	// calculate the SHA-1 hash
	return sha1.Sum(buf.Bytes()), nil
}

package tracker

import (
	"bytes"
	"net"
	"reflect"
	"testing"

	"github.com/piyushgupta53/go-torrent/internal/bencode"
)

func TestParseCompactPeers(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected []Peer
		wantErr  bool
	}{
		{
			name: "Valid compact peers",
			data: []byte{
				127, 0, 0, 1, 0x1A, 0xE1, // 127.0.0.1:6881
				192, 168, 1, 1, 0x1F, 0x90, // 192.168.1.1:8080
			},
			expected: []Peer{
				{IP: net.IPv4(127, 0, 0, 1), Port: 6881},
				{IP: net.IPv4(192, 168, 1, 1), Port: 8080},
			},
			wantErr: false,
		},
		{
			name:     "Invalid length",
			data:     []byte{127, 0, 0, 1, 0x1A}, // 5 bytes instead of 6
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCompactPeers(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseCompactPeers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("parseCompactPeers() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseAnnounceResponse(t *testing.T) {
	// Create a mock tracker response
	compactResponse := map[string]any{
		"interval":   int64(1800),
		"complete":   int64(5),
		"incomplete": int64(3),
		"peers":      string([]byte{127, 0, 0, 1, 0x1A, 0xE1}), // Compact format
	}

	nonCompactResponse := map[string]any{
		"interval":   int64(1800),
		"complete":   int64(5),
		"incomplete": int64(3),
		"peers": []any{
			map[string]any{
				"peer id": "01234567890123456789",
				"ip":      "127.0.0.1",
				"port":    int64(6881),
			},
		},
	}

	errorResponse := map[string]any{
		"failure reason": "Invalid info_hash",
	}

	tests := []struct {
		name     string
		response map[string]any
		expected *AnnounceResponse
		wantErr  bool
	}{
		{
			name:     "Compact response",
			response: compactResponse,
			expected: &AnnounceResponse{
				Interval:   1800,
				Complete:   5,
				Incomplete: 3,
				Peers: []Peer{
					{IP: net.IPv4(127, 0, 0, 1), Port: 6881},
				},
			},
			wantErr: false,
		},
		{
			name:     "Non-compact response",
			response: nonCompactResponse,
			expected: &AnnounceResponse{
				Interval:   1800,
				Complete:   5,
				Incomplete: 3,
				Peers: []Peer{
					{
						ID:   [20]byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9'},
						IP:   net.ParseIP("127.0.0.1"),
						Port: 6881,
					},
				},
			},
			wantErr: false,
		},
		{
			name:     "Error response",
			response: errorResponse,
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode the response
			var buf bytes.Buffer
			err := bencode.Encode(&buf, tt.response)
			if err != nil {
				t.Fatalf("Failed to encode test response: %v", err)
			}

			// Parse the response
			got, err := parseAnnounceResponse(buf.Bytes())
			if (err != nil) != tt.wantErr {
				t.Errorf("parseAnnounceResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("parseAnnounceResponse() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGeneratePeerID(t *testing.T) {
	peerID, err := GeneratePeerID()
	if err != nil {
		t.Fatalf("GeneratePeerID() error = %v", err)
	}

	// Check that it starts with our client prefix
	expectedPrefix := "-GT0001-"
	if !bytes.HasPrefix(peerID[:], []byte(expectedPrefix)) {
		t.Errorf("PeerID doesn't start with expected prefix %s", expectedPrefix)
	}

	// Check that it's exactly 20 bytes
	if len(peerID) != 20 {
		t.Errorf("PeerID length = %d, want 20", len(peerID))
	}
}

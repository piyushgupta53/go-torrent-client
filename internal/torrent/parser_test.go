package torrent

import (
	"bytes"
	"crypto/sha1"
	"reflect"
	"testing"
	"time"

	"github.com/piyushgupta53/go-torrent/internal/bencode"
)

func TestParse(t *testing.T) {
	// Create a mock torrent file bencode data
	singleFileData := map[string]any{
		"announce":      "http://tracker.example.com/announce",
		"creation date": int64(1617235200), // April 1, 2021
		"comment":       "Test torrent",
		"created by":    "go-torrent",
		"info": map[string]any{
			"name":         "test.txt",
			"piece length": int64(16384),
			"pieces":       string(make([]byte, 60)), // 3 pieces (20 bytes each)
			"length":       int64(32768),
			"private":      int64(0),
		},
	}

	multiFileData := map[string]any{
		"announce": "http://tracker.example.com/announce",
		"announce-list": []any{
			[]any{"http://tracker1.example.com/announce", "http://tracker2.example.com/announce"},
			[]any{"http://tracker3.example.com/announce"},
		},
		"creation date": int64(1617235200), // April 1, 2021
		"comment":       "Test torrent",
		"created by":    "go-torrent",
		"info": map[string]any{
			"name":         "test_dir",
			"piece length": int64(16384),
			"pieces":       string(make([]byte, 60)), // 3 pieces (20 bytes each)
			"files": []any{
				map[string]any{
					"length": int64(12345),
					"path":   []any{"file1.txt"},
				},
				map[string]any{
					"length": int64(67890),
					"path":   []any{"subdir", "file2.txt"},
				},
			},
			"private": int64(1),
		},
	}

	tests := []struct {
		name     string
		data     map[string]any
		expected *TorrentFile
		wantErr  bool
	}{
		{
			name: "Single File Test",
			data: singleFileData,
			expected: &TorrentFile{
				Announce:     "http://tracker.example.com/announce",
				CreationDate: time.Unix(1617235200, 0),
				Comment:      "Test torrent",
				CreatedBy:    "go-torrent",
				Info: InfoDict{
					PieceLength: 16384,
					Pieces:      string(make([]byte, 60)),
					Private:     false,
					Name:        "test.txt",
					Length:      32768,
					IsDirectory: false,
				},
				PiecesHash: [][20]byte{
					{}, // Empty hash for test
					{}, // Empty hash for test
					{}, // Empty hash for test
				},
			},
			wantErr: false,
		},
		{
			name: "Multi File Test",
			data: multiFileData,
			expected: &TorrentFile{
				Announce: "http://tracker.example.com/announce",
				AnnouceList: [][]string{
					{"http://tracker1.example.com/announce", "http://tracker2.example.com/announce"},
					{"http://tracker3.example.com/announce"},
				},
				CreationDate: time.Unix(1617235200, 0),
				Comment:      "Test torrent",
				CreatedBy:    "go-torrent",
				Info: InfoDict{
					PieceLength: 16384,
					Pieces:      string(make([]byte, 60)),
					Private:     true,
					Name:        "test_dir",
					Files: []FileDict{
						{
							Length: 12345,
							Path:   []string{"file1.txt"},
						},
						{
							Length: 67890,
							Path:   []string{"subdir", "file2.txt"},
						},
					},
					IsDirectory: true,
				},
				PiecesHash: [][20]byte{
					{}, // Empty hash for test
					{}, // Empty hash for test
					{}, // Empty hash for test
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip setting InfoHash in expected data as it's calculated during Parse

			// Parse the test data
			got, err := Parse(tt.data)

			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Skip comparing InfoHash in the test
				got.InfoHash = [20]byte{}

				if !reflect.DeepEqual(got, tt.expected) {
					t.Errorf("Parse() = %v, want %v", got, tt.expected)
				}
			}
		})
	}
}

func TestCalculateInfoHash(t *testing.T) {
	// Create a simple info dictionary
	info := map[string]any{
		"name":         "test.txt",
		"piece length": int64(16384),
		"pieces":       "abcdefghijklmnopqrst", // 20 bytes
		"length":       int64(32768),
	}

	// Expected hash calculation process:
	// 1. Bencode the info dictionary
	var buf bytes.Buffer
	err := bencode.Encode(&buf, info)
	if err != nil {
		t.Fatalf("Failed to encode info dictionary: %v", err)
	}

	// Calculate the hash manually to compare
	expectedHash := sha1.Sum(buf.Bytes())

	// Get the hash using our function
	hash, err := calculateHashInfo(info)
	if err != nil {
		t.Fatalf("calculateHashInfo() error = %v", err)
	}

	// Compare the expected and actual hash
	if !bytes.Equal(hash[:], expectedHash[:]) {
		t.Errorf("calculateHashInfo() = %x, want %x", hash, expectedHash)
	}
}

func TestTorrentFileHelpers(t *testing.T) {
	// Create a test torrent file
	torrent := &TorrentFile{
		Info: InfoDict{
			PieceLength: 16384,
			IsDirectory: true,
			Name:        "test_dir",
			Files: []FileDict{
				{
					Length: 10000,
					Path:   []string{"file1.txt"},
				},
				{
					Length: 20000,
					Path:   []string{"subdir", "file2.txt"},
				},
			},
		},
		PiecesHash: make([][20]byte, 3), // 3 pieces
	}

	// Test TotalLength
	expectedTotal := int64(30000) // 10000 + 20000
	if got := torrent.TotalLength(); got != expectedTotal {
		t.Errorf("TotalLength() = %v, want %v", got, expectedTotal)
	}

	// Test NumPieces
	expectedPieces := 3
	if got := torrent.NumPieces(); got != expectedPieces {
		t.Errorf("NumPieces() = %v, want %v", got, expectedPieces)
	}

	// Test PieceSize
	expectedSize1 := int64(16384) // Regular piece
	if got := torrent.PieceSize(0); got != expectedSize1 {
		t.Errorf("PieceSize(0) = %v, want %v", got, expectedSize1)
	}

	expectedSizeLast := int64(13616) // Last piece: 30000 - 2*16384 = 13616
	if got := torrent.PieceSize(2); got != expectedSizeLast {
		t.Errorf("PieceSize(2) = %v, want %v", got, expectedSizeLast)
	}

	// Test FilePathForPiece
	// We need a more detailed calculation for this test, but here's a simple one:
	expectedPaths := []string{"test_dir/file1.txt", "test_dir/subdir/file2.txt"}
	if got := torrent.FilePathForPiece(1); !reflect.DeepEqual(got, expectedPaths) {
		t.Errorf("FilePathForPiece(1) = %v, want %v", got, expectedPaths)
	}
}

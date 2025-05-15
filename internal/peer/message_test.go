// internal/peer/message_test.go
package peer

import (
	"bytes"
	"testing"
)

func TestMessage(t *testing.T) {
	tests := []struct {
		name    string
		message *Message
		want    []byte
	}{
		{
			name:    "Keep-alive",
			message: nil,
			want:    []byte{0, 0, 0, 0},
		},
		{
			name:    "Choke",
			message: &Message{ID: MsgChoke},
			want:    []byte{0, 0, 0, 1, 0},
		},
		{
			name:    "Unchoke",
			message: &Message{ID: MsgUnchoke},
			want:    []byte{0, 0, 0, 1, 1},
		},
		{
			name:    "Interested",
			message: &Message{ID: MsgInterested},
			want:    []byte{0, 0, 0, 1, 2},
		},
		{
			name:    "Not Interested",
			message: &Message{ID: MsgNotInterested},
			want:    []byte{0, 0, 0, 1, 3},
		},
		{
			name:    "Have",
			message: &Message{ID: MsgHave, Payload: []byte{0, 0, 0, 5}},
			want:    []byte{0, 0, 0, 5, 4, 0, 0, 0, 5},
		},
		{
			name:    "Request",
			message: &Message{ID: MsgRequest, Payload: SerializeRequest(0, 0, 16384)},
			want:    []byte{0, 0, 0, 13, 6, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x40, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.message.Serialize()
			if !bytes.Equal(got, tt.want) {
				t.Errorf("Serialize() = %v, want %v", got, tt.want)
			}

			// Test round-trip
			reader := bytes.NewReader(got)
			readMsg, err := ReadMessage(reader)
			if err != nil {
				t.Errorf("ReadMessage() error = %v", err)
				return
			}

			if tt.message == nil {
				if readMsg != nil {
					t.Errorf("ReadMessage() = %v, want nil", readMsg)
				}
			} else {
				if readMsg.ID != tt.message.ID {
					t.Errorf("ReadMessage() ID = %v, want %v", readMsg.ID, tt.message.ID)
				}
				if !bytes.Equal(readMsg.Payload, tt.message.Payload) {
					t.Errorf("ReadMessage() Payload = %v, want %v", readMsg.Payload, tt.message.Payload)
				}
			}
		})
	}
}

func TestBitfield(t *testing.T) {
	// Create a bitfield for 20 pieces
	bf := make(Bitfield, 3) // 3 bytes = 24 bits (for up to 24 pieces)

	// Test setting and checking pieces
	bf.SetPiece(0)
	bf.SetPiece(5)
	bf.SetPiece(19)

	testCases := []struct {
		piece int
		want  bool
	}{
		{0, true},
		{1, false},
		{5, true},
		{19, true},
		{20, false},
	}

	for _, tc := range testCases {
		if got := bf.HasPiece(tc.piece); got != tc.want {
			t.Errorf("HasPiece(%d) = %v, want %v", tc.piece, got, tc.want)
		}
	}
}

func TestRequestParsing(t *testing.T) {
	req := &Request{
		Index:  5,
		Begin:  16384,
		Length: 16384,
	}

	payload := SerializeRequest(req.Index, req.Begin, req.Length)

	parsed, err := ParseRequest(payload)
	if err != nil {
		t.Fatalf("ParseRequest() error = %v", err)
	}

	if parsed.Index != req.Index || parsed.Begin != req.Begin || parsed.Length != req.Length {
		t.Errorf("ParseRequest() = %v, want %v", parsed, req)
	}
}

func TestPieceParsing(t *testing.T) {
	block := []byte("This is some block data")
	piece := &Piece{
		Index: 3,
		Begin: 32768,
		Block: block,
	}

	payload := SerializePiece(piece.Index, piece.Begin, piece.Block)

	parsed, err := ParsePiece(payload)
	if err != nil {
		t.Fatalf("ParsePiece() error = %v", err)
	}

	if parsed.Index != piece.Index || parsed.Begin != piece.Begin || !bytes.Equal(parsed.Block, piece.Block) {
		t.Errorf("ParsePiece() = %v, want %v", parsed, piece)
	}
}

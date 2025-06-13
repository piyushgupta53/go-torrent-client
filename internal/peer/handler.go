package peer

import (
	"encoding/binary"
	"fmt"
	"sync"
)

// MessageHandler handles incoming messages from a peer
type MessageHandler struct {
	client    *Client
	pieces    map[int]bool
	mu        sync.RWMutex
	onUnchoke func()
	onPiece   func(*Piece)
}

// NewMessageHandler creates a new message handler
func NewMessageHandler(client *Client) *MessageHandler {
	return &MessageHandler{
		client: client,
		pieces: make(map[int]bool),
	}
}

// Start begins handling messages from the peer
func (h *MessageHandler) Start() {
	go h.messageLoop()
}

// messageLoop continuously reads and processes messages
func (h *MessageHandler) messageLoop() {
	for {
		msg, err := h.client.Read()
		if err != nil {
			fmt.Printf("Error reading from peer: %v\n", err)
			return
		}

		if err := h.handleMessage(msg); err != nil {
			fmt.Printf("Error handling message: %v\n", err)
		}
	}
}

// handleMessage processes a single message
func (h *MessageHandler) handleMessage(msg *Message) error {
	if msg == nil {
		// keep alive
		return nil
	}

	switch msg.ID {
	case MsgUnchoke:
		h.client.Choked = false
		fmt.Println("Peer unchoked us")
		if h.onUnchoke != nil {
			h.onUnchoke()
		}

	case MsgInterested:
		fmt.Println("Peer is interested")
		// For now, we can unchoke them
		return h.client.SendUnchoke()

	case MsgNotInterested:
		fmt.Println("Peer is not interested")

	case MsgHave:
		if len(msg.Payload) != 4 {
			return fmt.Errorf("invalid have message length")
		}

		pieceIndex := int(binary.BigEndian.Uint32(msg.Payload))
		h.mu.Lock()
		h.pieces[pieceIndex] = true
		h.mu.Unlock()
		fmt.Printf("Peer has piece %d\n", pieceIndex)

	case MsgBitfield:
		h.client.Bitfield = Bitfield(msg.Payload)
		fmt.Printf("Received bitfield (%d bytes)\n", len(msg.Payload))

		// Update our pieces map
		h.mu.Lock()
		for i := range len(msg.Payload) * 8 {
			if h.client.Bitfield.HasPiece(i) {
				h.pieces[i] = true
			}
		}
		h.mu.Unlock()

	case MsgRequest:
		req, err := ParseRequest(msg.Payload)
		if err != nil {
			return fmt.Errorf("invalid request: %w", err)
		}

		fmt.Printf("Peer requested piece %d, begin %d, length %d\n",
			req.Index, req.Begin, req.Length)
		// We would need to handle uploading here

	case MsgPiece:
		piece, err := ParsePiece(msg.Payload)
		if err != nil {
			return fmt.Errorf("invalid piece: %w", err)
		}
		fmt.Printf("Received piece %d, begin %d, length %d\n",
			piece.Index, piece.Begin, len(piece.Block))
		if h.onPiece != nil {
			h.onPiece(piece)
		}

	case MsgCancel:
		req, err := ParseRequest(msg.Payload)
		if err != nil {
			return fmt.Errorf("invalid cancel: %w", err)
		}
		fmt.Printf("Peer cancelled request for piece %d, begin %d, length %d\n",
			req.Index, req.Begin, req.Length)

	default:
		fmt.Printf("Unknown message type: %d\n", msg.ID)
	}

	return nil
}

// HasPiece returns true if the peer has a specific piece
func (h *MessageHandler) HasPiece(index int) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.pieces[index]
}

// RequestPiece requests a block from the peer
func (h *MessageHandler) RequestPiece(index, begin, length int) error {
	if h.client.Choked {
		return fmt.Errorf("cannot request piece: we are choked")
	}

	if !h.HasPiece(index) {
		return fmt.Errorf("peer doesn't have piece %d", index)
	}

	return h.client.SendRequest(index, begin, length)
}

// SetOnUnchoke sets the callback for when we're unchoked
func (h *MessageHandler) SetOnUnchoke(callback func()) {
	h.onUnchoke = callback
}

// SetOnPiece sets the callback for when we receive a piece
func (h *MessageHandler) SetOnPiece(callback func(*Piece)) {
	h.onPiece = callback
}

package peer

import (
	"encoding/binary"
	"fmt"
	"io"
)

type MessageID uint8

const (
	MsgChoke         MessageID = 0
	MsgUnchoke       MessageID = 1
	MsgInterested    MessageID = 2
	MsgNotInterested MessageID = 3
	MsgHave          MessageID = 4
	MsgBitfield      MessageID = 5
	MsgRequest       MessageID = 6
	MsgPiece         MessageID = 7
	MsgCancel        MessageID = 8
)

// Message represents a peer wire protocol
type Message struct {
	ID      MessageID
	Payload []byte
}

// Serialize converts a message to bytes for sending
func (m *Message) Serialize() []byte {
	if m == nil {
		// Keep-alive message (length = 0)
		return make([]byte, 4)
	}

	length := uint32(1 + len(m.Payload))
	buf := make([]byte, 1+length)

	binary.BigEndian.PutUint32(buf[0:4], length)
	buf[4] = byte(m.ID)
	copy(buf[5:], m.Payload)

	return buf
}

// Read reads a message from an io.Reader
func ReadMessage(r io.Reader) (*Message, error) {
	// Read the message length
	lengthBuf := make([]byte, 4)
	_, err := io.ReadFull(r, lengthBuf)
	if err != nil {
		return nil, err
	}

	length := binary.BigEndian.Uint32(lengthBuf)

	// Kee-alive message (length = 0)
	if length == 0 {
		return nil, nil
	}

	// Read the message ID and payload
	messageBuf := make([]byte, length)
	_, err = io.ReadFull(r, messageBuf)
	if err != nil {
		return nil, err
	}

	message := &Message{
		ID:      MessageID(messageBuf[0]),
		Payload: messageBuf[1:],
	}

	return message, nil
}

// String returns a string representation of the message
func (m *Message) String() string {
	if m == nil {
		return "keep-alive"
	}

	switch m.ID {
	case MsgChoke:
		return "choke"
	case MsgUnchoke:
		return "unchoke"
	case MsgInterested:
		return "interested"
	case MsgNotInterested:
		return "not interested"
	case MsgHave:
		return fmt.Sprintf("have (piece %d)", binary.BigEndian.Uint32(m.Payload))
	case MsgBitfield:
		return "bitfield"
	case MsgRequest:
		return "request"
	case MsgPiece:
		return "piece"
	case MsgCancel:
		return "cancel"
	default:
		return fmt.Sprintf("unknown (ID: %d)", m.ID)
	}
}

// Request represents a block request message
type Request struct {
	Index  int // Piece index
	Begin  int // Byte offset within the piece
	Length int // Length of the block
}

// ParseRequest parses a request message payload
func ParseRequest(payload []byte) (*Request, error) {
	if len(payload) != 12 {
		return nil, fmt.Errorf("invalid request payload length: %d", len(payload))
	}

	index := binary.BigEndian.Uint32(payload[0:4])
	begin := binary.BigEndian.Uint32(payload[4:8])
	length := binary.BigEndian.Uint32(payload[8:12])

	request := &Request{
		Index:  int(index),
		Begin:  int(begin),
		Length: int(length),
	}

	return request, nil
}

// SerializeRequest creates a request message payload
func SerializeRequest(index, begin, length int) []byte {
	payload := make([]byte, 12)
	binary.BigEndian.PutUint32(payload[0:4], uint32(index))
	binary.BigEndian.PutUint32(payload[4:8], uint32(begin))
	binary.BigEndian.PutUint32(payload[8:12], uint32(length))

	return payload
}

// Piece represents a piece message with block data
type Piece struct {
	Index int    // Piece index
	Begin int    // Byte offset within that piece
	Block []byte // Block data
}

// ParsePiece parses a piece message payload
func ParsePiece(payload []byte) (*Piece, error) {
	if len(payload) < 8 {
		return nil, fmt.Errorf("invalid piece payload length: %d", len(payload))
	}

	piece := &Piece{
		Index: int(binary.BigEndian.Uint32(payload[:4])),
		Begin: int(binary.BigEndian.Uint32(payload[4:8])),
		Block: payload[8:],
	}

	return piece, nil
}

// SerializePiece creates a piece message payload
func SerializePiece(index, begin int, block []byte) []byte {
	payload := make([]byte, 8+len(block))
	binary.BigEndian.PutUint32(payload[:4], uint32(index))
	binary.BigEndian.PutUint32(payload[4:8], uint32(begin))
	copy(payload[8:], block)

	return payload
}

// Bitfield message
type Bitfield []byte

// HasPiece returns true if the bitfield indicates having a piece
func (bf Bitfield) HasPiece(index int) bool {
	if index < 0 || index >= len(bf)*8 {
		return false
	}

	byteIndex := index / 8
	offset := index % 8

	return bf[byteIndex]>>(7-offset)&1 != 0
}

// SetPiece sets a piece as available in the bitfield
func (bf Bitfield) SetPiece(index int) {
	if index < 0 || index >= len(bf)*8 {
		return
	}

	byteIndex := index / 8
	offset := index % 8

	bf[byteIndex] |= 1 << (7 - offset)
}

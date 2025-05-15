package peer

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

// Client represents a connection to a peer
type Client struct {
	Conn     net.Conn
	PeerID   [20]byte
	InfoHash [20]byte
	Choked   bool
	Bitfield Bitfield
}

// NewClient creates a new peer connection
func NewClient(peerAddr string, infoHash, ourPeerID [20]byte) (*Client, error) {
	// Set timeout for connection
	conn, err := net.DialTimeout("tcp", peerAddr, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to peer %s: %w", peerAddr, err)
	}

	// Perform handshake
	peerHandshake, err := DoHandshake(conn, infoHash, ourPeerID)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("handshake failed with %s: %w", peerAddr, err)
	}

	client := &Client{
		Conn:     conn,
		PeerID:   peerHandshake.PeerID,
		InfoHash: infoHash,
		Choked:   true,
	}

	// Read bitfield if peer sends it
	if err := client.readBitfield(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to read bitfield: %w", err)
	}

	return client, nil
}

// readBitfield reads the initial bitfield message if present
func (c *Client) readBitfield() error {
	// Set a short timeout for the bitfield message
	c.Conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	defer c.Conn.SetReadDeadline(time.Time{})

	msg, err := ReadMessage(c.Conn)
	if err != nil {
		// Timeout is ok - peer might not send bitfield
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return nil
		}

		return err
	}

	if msg == nil {
		// Keep-alive message
		return nil
	}

	if msg.ID == MsgBitfield {
		c.Bitfield = Bitfield(msg.Payload)
	}

	return nil
}

// SendMessage sends a message to the peer
func (c *Client) SendMessage(msg *Message) error {
	c.Conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
	_, err := c.Conn.Write(msg.Serialize())
	return err
}

// SendInterested sends an interested message
func (c *Client) SendInterested() error {
	return c.SendMessage(&Message{ID: MsgInterested})
}

// SendNotInterested sends a not interested message
func (c *Client) SendNotInterested() error {
	return c.SendMessage(&Message{ID: MsgNotInterested})
}

// SendUnchoke sends an unchoke message
func (c *Client) SendUnchoke() error {
	return c.SendMessage(&Message{ID: MsgUnchoke})
}

// SendRequest sends a request for a block
func (c *Client) SendRequest(index, begin, length int) error {
	payload := SerializeRequest(index, begin, length)
	return c.SendMessage(&Message{
		ID:      MsgRequest,
		Payload: payload,
	})
}

// SendHave sends a have message for a piece
func (c *Client) SendHave(index int) error {
	payload := make([]byte, 4)
	binary.BigEndian.PutUint32(payload, uint32(index))
	return c.SendMessage(&Message{
		ID:      MsgHave,
		Payload: payload,
	})
}

// SendKeepAlive sends a keep-alive message
func (c *Client) SendKeepAlive() error {
	_, err := c.Conn.Write(make([]byte, 4))
	return err
}

// Close closes the connection to the peer
func (c *Client) Close() error {
	return c.Conn.Close()
}

// Read reads a message from the peer
func (c *Client) Read() (*Message, error) {
	c.Conn.SetReadDeadline(time.Now().Add(3 * time.Minute))
	return ReadMessage(c.Conn)
}

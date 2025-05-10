package tracker

import "net"

type Client struct {
	PeerID   [20]byte // Our unique peer ID
	HTTPPort int      // Port we're listening on
}

func NewClient(peerID [20]byte, port int) *Client {
	return &Client{
		PeerID:   peerID,
		HTTPPort: port,
	}
}

// AnnounceRequest contains the parameters for a tracker announce request
type AnnounceRequest struct {
	InfoHash   [20]byte
	PeerID     [20]byte
	Port       int
	Uploaded   int64
	Downloaded int64
	Left       int64
	Compact    bool
	Event      string
}

// AnnounceResponse contains the response from a tracker
type AnnounceResponse struct {
	Interval   int
	Peers      []Peer
	Complete   int
	Incomplete int
}

type Peer struct {
	ID   [20]byte
	IP   net.IP
	Port int
}

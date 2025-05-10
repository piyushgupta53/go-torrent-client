package tracker

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/piyushgupta53/go-torrent/internal/torrent"
)

// DiscoverPeers contacts the tracker(s) to get a list of peers
func (c *Client) DiscoverPeers(torrent *torrent.TorrentFile) ([]Peer, error) {
	// Create announce request
	req := &AnnounceRequest{
		InfoHash:   torrent.InfoHash,
		PeerID:     c.PeerID,
		Port:       c.HTTPPort,
		Uploaded:   0,
		Downloaded: 0,
		Left:       torrent.TotalLength(),
		Compact:    true,
		Event:      "started",
	}

	// Contact the tracker
	response, err := c.Announce(torrent.Announce, req)
	if err != nil {
		return nil, fmt.Errorf("failed to announce to tracker: %w", err)
	}

	// Shuffle the peers for better distribution
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(response.Peers), func(i, j int) {
		response.Peers[i], response.Peers[j] = response.Peers[j], response.Peers[i]
	})

	return response.Peers, nil
}

// String returns a string representation of a peer
func (p *Peer) String() string {
	return fmt.Sprintf("%s:%d", p.IP.String(), p.Port)
}

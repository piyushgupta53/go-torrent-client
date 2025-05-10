package tracker

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/piyushgupta53/go-torrent/internal/bencode"
)

// Announce sends an announce request to the tracker and returns the response
func (c *Client) Announce(trackerURL string, req *AnnounceRequest) (*AnnounceResponse, error) {
	// Build the URL with the query parameters
	u, err := url.Parse(trackerURL)
	if err != nil {
		return nil, fmt.Errorf("invalid tracker URL: %w", err)
	}

	// Build query parameters
	params := url.Values{}

	params.Add("info_hash", string(req.InfoHash[:]))
	params.Add("peer_id", string(req.PeerID[:]))
	params.Add("port", strconv.Itoa(req.Port))
	params.Add("uploaded", strconv.FormatInt(req.Uploaded, 10))
	params.Add("downloaded", strconv.FormatInt(req.Downloaded, 10))
	params.Add("left", strconv.FormatInt(req.Left, 10))

	if req.Compact {
		params.Add("compact", "1")
	} else {
		params.Add("compact", "0")
	}

	if req.Event != "" {
		params.Add("event", req.Event)
	}

	u.RawQuery = params.Encode()

	// Create HTTP client with a timeout
	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	// Send the request
	resp, err := client.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("failed to contact tracker: %w", err)
	}

	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read tracker response: %w", err)
	}

	// Parse the response
	return parseAnnounceResponse(body)
}

// parseAnnounceResponse parses the bencode-encoded tracker response
func parseAnnounceResponse(data []byte) (*AnnounceResponse, error) {

	decoded, err := bencode.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode tracker response: %w", err)
	}

	dict, ok := decoded.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("tracker response is not a dictionary")
	}

	// Check for error from tracker
	if failureReason, ok := dict["failure reason"]; ok {
		reason, ok := failureReason.(string)
		if !ok {
			return nil, fmt.Errorf("invalid failure reason format")
		}

		return nil, fmt.Errorf("tracker error: %s", reason)
	}

	response := &AnnounceResponse{}

	// Parse interval
	if internalVal, ok := dict["interval"]; ok {
		interval, ok := internalVal.(int)
		if !ok {
			return nil, fmt.Errorf("invalid interval format")
		}

		response.Interval = int(interval)
	}

	// Parse complete count (seeders)
	if completeVal, ok := dict["complete"]; ok {
		complete, ok := completeVal.(int)
		if !ok {
			return nil, fmt.Errorf("invalid complete format")
		}

		response.Complete = int(complete)
	}

	// Parse peers
	if peersVal, ok := dict["peers"]; ok {
		switch peers := peersVal.(type) {
		case string:
			// Compact format (most comman)
			response.Peers, err = parseCompactPeers([]byte(peers))
			if err != nil {
				return nil, fmt.Errorf("failed to parse compact peers: %w", err)
			}
		case []interface{}:
			// Non-compact format
			response.Peers, err = parseNonCompactPeers(peers)
			if err != nil {
				return nil, fmt.Errorf("failed to parse non-compact peers: %w", err)
			}
		default:
			return nil, fmt.Errorf("invalid peers format")
		}
	}

	return response, nil
}

// parseCompactPeers parses the compact peers format
// Format: 6 bytes IP + 2 bytes port
func parseCompactPeers(data []byte) ([]Peer, error) {
	if len(data)%6 != 0 {
		return nil, fmt.Errorf("invalid compact peers length: %d", len(data))
	}

	numPeers := len(data) / 6
	peers := make([]Peer, numPeers)

	for i := 0; i < numPeers; i++ {
		offset := i * 6

		// Parse IP (4 bytes)
		ip := net.IP(data[offset : offset+4])

		// Parse port (2 bytes, big endian)
		port := binary.BigEndian.Uint16(data[offset+4 : offset+6])

		peers[i] = Peer{
			IP:   ip,
			Port: int(port),
		}
	}

	return peers, nil
}

// parseNonCompactPeers parses the non-compact peer format
func parseNonCompactPeers(data []interface{}) ([]Peer, error) {
	peers := make([]Peer, len(data))

	for i, peerData := range data {
		peerDict, ok := peerData.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("peer %d is not a dictionary", i)
		}

		// Parse Peer ID
		if peerIDVal, ok := peerDict["peer id"]; ok {
			peerIDStr, ok := peerIDVal.(string)
			if !ok {
				return nil, fmt.Errorf("peer %d has invalid peer id", i)
			}

			copy(peers[i].ID[:], []byte(peerIDStr))
		}

		// Parse IP
		ipVal, ok := peerDict["ip"]
		if !ok {
			return nil, fmt.Errorf("peer %d has invalid ip", i)
		}

		ipStr, ok := ipVal.(string)
		if !ok {
			return nil, fmt.Errorf("peer %d has invalid ip", i)
		}

		peers[i].IP = net.ParseIP(ipStr)
		if peers[i].IP == nil {
			return nil, fmt.Errorf("peer %d has invalid ip address: %s", i, ipStr)
		}

		// Parse Port
		portVal, ok := peerDict["port"]
		if !ok {
			return nil, fmt.Errorf("peer %d missing port", i)
		}

		port, ok := portVal.(int64)
		if !ok {
			return nil, fmt.Errorf("peer %d has invalid port", i)
		}

		peers[i].Port = int(port)
	}

	return peers, nil
}

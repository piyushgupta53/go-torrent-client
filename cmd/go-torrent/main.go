// cmd/go-torrent/main.go
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/piyushgupta53/go-torrent/internal/peer"
	"github.com/piyushgupta53/go-torrent/internal/torrent"
	"github.com/piyushgupta53/go-torrent/internal/tracker"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go-torrent <torrent-file>")
		os.Exit(1)
	}

	torrentPath := os.Args[1]

	// Parse the torrent file
	torrentFile, err := torrent.ParseFromFile(torrentPath)
	if err != nil {
		fmt.Printf("Error parsing torrent file: %v\n", err)
		os.Exit(1)
	}

	// Display torrent info
	fmt.Printf("Torrent: %s\n", filepath.Base(torrentPath))
	fmt.Printf("Announce URL: %s\n", torrentFile.Announce)
	fmt.Printf("Total Size: %d bytes\n", torrentFile.TotalLength())
	fmt.Printf("Pieces: %d (each %d bytes)\n", torrentFile.NumPieces(), torrentFile.Info.PieceLength)
	fmt.Printf("Info Hash: %x\n", torrentFile.InfoHash)

	// Generate peer ID
	peerID, err := tracker.GeneratePeerID()
	if err != nil {
		fmt.Printf("Error generating peer ID: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Our Peer ID: %x\n", peerID)

	// Create tracker client and discover peers
	trackerClient := tracker.NewClient(peerID, 6881)

	fmt.Println("\nDiscovering peers...")
	peers, err := trackerClient.DiscoverPeers(torrentFile)
	if err != nil {
		fmt.Printf("Error discovering peers: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found %d peers\n", len(peers))

	// Create peer connection pool
	peerPool := peer.NewPool(torrentFile.InfoHash, peerID)

	// Connect to peers
	fmt.Println("\nConnecting to peers...")
	peerPool.Connect(peers, 10) // Try to connect to up to 10 peers

	// Wait a bit for connections to establish
	time.Sleep(5 * time.Second)

	connectedPeers := peerPool.GetConnectedPeers()
	fmt.Printf("\nConnected to %d peers\n", connectedPeers)

	// If we have connections, test sending some messages
	if connectedPeers > 0 {
		fmt.Println("\nTesting peer communication...")

		// Send interested message to all connected peers
		for addr, session := range peerPool.GetPeers() {
			fmt.Printf("Sending interested to %s...\n", addr)
			if err := session.SendInterested(); err != nil {
				fmt.Printf("Error sending interested to %s: %v\n", addr, err)
			}

			// Try to read a message
			go func(addr string, session *peer.Session) {
				msg, err := session.Read()
				if err != nil {
					fmt.Printf("Error reading from %s: %v\n", addr, err)
					return
				}

				if msg == nil {
					fmt.Printf("Received keep-alive from %s\n", addr)
				} else {
					fmt.Printf("Received %s from %s\n", msg.String(), addr)

					// Handle unchoke message
					if msg.ID == peer.MsgUnchoke {
						fmt.Printf("Peer %s unchoked us!\n", addr)
					}
				}
			}(addr, session)
		}

		// Wait for responses
		time.Sleep(10 * time.Second)
	}

	// Cleanup
	fmt.Println("\nClosing connections...")
	peerPool.CloseAll()

	fmt.Println("Peer communication test complete. Next step would be downloading pieces.")
}

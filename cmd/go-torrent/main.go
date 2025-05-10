package main

import (
	"fmt"
	"os"
	"path/filepath"

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

	if torrentFile.Info.IsDirectory {
		fmt.Printf("Content: Directory (%s) with %d files\n", torrentFile.Info.Name, len(torrentFile.Info.Files))
	} else {
		fmt.Printf("Content: Single file (%s)\n", torrentFile.Info.Name)
	}

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

	// Create tracker client
	// Using port 6881 as default BitTorrent port
	trackerClient := tracker.NewClient(peerID, 6881)

	// Discover peers
	fmt.Println("\nDiscovering peers...")
	peers, err := trackerClient.DiscoverPeers(torrentFile)
	if err != nil {
		fmt.Printf("Error discovering peers: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found %d peers:\n", len(peers))
	for i, peer := range peers {
		fmt.Printf("  Peer %d: %s\n", i+1, peer.String())
		if i >= 10 { // Limit output to first 10 peers
			fmt.Printf("  ... and %d more\n", len(peers)-10)
			break
		}
	}

	fmt.Println("\nPeer discovery successful! Next step would be peer handshake and communication.")
}

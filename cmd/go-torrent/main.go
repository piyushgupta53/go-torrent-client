// cmd/go-torrent/main.go
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/piyushgupta53/go-torrent/internal/torrent"
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

	// Display information about the torrent
	fmt.Printf("Torrent: %s\n", filepath.Base(torrentPath))
	fmt.Printf("Announce URL: %s\n", torrentFile.Announce)

	if torrentFile.Info.IsDirectory {
		fmt.Printf("Content: Directory (%s) with %d files\n", torrentFile.Info.Name, len(torrentFile.Info.Files))
		fmt.Printf("Total Size: %d bytes\n", torrentFile.TotalLength())

		for i, file := range torrentFile.Info.Files {
			fmt.Printf("  File %d: %s (%d bytes)\n", i+1, filepath.Join(file.Path...), file.Length)
		}
	} else {
		fmt.Printf("Content: Single file (%s)\n", torrentFile.Info.Name)
		fmt.Printf("Size: %d bytes\n", torrentFile.Info.Length)
	}

	fmt.Printf("Pieces: %d (each %d bytes)\n", torrentFile.NumPieces(), torrentFile.Info.PieceLength)
	fmt.Printf("Info Hash: %x\n", torrentFile.InfoHash)

	fmt.Println("\nTorrent parsed successfully. We'll implement downloading in the next stage.")
}

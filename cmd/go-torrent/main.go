package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go-torrent <torrent-file>")
		os.Exit(1)
	}

	torrentFile := os.Args[1]
	fmt.Printf("Starting BitTorrent client for: %s\n", torrentFile)

	// TODO: We'll develop the actual functionality as we develop each component
}

package main

import (
	"fmt"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/piyushgupta53/go-torrent/internal/download"
	"github.com/piyushgupta53/go-torrent/internal/torrent"
	"github.com/piyushgupta53/go-torrent/internal/tracker"
)

const (
	clearLine = "\r\033[K"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go-torrent <torrent-file> [download-path]")
		os.Exit(1)
	}

	torrentPath := os.Args[1]

	// Determine download path
	downloadPath := "."
	if len(os.Args) >= 3 {
		downloadPath = os.Args[2]
	}

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

		// Display some file information (limit to 5 files to avoid cluttering the screen)
		var totalShown int
		var totalSize int64

		for i, file := range torrentFile.Info.Files {
			if i < 5 {
				fmt.Printf("  File %d: %s (%s)\n",
					i+1,
					filepath.Join(file.Path...),
					formatSize(file.Length))
				totalShown++
			}
			totalSize += file.Length
		}

		if totalShown < len(torrentFile.Info.Files) {
			remaining := len(torrentFile.Info.Files) - totalShown
			fmt.Printf("  ... and %d more files\n", remaining)
		}

		fmt.Printf("Total Size: %s\n", formatSize(totalSize))
	} else {
		fmt.Printf("Content: Single file (%s)\n", torrentFile.Info.Name)
		fmt.Printf("Size: %s\n", formatSize(torrentFile.Info.Length))
	}

	fmt.Printf("Pieces: %d (each %s)\n",
		torrentFile.NumPieces(),
		formatSize(torrentFile.Info.PieceLength))

	// Generate peer ID
	peerID, err := tracker.GeneratePeerID()
	if err != nil {
		fmt.Printf("Error generating peer ID: %v\n", err)
		os.Exit(1)
	}

	// Create download manager
	dm := download.NewDownloadManager(torrentFile, peerID, downloadPath, 50)

	// Handle Ctrl+C gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Printf("\nShutting down...\n")
		dm.Stop()
		os.Exit(0)
	}()

	// Set up callbacks
	completedPieces := make(map[int]bool)
	dm.OnPieceCompleted = func(index int) {
		completedPieces[index] = true
		fmt.Printf("%sPiece %d completed\n", clearLine, index)
	}

	dm.OnDownloadComplete = func() {
		fmt.Printf("\n%sDownload complete!\n", clearLine)
	}

	var lastSpeedDisplay float64
	var lastProgressDisplay float64
	var lastPeersDisplay int

	dm.OnStatsUpdated = func(stats download.Stats) {
		// Only update display if values change significantly
		speedKBps := float64(stats.DownloadSpeed) / 1024.0

		// Skip small changes to reduce flickering
		if stats.Progress == lastProgressDisplay &&
			math.Abs(speedKBps-lastSpeedDisplay) < 5.0 &&
			stats.ActivePeers == lastPeersDisplay {
			return
		}

		lastProgressDisplay = stats.Progress
		lastSpeedDisplay = speedKBps
		lastPeersDisplay = stats.ActivePeers

		// Format speed
		var speedStr string
		if speedKBps < 1024 {
			speedStr = fmt.Sprintf("%.1f KB/s", speedKBps)
		} else {
			speedStr = fmt.Sprintf("%.2f MB/s", speedKBps/1024.0)
		}

		// Format ETA
		var etaStr string
		if stats.DownloadSpeed > 0 {
			if stats.TimeRemaining > time.Hour*24 {
				days := int(stats.TimeRemaining.Hours()) / 24
				hours := int(stats.TimeRemaining.Hours()) % 24
				etaStr = fmt.Sprintf("%dd %dh", days, hours)
			} else if stats.TimeRemaining > time.Hour {
				etaStr = fmt.Sprintf("%dh %dm",
					int(stats.TimeRemaining.Hours()),
					int(stats.TimeRemaining.Minutes())%60)
			} else {
				etaStr = fmt.Sprintf("%dm %ds",
					int(stats.TimeRemaining.Minutes()),
					int(stats.TimeRemaining.Seconds())%60)
			}
		} else {
			etaStr = "calculating..."
		}

		// Display progress bar
		width := 30
		completed := int(float64(width) * stats.Progress / 100.0)
		bar := strings.Repeat("█", completed) + strings.Repeat("░", width-completed)

		fmt.Printf("%s[%s] %.1f%% | %s | Peers: %d | ETA: %s",
			clearLine, bar, stats.Progress, speedStr, stats.ActivePeers, etaStr)
	}

	// Start download
	fmt.Printf("\nStarting download to %s...\n", downloadPath)
	if err := dm.Start(); err != nil {
		fmt.Printf("Failed to start download: %v\n", err)
		os.Exit(1)
	}

	// Wait forever (shutdown happens through signal handler)
	select {}
}

// formatSize formats a byte size into a human-readable format
func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
		TB = 1024 * GB
	)

	if bytes < KB {
		return fmt.Sprintf("%d bytes", bytes)
	} else if bytes < MB {
		return fmt.Sprintf("%.1f KB", float64(bytes)/KB)
	} else if bytes < GB {
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	} else if bytes < TB {
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	} else {
		return fmt.Sprintf("%.2f TB", float64(bytes)/TB)
	}
}

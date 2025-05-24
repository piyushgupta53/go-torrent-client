package download

import (
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/piyushgupta53/go-torrent/internal/peer"
	"github.com/piyushgupta53/go-torrent/internal/torrent"
)

// PieceManager handles the downloading of pieces
type PieceManager struct {
	Torrent    *torrent.TorrentFile
	Pieces     []*Piece
	Downloaded map[int]bool
	InProgress map[int]bool
	Missing    map[int]bool
	Completed  int
	mu         sync.RWMutex
}

// NewPieceManager creates a new piece manager
func NewPieceManager(torrentFile *torrent.TorrentFile) *PieceManager {
	// Create all pieces
	pieces := make([]*Piece, torrentFile.NumPieces())
	for i := 0; i < torrentFile.NumPieces(); i++ {
		pieceSize := torrentFile.PieceSize(i)
		pieces[i] = NewPiece(i, torrentFile.PiecesHash[i], int(pieceSize))
	}

	// Initialize maps
	missing := make(map[int]bool)
	for i := 0; i < torrentFile.NumPieces(); i++ {
		missing[i] = true
	}

	return &PieceManager{
		Torrent:    torrentFile,
		Pieces:     pieces,
		Downloaded: make(map[int]bool),
		InProgress: make(map[int]bool),
		Missing:    missing,
		Completed:  0,
	}
}

// PieceCount returns the total number of pieces
func (pm *PieceManager) PieceCount() int {
	return len(pm.Pieces)
}

// DownloadedCount returns the number of downloaded pieces
func (pm *PieceManager) DownloadedCount() int {
	pm.mu.RLock()
	defer pm.mu.Unlock()

	return pm.Completed
}

// PickPiece selects a piece to download using the given strategy
func (pm *PieceManager) PickPiece(peersBitfield []peer.Bitfield, strategy string) *Piece {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Get pieces the peers have
	available := make(map[int]int) // piece index -> count of peers who have it
	for _, bitfield := range peersBitfield {
		for i := 0; i < len(pm.Pieces); i++ {

			if bitfield.HasPiece(i) && (pm.Missing[i] || pm.InProgress[i]) {
				available[i]++
			}
		}
	}

	// Filter out pieces that are already downloaded
	var candidates []int
	for pieceIndex := range available {
		if !pm.Downloaded[pieceIndex] {
			candidates = append(candidates, pieceIndex)
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	// Apply the selected strategy
	switch strategy {
	case "rarest_first":
		// Sort by rarity (ascending)
		sort.Slice(candidates, func(i, j int) bool {
			return available[candidates[i]] < available[candidates[j]]
		})
	case "random":
		// Shuffle the candidates
		rand.Seed(time.Now().UnixNano())
		rand.Shuffle(len(candidates), func(i, j int) {
			candidates[i], candidates[j] = candidates[j], candidates[i]
		})
	default:
		// Default to sequential
		sort.Ints(candidates)
	}

	// Pick the candidate that isn't already in progress
	for _, pieceIndex := range candidates {
		if !pm.InProgress[pieceIndex] {
			pm.InProgress[pieceIndex] = true
			delete(pm.Missing, pieceIndex)
			return pm.Pieces[pieceIndex]
		}
	}

	// If all candidates are in progress, pick the first candidate anyway
	if len(candidates) > 0 {
		pieceIndex := candidates[0]
		return pm.Pieces[pieceIndex]
	}

	return nil
}

// MarkPieceCompleted marks a piece as successfully downloaded and verified
func (pm *PieceManager) MarkPieceCompleted(pieceIndex int) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pieceIndex < 0 || pieceIndex >= len(pm.Pieces) {
		return fmt.Errorf("invalid piece index: %d", pieceIndex)
	}

	if pm.Downloaded[pieceIndex] {
		return nil // Already marked as downloaded
	}

	piece := pm.Pieces[pieceIndex]

	if !piece.Verify() {
		// Reset the piece
		piece.ResetRequests()
		delete(pm.InProgress, pieceIndex)
		pm.Missing[pieceIndex] = true
		return fmt.Errorf("piece %d verification failed", pieceIndex)
	}

	// Mark as download
	pm.Downloaded[pieceIndex] = true
	delete(pm.InProgress, pieceIndex)
	pm.Completed++

	// Update the piece state
	piece.State = PieceStateComplete

	return nil
}

// AddBlock adds a downloaded block to its corresponding piece
func (pm *PieceManager) AddBlock(pieceIndex, begin int, data []byte) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pieceIndex < 0 || pieceIndex >= len(pm.Pieces) {
		return fmt.Errorf("invalid piece index: %d", pieceIndex)
	}

	piece := pm.Pieces[pieceIndex]
	return piece.AddBlock(begin, data)
}

// IsComplete returns true if all pieces have been downloaded
func (pm *PieceManager) IsComplete() bool {
	pm.mu.RLock()
	defer pm.mu.Unlock()

	return len(pm.Pieces) == pm.Completed
}

// Progress returns the download progress as a percentage (0.0 to 1.0)
func (pm *PieceManager) Progress() float64 {
	pm.mu.RLock()
	defer pm.mu.Unlock()

	if len(pm.Pieces) == 0 {
		return 0.0
	}

	return float64(pm.Completed) / float64(len(pm.Pieces))
}

// ResetPiece resets a piece to the "not downloaded" state
func (pm *PieceManager) ResetPiece(pieceIndex int) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pieceIndex < 0 || pieceIndex >= len(pm.Pieces) {
		return nil
	}

	piece := pm.Pieces[pieceIndex]
	piece.ResetRequests()

	delete(pm.InProgress, pieceIndex)
	delete(pm.InProgress, pieceIndex)

	pm.Missing[pieceIndex] = true

	if piece.GetState() == PieceStateComplete {
		pm.Completed--
	}

	piece.State = PieceStateNone

	return nil

}

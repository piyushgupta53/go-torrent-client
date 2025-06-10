package download

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/piyushgupta53/go-torrent/internal/peer"
	"github.com/piyushgupta53/go-torrent/internal/torrent"
	"github.com/piyushgupta53/go-torrent/internal/tracker"
)

var (
	ErrDownloadCancelled = errors.New("download cancelled")
)

// Stats contains download statistics
type Stats struct {
	Downloaded      int64         // Bytes downloaded
	Uploaded        int64         // Bytes uploaded
	DownloadSpeed   int64         // Bytes per second
	UploadSpeed     int64         // Bytes per second
	PiecesCompleted int           // Number of completed pieces
	PiecesTotal     int           // Total number of pieces
	Progress        float64       // Download progress percentage
	ActivePeers     int           // Number of connected peers
	State           string        // Current state
	TimeRemaining   time.Duration // Estimated time remaining
}

// DownloadManager coordinates the entire download process
type DownloadManager struct {
	Torrent      *torrent.TorrentFile
	PeerID       [20]byte
	PeerPool     *peer.Pool
	PieceManager *PieceManager
	Storage      *FileStorage
	Stats        Stats

	maxPeers     int
	pieceTimeout time.Duration
	downloadPath string

	activePieces  map[int]string    // pieceIndex -> peerAddr
	pieceTimeouts map[int]time.Time // pieceIndex -> timeout

	cancel context.CancelFunc
	ctx    context.Context
	mu     sync.Mutex

	// Callbacks
	OnPieceCompleted   func(index int)
	OnPeerConnected    func(addr string)
	OnPeerDisconnected func(addr string)
	OnDownloadComplete func()
	OnStatsUpdated     func(stats Stats)
}

// NewDownloadManager creates a new download manager
func NewDownloadManager(
	torrentFile *torrent.TorrentFile,
	peerID [20]byte,
	downloadPath string,
	maxPeers int,
) *DownloadManager {

	// Use reasonable defaults if not specified
	if maxPeers <= 0 {
		maxPeers = 30
	}

	return &DownloadManager{
		Torrent:       torrentFile,
		PeerID:        peerID,
		PeerPool:      peer.NewPool(torrentFile.InfoHash, peerID),
		PieceManager:  NewPieceManager(torrentFile),
		downloadPath:  downloadPath,
		maxPeers:      maxPeers,
		pieceTimeout:  5 * time.Minute,
		activePieces:  make(map[int]string),
		pieceTimeouts: make(map[int]time.Time),
		Stats: Stats{
			PiecesTotal: torrentFile.NumPieces(),
			State:       "Initializing",
		},
	}
}

// Start begins the download process
func (dm *DownloadManager) Start() error {
	// Create storage
	var err error
	dm.Storage, err = NewFileStorage(dm.Torrent, dm.downloadPath)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Create context with cancellation
	dm.ctx, dm.cancel = context.WithCancel(context.Background())

	// Start background workers
	go dm.peerManagerWorker()
	go dm.pieceManagerWorker()
	go dm.statsWorker()

	dm.updateState("Started")

	return nil
}

// Stop stops the download process
func (dm *DownloadManager) Stop() {
	if dm.cancel != nil {
		dm.cancel()
	}

	if dm.Storage != nil {
		dm.Storage.Close()
	}

	dm.updateState("Stopped")
}

// peerManagerWorker manages peer connections
func (dm *DownloadManager) peerManagerWorker() {
	trackerInterval := 30 * time.Second
	trackerTicker := time.NewTicker(trackerInterval)
	defer trackerTicker.Stop()

	// Initial peer discovery
	dm.discoverPeers()

	for {
		select {
		case <-dm.ctx.Done():
			return
		case <-trackerTicker.C:
			dm.discoverPeers()
		}
	}
}

// discoverPeers discovers new peers from the tracker
func (dm *DownloadManager) discoverPeers() {
	dm.updateState("Discovering peers")

	// Create tracker client
	trackerClient := tracker.NewClient(dm.PeerID, 6881)

	// Prepare announce request
	req := &tracker.AnnounceRequest{
		InfoHash:   dm.Torrent.InfoHash,
		PeerID:     dm.PeerID,
		Port:       6881,
		Uploaded:   dm.Stats.Uploaded,
		Downloaded: dm.Stats.Downloaded,
		Left:       dm.Torrent.TotalLength() - dm.Stats.Downloaded,
		Compact:    true,
		Event:      "",
	}

	// Contact tracker
	resp, err := trackerClient.Announce(dm.Torrent.Announce, req)
	if err != nil {
		fmt.Printf("Tracker error: %v\n", err)
		return
	}

	// Connect to new peers
	currentPeers := dm.PeerPool.GetConnectedPeers()
	neededPeers := dm.maxPeers - currentPeers

	if neededPeers > 0 {
		// Try to connect to peers
		connected := dm.PeerPool.Connect(resp.Peers, neededPeers)
		if connected > 0 {
			fmt.Printf("Connected to %d new peers\n", connected)
		}
	}

	dm.updateState("Downloading")
}

// pieceManagerWorker manages piece downloads
func (dm *DownloadManager) pieceManagerWorker() {
	pieceTicker := time.NewTicker(1 * time.Second)
	defer pieceTicker.Stop()

	for {
		select {
		case <-dm.ctx.Done():
			return
		case <-pieceTicker.C:
			dm.managePieceDownloads()
		}
	}
}

// managePieceDownloads coordinates piece downloads
func (dm *DownloadManager) managePieceDownloads() {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	// Check for completed or timed out pieces
	now := time.Now()
	for pieceIndex, timeout := range dm.pieceTimeouts {
		if now.After(timeout) {
			// Piece timed out
			fmt.Printf("Piece %d timed out\n", pieceIndex)

			// Reset the piece
			dm.PieceManager.ResetPiece(pieceIndex)
			delete(dm.activePieces, pieceIndex)
			delete(dm.pieceTimeouts, pieceIndex)
		}
	}

	// Get all unchoked peer sessions
	unchokedSessions := dm.PeerPool.GetUnchokedSessions()
	if len(unchokedSessions) == 0 {
		return
	}

	// Get bitfields from all peers
	var bitfields []peer.Bitfield
	for _, session := range unchokedSessions {
		// Create a bitfield based on what pieces the peer has
		bf := make(peer.Bitfield, (dm.Torrent.NumPieces()+7)/8)
		for i := 0; i < dm.Torrent.NumPieces(); i++ {
			if session.HasPiece(i) {
				bf.SetPiece(i)
			}
		}
		bitfields = append(bitfields, bf)
	}

	// Limit concurrent downloads
	maxConcurrent := 5
	if len(dm.activePieces) >= maxConcurrent {
		return
	}

	// Try to download pieces
	for _, session := range unchokedSessions {
		if len(dm.activePieces) >= maxConcurrent {
			break
		}

		// Skip if this peer already has an active download
		peerHasActive := false
		for _, peerAddr := range dm.activePieces {
			if peerAddr == session.GetAddr() {
				peerHasActive = true
				break
			}
		}

		if peerHasActive {
			continue
		}

		// Pick a piece to download
		pieceToDownload := dm.PieceManager.PickPiece(bitfields, "rarest_first")
		if pieceToDownload == nil {
			continue
		}

		// Start downloading the piece
		dm.downloadPieceFromPeer(pieceToDownload, session)
	}
}

// downloadPieceFromPeer initiates a piece download from a specific peer
func (dm *DownloadManager) downloadPieceFromPeer(piece *Piece, session *peer.Session) {
	// Register piece as active
	dm.activePieces[piece.Index] = session.GetAddr()
	dm.pieceTimeouts[piece.Index] = time.Now().Add(dm.pieceTimeout)

	// Set callback for when we receive a piece
	session.SetOnPiece(func(receivedPiece *peer.Piece) {
		// Process the received block
		dm.processReceivedBlock(receivedPiece, piece, session)
	})

	// Request the first block
	dm.requestNextBlock(piece, session)
}

// processReceivedBlock handles a received block from a peer
func (dm *DownloadManager) processReceivedBlock(
	receivedPiece *peer.Piece,
	piece *Piece,
	session *peer.Session,
) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	// Make sure this is a block we're expecting
	if receivedPiece.Index != piece.Index {
		return
	}

	// Add the block to the piece
	err := dm.PieceManager.AddBlock(receivedPiece.Index, receivedPiece.Begin, receivedPiece.Block)
	if err != nil {
		fmt.Printf("Error adding block: %v\n", err)
		return
	}

	// Update stats
	dm.Stats.Downloaded += int64(len(receivedPiece.Block))

	// Check if the piece is complete
	// Continue from internal/download/downloader.go
	// processReceivedBlock continued...

	// Check if the piece is complete
	if piece.IsComplete() {
		// Verify the piece
		if piece.Verify() {
			fmt.Printf("Piece %d completed and verified\n", piece.Index)

			// Mark the piece as completed
			err := dm.PieceManager.MarkPieceCompleted(piece.Index)
			if err != nil {
				fmt.Printf("Error marking piece as completed: %v\n", err)
				return
			}

			// Write the piece to disk
			pieceData := piece.AssembleData()
			err = dm.Storage.WritePiece(piece.Index, pieceData)
			if err != nil {
				fmt.Printf("Error writing piece to disk: %v\n", err)
				return
			}

			// Update stats
			dm.Stats.PiecesCompleted++
			dm.Stats.Progress = float64(dm.Stats.PiecesCompleted) / float64(dm.Stats.PiecesTotal) * 100

			// Cleanup
			delete(dm.activePieces, piece.Index)
			delete(dm.pieceTimeouts, piece.Index)

			// Notify completion
			if dm.OnPieceCompleted != nil {
				dm.OnPieceCompleted(piece.Index)
			}

			// Check if entire download is complete
			if dm.PieceManager.IsComplete() {
				dm.updateState("Complete")
				if dm.OnDownloadComplete != nil {
					dm.OnDownloadComplete()
				}
			}

			// Send have message to all peers
			dm.PeerPool.BroadcastHave(piece.Index)
		} else {
			fmt.Printf("Piece %d failed verification\n", piece.Index)

			// Reset the piece
			dm.PieceManager.ResetPiece(piece.Index)
			delete(dm.activePieces, piece.Index)
			delete(dm.pieceTimeouts, piece.Index)
		}
	} else {
		// Request next block
		dm.requestNextBlock(piece, session)
	}
}

// requestNextBlock requests the next block from a peer
func (dm *DownloadManager) requestNextBlock(piece *Piece, session *peer.Session) {
	// Get next block to request
	block := piece.NextRequest()
	if block == nil {
		return
	}

	// Request the block
	err := session.RequestBlock(piece.Index, block.Begin, block.Length)
	if err != nil {
		fmt.Printf("Error requesting block: %v\n", err)
		return
	}
}

// statsWorker periodically updates download statistics
func (dm *DownloadManager) statsWorker() {
	statsTicker := time.NewTicker(1 * time.Second)
	defer statsTicker.Stop()

	var lastDownloaded int64
	var lastTime time.Time = time.Now()

	for {
		select {
		case <-dm.ctx.Done():
			return
		case <-statsTicker.C:
			dm.updateStats(lastDownloaded, lastTime)
			lastDownloaded = dm.Stats.Downloaded
			lastTime = time.Now()
		}
	}
}

// updateStats updates download statistics
func (dm *DownloadManager) updateStats(lastDownloaded int64, lastTime time.Time) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	currentTime := time.Now()
	timeDiff := currentTime.Sub(lastTime).Seconds()

	if timeDiff > 0 {
		byteDiff := dm.Stats.Downloaded - lastDownloaded
		dm.Stats.DownloadSpeed = int64(float64(byteDiff) / timeDiff)
	}

	dm.Stats.ActivePeers = dm.PeerPool.GetConnectedPeers()
	dm.Stats.PiecesCompleted = dm.PieceManager.DownloadedCount()
	dm.Stats.Progress = dm.PieceManager.Progress()

	// Calculate time remaining
	if dm.Stats.DownloadSpeed > 0 {
		bytesLeft := dm.Torrent.TotalLength() - dm.Stats.Downloaded
		secondsLeft := float64(bytesLeft) / float64(dm.Stats.DownloadSpeed)
		dm.Stats.TimeRemaining = time.Duration(secondsLeft) * time.Second
	}

	// Notify stats update
	if dm.OnStatsUpdated != nil {
		dm.OnStatsUpdated(dm.Stats)
	}
}

// updateState updates the current state
func (dm *DownloadManager) updateState(state string) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	dm.Stats.State = state

	// Notify stats update
	if dm.OnStatsUpdated != nil {
		dm.OnStatsUpdated(dm.Stats)
	}
}

// GetStats returns the current download statistics
func (dm *DownloadManager) GetStats() Stats {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	return dm.Stats
}

// IsComplete returns true if the download is complete
func (dm *DownloadManager) IsComplete() bool {
	return dm.PieceManager.IsComplete()
}

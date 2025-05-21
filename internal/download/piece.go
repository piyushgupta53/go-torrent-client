package download

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"fmt"
	"sync"
)

const (
	// BlockSize is the default size of a block (16KB)
	BlockSize = 16 * 1024
)

var (
	ErrInvalidPiece = errors.New("invalid piece")
)

// PieceState represents the state of a piece
type PieceState int

const (
	PieceStateNone PieceState = iota
	PieceStatePending
	PieceStateComplete
)

// Block represents a block within a piece
type Block struct {
	Index  int    // Block index within the piece
	Begin  int    // Offset within the piece
	Length int    // Length of the block
	Data   []byte // Block data (nil if not downloaded)
}

// Piece represents a piece of the torrent
type Piece struct {
	Index      int          // Piece index
	Hash       [20]byte     // Expected SHA-1 hash
	Length     int          // Piece length in bytes
	Blocks     []*Block     // Blocks within the piece
	State      PieceState   // Current state of the piece
	Downloaded int          // Number of bytes downloaded
	Requested  map[int]bool // Tracks which blocks have been requested
	mu         sync.RWMutex // Mutex for concurrent access
}

// NewPiece creates a new piece
func NewPiece(index int, hash [20]byte, length int) *Piece {
	// Calculate the number of blocks needed
	numBlocks := length / BlockSize
	if length%BlockSize != 0 {
		numBlocks++
	}

	// Create blocks
	blocks := make([]*Block, numBlocks)

	for i := 0; i < numBlocks; i++ {
		begin := i * BlockSize
		blockLen := BlockSize

		// Last block might be smaller
		if i == numBlocks-1 && length%BlockSize != 0 {
			blockLen = length % BlockSize
		}

		blocks[i] = &Block{
			Index:  i,
			Begin:  begin,
			Length: blockLen,
		}
	}

	return &Piece{
		Index:     index,
		Hash:      hash,
		Length:    length,
		Blocks:    blocks,
		State:     PieceStateNone,
		Requested: make(map[int]bool),
	}
}

// MarkRequested marks a block as requested
func (p *Piece) MarkRequested(blockIndex int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if blockIndex >= 0 && blockIndex < len(p.Blocks) {
		p.Requested[blockIndex] = true
		p.State = PieceStatePending
	}
}

// AddBlock adds a downloaded block to the piece
func (p *Piece) AddBlock(begin int, data []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Find the block
	for i, block := range p.Blocks {
		if begin == block.Begin {
			// Check length
			if len(data) != block.Length {
				return fmt.Errorf("block length mistmatch: got %d, expected: %d", len(data), block.Length)
			}

			// Add data
			p.Blocks[i].Data = data
			p.Downloaded += len(data)

			return nil
		}
	}

	return fmt.Errorf("no block found with begin offset %d", begin)
}

// IsComplete returns true if all blocks have been downloaded
func (p *Piece) IsComplete() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.Length == p.Downloaded
}

// AssembleData assembles all block data into a single byte slice
func (p *Piece) AssembleData() []byte {
	p.mu.RLock()
	defer p.mu.Unlock()

	if !p.IsComplete() {
		return nil
	}

	data := make([]byte, p.Length)

	for _, block := range p.Blocks {
		if block.Data != nil {
			copy(data[block.Begin:], block.Data)
		}
	}

	return data
}

// Verify checks if the piece data matches the expected hash
func (p *Piece) Verify() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.IsComplete() {
		return false
	}

	data := p.AssembleData()

	if data == nil {
		return false
	}

	hash := sha1.Sum(data)
	return bytes.Equal(p.Hash[:], hash[:])
}

// NextRequest returns the next block to request, or nil if all blocks are requested
func (p *Piece) NextRequest() *Block {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, block := range p.Blocks {
		if block.Data != nil && !p.Requested[block.Index] {
			p.Requested[block.Index] = true
			return block
		}
	}

	return nil
}

// GetState returns the current state of the piece
func (p *Piece) GetState() PieceState {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.State
}

// ResetRequests marks all blocks as not requested
func (p *Piece) ResetRequests() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.Requested = make(map[int]bool)
	if p.State == PieceStatePending {
		p.State = PieceStateNone
	}
}

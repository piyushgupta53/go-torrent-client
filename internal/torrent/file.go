package torrent

import (
	"path/filepath"
	"time"
)

type TorrentFile struct {
	Announce     string     // URL of the tracker
	AnnouceList  [][]string // List of backup trackers
	CreationDate time.Time
	Comment      string
	CreatedBy    string
	Encoding     string
	Info         InfoDict
	InfoHash     [20]byte
	PiecesHash   [][20]byte
}

type InfoDict struct {
	PieceLength int64
	Pieces      string
	Private     bool
	Name        string
	Length      int64
	Files       []FileDict
	IsDirectory bool
}

type FileDict struct {
	Length int64
	Path   []string
}

// TotalLength returns the total length of all files in the torrent
func (t *TorrentFile) TotalLength() int64 {
	if !t.Info.IsDirectory {
		return t.Info.Length
	}

	var length int64
	for _, file := range t.Info.Files {
		length += file.Length
	}

	return length
}

// NumPieces returns the number of pieces in the torrent
func (t *TorrentFile) NumPieces() int {
	return len(t.PiecesHash)
}

// PieceSize returns the size of a specific piece
func (t *TorrentFile) PieceSize(index int) int64 {
	if index < 0 || index >= t.NumPieces() {
		return 0
	}

	// For all pieces except the last one, the size is the piece length
	if index < t.NumPieces()-1 {
		return t.Info.PieceLength
	}

	// For the last piece, the size might be less than the piece length
	totalLength := t.TotalLength()
	lastPieceSize := totalLength % t.Info.PieceLength
	if lastPieceSize == 0 {
		return t.Info.PieceLength
	}
	return lastPieceSize
}

// FilePathForPiece returns the file path(s) that contain the specified piece
func (t *TorrentFile) FilePathForPiece(index int) []string {
	if index < 0 || index >= t.NumPieces() {
		return nil
	}

	// For single file torrents, just return the file name
	if !t.Info.IsDirectory {
		return []string{t.Info.Name}
	}

	// For multi-file torrents, determine which files contain this piece
	pieceOffset := int64(index) * t.Info.PieceLength
	pieceEnd := pieceOffset + t.PieceSize(index)

	var currentOffset int64
	var result []string

	for _, file := range t.Info.Files {
		fileStart := currentOffset
		fileEnd := fileStart + file.Length

		// Check if this file overlaps with the piece
		if fileEnd > pieceOffset && fileStart < pieceEnd {
			// Construct the full path
			path := filepath.Join(append([]string{t.Info.Name}, file.Path...)...)
			result = append(result, path)
		}

		currentOffset = fileEnd
	}

	return result
}

package download

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/piyushgupta53/go-torrent/internal/torrent"
)

type FileStorage struct {
	Torrent  *torrent.TorrentFile
	BasePath string
	Files    []*os.File
	mu       sync.Mutex
}

// NewFileStorage creates a new file storage handler
func NewFileStorage(torrentFile *torrent.TorrentFile, basepath string) (*FileStorage, error) {
	if basepath == "" {
		basepath = "."
	}

	fs := &FileStorage{
		Torrent:  torrentFile,
		BasePath: basepath,
	}

	// Create the target directory structure
	if err := fs.createDirectories(); err != nil {
		return nil, err
	}

	return fs, nil
}

// createDirectories creates the necessary directory structure
func (fs *FileStorage) createDirectories() error {
	if fs.Torrent.Info.IsDirectory {
		// Create the base directory
		dirPath := filepath.Join(fs.BasePath, fs.Torrent.Info.Name)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory '%s': %w", dirPath, err)
		}

		// Create subdirectories for multi-file torrents
		for _, file := range fs.Torrent.Info.Files {
			if len(file.Path) <= 1 {
				// skip as it's a file in root folder
				continue
			}

			subPath := filepath.Join(append([]string{dirPath}, file.Path[:len(file.Path)-1]...)...)
			if err := os.MkdirAll(subPath, 0755); err != nil {
				return fmt.Errorf("failed to create directory '%s': %w", subPath, err)
			}
		}

	}

	return nil
}

// openFiles opens all files for writing
func (fs *FileStorage) openFiles() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if fs.Torrent.Info.IsDirectory {
		// Multi-file mode
		fs.Files = make([]*os.File, len(fs.Torrent.Info.Files))

		for i, fileInfo := range fs.Torrent.Info.Files {
			filePath := filepath.Join(append([]string{fs.BasePath, fs.Torrent.Info.Name}, fileInfo.Path...)...)

			// Create the file (truncate if exists)
			file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0644)
			if err != nil {
				fs.closeFiles()
				return fmt.Errorf("failed to open file '%s': %w", filePath, err)
			}

			// Set the file size
			if err := file.Truncate(fileInfo.Length); err != nil {
				file.Close()
				fs.closeFiles()
				return fmt.Errorf("failed to set file size for '%s': %w", filePath, err)
			}

			fs.Files[i] = file
		}
	} else {
		// Single-file mode
		fs.Files = make([]*os.File, 1)

		filePath := filepath.Join(fs.BasePath, fs.Torrent.Info.Name)

		// Open file
		file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			file.Close()
			return fmt.Errorf("failed to set file size for '%s': %w", filePath, err)
		}

		fs.Files[0] = file
	}

	return nil
}

func (fs *FileStorage) closeFiles() {
	for i, file := range fs.Files {
		if file != nil {
			file.Close()
			fs.Files[i] = nil
		}
	}
}

// WritePiece writes a piece to the appropriate files
func (fs *FileStorage) WritePiece(pieceIndex int, data []byte) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Calculate the piece offset in the overall torrent data
	pieceOffset := int64(pieceIndex) * fs.Torrent.Info.PieceLength

	// Handle the single file case
	if !fs.Torrent.Info.IsDirectory {
		_, err := fs.Files[0].WriteAt(data, pieceOffset)
		return err
	}

	// Handle the multi-file case
	var bytesWritten int
	var fileOffset int64

	for i, fileInfo := range fs.Torrent.Info.Files {
		// Check if this file contains part of the piece
		if pieceOffset >= fileOffset && pieceOffset < fileOffset+fileInfo.Length || fileOffset >= pieceOffset && fileOffset < pieceOffset+int64(len(data)) {

			// Calculate overlap between piece and file
			overlapStart := max(pieceOffset, fileOffset)
			overlapEnd := min(pieceOffset+int64(len(data)), fileOffset+fileInfo.Length)
			overlapSize := overlapEnd - overlapStart

			if overlapSize < 0 {
				continue
			}

			// Calculate offsets for writing
			fileWriteOffset := overlapStart - fileOffset
			pieceReadOffset := int(overlapStart - pieceOffset)

			// Write the data
			_, err := fs.Files[i].WriteAt(data[pieceReadOffset:pieceReadOffset+int(overlapSize)], fileWriteOffset)
			if err != nil {
				return fmt.Errorf("failed to write to file %d: %w", i, err)
			}

			bytesWritten += int(overlapSize)

			if bytesWritten >= len(data) {
				break
			}
		}

		fileOffset += fileInfo.Length
	}

	return nil
}

// Close closes all open files and cleans up resources
func (fs *FileStorage) Close() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.closeFiles()
	return nil
}

// Helper functions
func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

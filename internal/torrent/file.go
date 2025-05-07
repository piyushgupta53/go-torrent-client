package torrent

import "time"

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

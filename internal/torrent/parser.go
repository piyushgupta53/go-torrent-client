package torrent

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/piyushgupta53/go-torrent/internal/bencode"
)

// Parse errors
var (
	ErrInvalidTorrentFile = errors.New("invalid torrent file")
	ErrInvalidInfoDict    = errors.New("invalid info dictionary")
	ErrInvalidPieces      = errors.New("invalid pieces")
)

// ParseFromFile reads a .torrent file and returns a TorrentFile struct
func ParseFromFile(path string) (*TorrentFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	// Decode the bencode data
	data, err := bencode.Decode(file)
	if err != nil {
		return nil, err
	}

	// Convert the decoded data to a TorrentFile struct
	return Parse(data)
}

// Parse converts the decoded bencode data into a TorrentFile struct
func Parse(data any) (*TorrentFile, error) {
	dict, ok := data.(map[string]any)
	if !ok {
		return nil, ErrInvalidTorrentFile
	}

	// Create a new TorrentFile strcut
	t := &TorrentFile{}

	// Parse annouce URL
	annouceVal, ok := dict["annouce"]
	if !ok {
		return nil, fmt.Errorf("%w: missing annouce URL", ErrInvalidTorrentFile)
	}

	annouce, ok := annouceVal.(string)
	if !ok {
		return nil, fmt.Errorf("%w: annouce is not a string", ErrInvalidTorrentFile)
	}

	t.Announce = annouce

	// Parse annouce-list
	if announceListVal, ok := dict["annouce-list"]; ok {
		announceList, ok := announceListVal.([]any)
		if !ok {
			return nil, fmt.Errorf("%w: annouce-list is not a list", ErrInvalidTorrentFile)
		}

		t.AnnouceList = make([][]string, len(announceList))
		for i, tier := range announceList {
			tierList, ok := tier.([]any)
			if !ok {
				return nil, fmt.Errorf("%w: annouce-list tier is not a list", ErrInvalidInfoDict)
			}

			t.AnnouceList[i] = make([]string, len(tierList))
			for j, tracker := range tierList {
				trackerURL, ok := tracker.(string)
				if !ok {
					return nil, fmt.Errorf("%w: tracker URL is not a string", ErrInvalidTorrentFile)
				}
				t.AnnouceList[i][j] = trackerURL
			}
		}
	}

	// Parse creation date
	if creationDateVal, ok := dict["creation date"]; ok {
		creationDate, ok := creationDateVal.(int64)
		if !ok {
			return nil, fmt.Errorf("%w: creation date is not an integer", ErrInvalidTorrentFile)
		}

		t.CreationDate = time.Unix(creationDate, 0)
	}

	// Parse comment
	if commentVal, ok := dict["comment"]; ok {
		comment, ok := commentVal.(string)
		if !ok {
			return nil, fmt.Errorf("%w: comment is not a string", ErrInvalidTorrentFile)
		}

		t.Comment = comment
	}

	// Parse created by
	if createdByVal, ok := dict["created by"]; ok {
		createdBy, ok := createdByVal.(string)
		if !ok {
			return nil, fmt.Errorf("%w: created by is not a string", ErrInvalidTorrentFile)
		}

		t.CreatedBy = createdBy
	}

	// Parse encoding (optional)
	if encodingVal, ok := dict["encoding"]; ok {
		encoding, ok := encodingVal.(string)
		if !ok {
			return nil, fmt.Errorf("%w: encoding is not a string", ErrInvalidTorrentFile)
		}
		t.Encoding = encoding
	}

	infoVal, ok := dict["info"]
	if !ok {
		return nil, fmt.Errorf("%w: missing info dictionary", ErrInvalidTorrentFile)
	}

	infoDict, ok := infoVal.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%w: info is not a dictionary", ErrInvalidTorrentFile)
	}

	// Parse into fields
	if err := parseInfoDict(infoDict, &t.Info); err != nil {
		return nil, err
	}

	// Calculate the info hash
	infoHash, err := calculateHashInfo(infoDict)
	if err != nil {
		return nil, err
	}

	t.InfoHash = infoHash

	// Parse pieces hash
	piecesHash, err := parsePieces(t.Info.Pieces)
	if err != nil {
		return nil, err
	}

	t.PiecesHash = piecesHash
	return t, nil
}

// parseInfoDict parses the info dictionary
func parseInfoDict(info map[string]any, infoDict *InfoDict) error {
	// parse piece length
	pieceLengthVal, ok := info["piece length"]
	if !ok {
		return fmt.Errorf("%w: missing piece length", ErrInvalidInfoDict)
	}

	pieceLength, ok := pieceLengthVal.(int64)
	if !ok {
		return fmt.Errorf("%w: piece length is not an integer", ErrInvalidInfoDict)
	}

	infoDict.PieceLength = pieceLength

	// parse pieces hashes
	piecesVal, ok := info["pieces"]
	if !ok {
		return fmt.Errorf("%w: missing pieces", ErrInvalidInfoDict)
	}

	pieces, ok := piecesVal.(string)
	if !ok {
		return fmt.Errorf("%w: pieces is not a string", ErrInvalidInfoDict)
	}

	infoDict.Pieces = pieces

	// parse private flag
	if privateVal, ok := info["private"]; ok {
		private, ok := privateVal.(int64)
		if !ok {
			return fmt.Errorf("%w: private is not an integer", ErrInvalidInfoDict)
		}

		infoDict.Private = private == 1
	}

	// parse name
	nameVal, ok := info["name"]
	if !ok {
		return fmt.Errorf("%w: missing name", ErrInvalidInfoDict)
	}

	name, ok := nameVal.(string)
	if !ok {
		return fmt.Errorf("%w: name is not a string", ErrInvalidInfoDict)
	}
	infoDict.Name = name

	// check if single file or multi-file
	if lengthVal, ok := info["length"]; ok {
		// Single file mode
		length, ok := lengthVal.(int64)
		if !ok {
			return fmt.Errorf("%w: length is not an integer", ErrInvalidInfoDict)
		}

		infoDict.Length = length
		infoDict.IsDirectory = false
	} else if filesVal, ok := info["files"]; ok {
		// Multi-file mode
		files, ok := filesVal.([]any)
		if !ok {
			return fmt.Errorf("%w: files is not a list", ErrInvalidInfoDict)
		}

		infoDict.Files = make([]FileDict, len(files))

		for i, fileVal := range files {
			fileDict, ok := fileVal.(map[string]any)
			if !ok {
				return fmt.Errorf("%w: file is not a dictionary", ErrInvalidInfoDict)
			}

			// Parse file length
			fileLengthVal, ok := fileDict["length"]
			if !ok {
				return fmt.Errorf("%w: missing file length", ErrInvalidInfoDict)
			}

			fileLength, ok := fileLengthVal.(int64)
			if !ok {
				return fmt.Errorf("%w: file length is not an integer", ErrInvalidInfoDict)
			}

			infoDict.Files[i].Length = fileLength

			// parse file path
			pathVal, ok := fileDict["path"]
			if !ok {
				return fmt.Errorf("%w: path is missing", ErrInvalidInfoDict)
			}

			pathList, ok := pathVal.([]any)
			if !ok {
				return fmt.Errorf("%w: path is not a list", ErrInvalidInfoDict)
			}

			infoDict.Files[i].Path = make([]string, len(pathList))
			for j, pathElemVal := range pathList {
				pathElem, ok := pathElemVal.(string)
				if !ok {
					return fmt.Errorf("%w: path element is not a string", ErrInvalidInfoDict)
				}

				infoDict.Files[i].Path[j] = pathElem
			}
		}
		infoDict.IsDirectory = true
	} else {
		return fmt.Errorf("%w: neither length nor files found", ErrInvalidInfoDict)
	}

	return nil
}

// parsePieces extracts the SHA-1 hashes from the pieces string
func parsePieces(pieces string) ([][20]byte, error) {
	numPieces := len(pieces) / 20
	hashes := make([][20]byte, numPieces)

	for i := range numPieces {
		copy(hashes[i][:], pieces[i*20:(i+1)*20])
	}

	return hashes, nil
}

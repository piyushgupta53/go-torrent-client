# Go-Torrent

A BitTorrent client implementation in Go that supports downloading files using the BitTorrent protocol. This project implements the core functionality of a BitTorrent client, including torrent file parsing, peer discovery, and file downloading.

## Features

- **Bencode Encoding/Decoding**

  - Support for all Bencode types (strings, integers, lists, dictionaries)
  - Robust error handling and validation

- **Torrent File Processing**

  - Parse .torrent files (both single-file and multi-file torrents)
  - Info hash calculation
  - Piece hash extraction
  - Support for all standard torrent file fields

- **Peer Discovery and Communication**

  - HTTP tracker support
  - Peer handshake protocol
  - Complete BitTorrent message protocol
  - Peer connection management

- **Download Functionality**
  - Piece and block management
  - Concurrent downloads
  - Progress tracking
  - File assembly for both single and multi-file torrents
  - Storage management with block-level granularity

## Project Structure

```
.
├── cmd/          # Command-line interface and main application
├── internal/     # Internal packages
│   ├── bencode/  # Bencode encoding/decoding
│   ├── torrent/  # Torrent file processing
│   ├── tracker/  # Tracker protocol implementation
│   ├── peer/     # Peer communication
│   └── download/ # Download management
└── pkg/          # Public packages
```

## Requirements

- Go 1.21 or later

## Installation

```bash
git clone https://github.com/yourusername/go-torrent.git
cd go-torrent
go mod download
```

## Usage

[Usage instructions will be added as the project matures]

## Development Status

This project is actively under development. Current implementation status can be found in [checkpoint.md](checkpoint.md).

### Planned Features

- Magnet link support
- Metadata exchange protocol
- DHT (Distributed Hash Table) support

## Acknowledgments

- [BitTorrent Protocol Specification](https://wiki.theory.org/BitTorrentSpecification)
- [Bencode Specification](https://wiki.theory.org/BitTorrentSpecification#Bencoding)

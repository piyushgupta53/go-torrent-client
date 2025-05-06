package bencode

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Common errors
var (
	ErrInvalidBencode = errors.New("invalid bencode format")
	ErrIntegerFormat  = errors.New("invalid integer format")
	ErrStringLength   = errors.New("invalid string length")
)

func Decode(r io.Reader) (interface{}, error) {
	br := bufio.NewReader(r)

	return decodeNext(br)
}

func decodeNext(r *bufio.Reader) (interface{}, error) {
	// peek the first byte to determine the type
	b, err := r.Peek(1)

	if err != nil {
		return nil, err
	}

	switch {
	case b[0] >= '0' && b[0] <= 9:
		return decodeString(r)
	case b[0] == 'i':
		return decodeInteger(r)
	case b[0] == 'l':
		return decodeList(r)
	case b[0] == 'd':
		return decodeDict(r)
	default:
		return nil, ErrInvalidBencode
	}
}

func decodeString(r *bufio.Reader) (string, error) {
	// Read digits until we hit a colon
	lengthStr, err := readUntil(r, ':')

	if err != nil {
		return "", err
	}

	// convert length string into an integer
	length, err := strconv.Atoi(lengthStr)
	if err != nil {
		return "", err
	}

	// Read exactly length bytes
	stringBytes := make([]byte, length)
	_, err = io.ReadFull(r, stringBytes)

	if err != nil {
		return "", err
	}

	return string(stringBytes), nil
}

// e.g. 4:spam
func readUntil(r *bufio.Reader, delimiter byte) (string, error) {
	var result []byte

	for {
		b, err := r.ReadByte()

		if err != nil {
			return "", err
		}

		if b == delimiter {
			break
		}

		result = append(result, b)
	}

	return string(result), nil
}

// e.g. i42e
func decodeInteger(r *bufio.Reader) (int64, error) {
	// Skip the leading 'i'
	_, err := r.ReadByte()
	if err != nil {
		return 0, err
	}

	// Read digits until we hit 'e'
	numStr, err := readUntil(r, 'e')
	if err != nil {
		return 0, err
	}

	// Validate the integer format
	if len(numStr) > 1 && numStr[0] == '0' {
		return 0, ErrIntegerFormat
	}

	if len(numStr) > 1 && strings.HasPrefix(numStr, "-0") {
		return 0, ErrIntegerFormat
	}

	// Convert string int to integer
	num, err := strconv.ParseInt(numStr, 10, 64)

	if err != nil {
		return 0, fmt.Errorf("invalid interger: %w", err)
	}

	return num, nil
}

// Example: l4:spam4:eggse represents the list ["spam", "eggs"]
func decodeList(r *bufio.Reader) ([]interface{}, error) {
	// Skip the leading 'l'
	_, err := r.ReadByte()

	if err != nil {
		return nil, err
	}

	var list []interface{}

	// Keep decoding until we hit 'e'
	for {
		// Peek to see if we've reached the end of the list
		b, err := r.Peek(1)
		if err != nil {
			return nil, err
		}

		if b[0] == 'e' {
			// Skip the trailing 'e'
			_, err = r.ReadByte()
			return list, err
		}

		// Decode the next item
		item, err := decodeNext(r)

		if err != nil {
			return nil, err
		}

		list = append(list, item)
	}
}

// Example: d3:cow3:moo4:spam4:eggse represents the map {"cow": "moo", "spam": "eggs"}
func decodeDict(r *bufio.Reader) (map[string]interface{}, error) {
	// Skip the leading 'd'
	_, err := r.ReadByte()
	if err != nil {
		return nil, err
	}

	dict := make(map[string]interface{})

	for {
		// Peek to see if we've reached the end 'e'
		b, err := r.Peek(1)

		if err != nil {
			return nil, err
		}

		if b[0] == 'e' {
			// Skip the trailing byte 'e'
			_, err = r.ReadByte()
			return dict, err
		}

		key, err := decodeNext(r)
		if err != nil {
			return nil, err
		}

		keyStr, ok := key.(string)
		if !ok {
			return nil, ErrInvalidBencode
		}

		value, err := decodeNext(r)
		if err != nil {
			return nil, err
		}

		dict[keyStr] = value
	}
}

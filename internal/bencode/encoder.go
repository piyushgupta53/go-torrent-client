package bencode

import (
	"fmt"
	"io"
	"sort"
)

// Encode writes a bencode representation of v to the provided writer
func Encode(w io.Writer, v any) error {
	return encodeValue(w, v)
}

// encodeValue writes the bencode representation of a value
func encodeValue(w io.Writer, v any) error {
	switch val := v.(type) {
	case string:
		return encodeString(w, val)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return encodeInteger(w, val)
	case []any:
		return encodeList(w, val)
	case map[string]any:
		return encodeDict(w, val)
	default:
		return fmt.Errorf("cannont encode type %T", v)
	}
}

// encodeString writes a bencoded string
func encodeString(w io.Writer, s string) error {
	_, err := fmt.Fprintf(w, "%d:%s", len(s), s)
	return err
}

func encodeInteger(w io.Writer, v any) error {
	_, err := fmt.Fprintf(w, "i%de", v)
	return err
}

func encodeList(w io.Writer, list []any) error {
	if _, err := w.Write([]byte("l")); err != nil {
		return err
	}

	for _, item := range list {
		if err := encodeValue(w, item); err != nil {
			return err
		}
	}

	_, err := w.Write([]byte("e"))
	return err
}

// encodeDict writes a bencoded dictionary
func encodeDict(w io.Writer, dict map[string]any) error {
	if _, err := w.Write([]byte("d")); err != nil {
		return err
	}

	// Sort keys according to bencode spec
	keys := make([]string, 0, len(dict))
	for key := range dict {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Write each key-value pair in sorted order
	for _, key := range keys {
		if err := encodeString(w, key); err != nil {
			return err
		}

		if err := encodeValue(w, dict[key]); err != nil {
			return err
		}
	}

	_, err := w.Write([]byte("e"))
	return err
}

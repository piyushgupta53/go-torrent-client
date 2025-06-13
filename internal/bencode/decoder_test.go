package bencode

import (
	"bytes"
	"reflect"
	"testing"
)

func TestDecodeString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		wantErr  bool
	}{
		{"4:spam", "spam", false},
		{"0:", "", false},
		{"5:hello", "hello", false},
		{"10:1234567890", "1234567890", false},
		// Error cases
		{"4spam", "", true},   // No colon
		{"-1:spam", "", true}, // Negative length
		{"4:spa", "", true},   // Incomplete string
	}

	for _, tt := range tests {
		r := bytes.NewBufferString(tt.input)
		got, err := Decode(r)

		if (err != nil) != tt.wantErr {
			t.Errorf("Decode(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}

		if !tt.wantErr && got != tt.expected {
			t.Errorf("Decode(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestDecodeInteger(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		wantErr  bool
	}{
		{"i3e", 3, false},
		{"i0e", 0, false},
		{"i-3e", -3, false},
		{"i123456789e", 123456789, false},
		// Error cases
		{"i03e", 0, true},     // Leading zero
		{"i-0e", 0, true},     // Negative zero
		{"ie", 0, true},       // Empty integer
		{"i123", 0, true},     // No end marker
		{"iabc123e", 0, true}, // Invalid characters
	}

	for _, tt := range tests {
		r := bytes.NewBufferString(tt.input)
		got, err := Decode(r)

		if (err != nil) != tt.wantErr {
			t.Errorf("Decode(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}

		if !tt.wantErr && got != tt.expected {
			t.Errorf("Decode(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestDecodeList(t *testing.T) {
	tests := []struct {
		input    string
		expected []any
		wantErr  bool
	}{
		{"le", []any{}, false},
		{"l4:spame", []any{"spam"}, false},
		{"l4:spam4:eggse", []any{"spam", "eggs"}, false},
		{"li3ei4ee", []any{int64(3), int64(4)}, false},
		{"l4:spami3ee", []any{"spam", int64(3)}, false},
		// Nested list
		{"ll4:spamee", []any{[]any{"spam"}}, false},
		// Error cases
		{"l4:spam", nil, true}, // No end marker
	}

	for _, tt := range tests {
		r := bytes.NewBufferString(tt.input)
		got, err := Decode(r)

		if (err != nil) != tt.wantErr {
			t.Errorf("Decode(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}

		if !tt.wantErr && !reflect.DeepEqual(got, tt.expected) {
			t.Errorf("Decode(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestDecodeDict(t *testing.T) {
	tests := []struct {
		input    string
		expected map[string]any
		wantErr  bool
	}{
		{"de", map[string]any{}, false},
		{"d3:cow3:mooe", map[string]any{"cow": "moo"}, false},
		{"d4:spam4:eggs3:cowi3ee", map[string]any{"spam": "eggs", "cow": int64(3)}, false},
		// Nested dict
		{"d4:dictd3:cow3:mooee", map[string]any{"dict": map[string]any{"cow": "moo"}}, false},
		// Dict with list
		{"d4:listl4:spam4:eggsee", map[string]any{"list": []any{"spam", "eggs"}}, false},
		// Error cases
		{"d4:spam", nil, true}, // No end marker
	}

	for _, tt := range tests {
		r := bytes.NewBufferString(tt.input)
		got, err := Decode(r)

		if (err != nil) != tt.wantErr {
			t.Errorf("Decode(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}

		if !tt.wantErr && !reflect.DeepEqual(got, tt.expected) {
			t.Errorf("Decode(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestComplex(t *testing.T) {
	// Test a typical torrent file structure
	input := "d8:announce35:http://tracker.example.com/announce4:infod6:lengthi12345e4:name8:test.txt12:piece lengthi16384e6:pieces20:abcdefghijklmnopqrstee"

	expected := map[string]any{
		"announce": "http://tracker.example.com/announce",
		"info": map[string]any{
			"length":       int64(12345),
			"name":         "test.txt",
			"piece length": int64(16384),
			"pieces":       "abcdefghijklmnopqrst",
		},
	}

	r := bytes.NewBufferString(input)
	got, err := Decode(r)

	if err != nil {
		t.Errorf("Decode() error = %v", err)
		return
	}

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Decode() = %v, want %v", got, expected)
	}
}

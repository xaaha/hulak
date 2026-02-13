package utils

import (
	"bytes"
	"strings"
	"testing"
)

type noOpColorProvider struct{}

func (n noOpColorProvider) ColorString(s string) string { return s }
func (n noOpColorProvider) ColorNumber(s string) string { return s }
func (n noOpColorProvider) ColorBool(s string) string   { return s }
func (n noOpColorProvider) ColorNull(s string) string   { return s }
func (n noOpColorProvider) ColorKey(s string) string    { return s }

func TestFormatJSONColored(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
		wantErr  bool
	}{
		{
			name:     "simple object",
			input:    `{"name":"Alice","age":30}`,
			contains: []string{`"name"`, `"Alice"`, "30"},
			wantErr:  false,
		},
		{
			name:     "nested object",
			input:    `{"user":{"id":1,"active":true}}`,
			contains: []string{`"user"`, `"id"`, "1", "true"},
			wantErr:  false,
		},
		{
			name:     "array",
			input:    `[1,2,3]`,
			contains: []string{"1", "2", "3"},
			wantErr:  false,
		},
		{
			name:     "null value",
			input:    `{"key":null}`,
			contains: []string{"null"},
			wantErr:  false,
		},
		{
			name:     "empty object",
			input:    `{}`,
			contains: []string{"{}"},
			wantErr:  false,
		},
		{
			name:     "empty array",
			input:    `[]`,
			contains: []string{"[]"},
			wantErr:  false,
		},
		{
			name:    "invalid JSON",
			input:   `{not valid}`,
			wantErr: true,
		},
	}

	provider := noOpColorProvider{}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := FormatJSONColored([]byte(tc.input), provider)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			for _, s := range tc.contains {
				if !strings.Contains(result, s) {
					t.Errorf("expected output to contain %q, got:\n%s", s, result)
				}
			}
		})
	}
}

func benchmarkMarshall(b *testing.B) {
	var buf bytes.Buffer
	provider := fatihColorProvider{}
	simpleMap := make(map[string]any)
	simpleMap["a"] = 1
	simpleMap["b"] = "bee"
	simpleMap["c"] = [3]float64{1, 2, 3}
	simpleMap["d"] = [3]string{"one", "two", "three"}

	for b.Loop() {
		buf.Reset()
		marshalValue(simpleMap, &buf, 0, provider)
	}
}

func BenchmarkMarshall(b *testing.B) { benchmarkMarshall(b) }

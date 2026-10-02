package utils

import (
	"strings"
	"testing"
)

func TestD22OnlyNonEmptyDocumentsCount(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantLine string
	}{
		{"plain mapping", "method: POST\nurl: https://e.com\n", ""},
		{"leading separator", "---\nmethod: POST\nurl: https://e.com\n", ""},
		{"trailing separator", "method: POST\nurl: https://e.com\n---\n", ""},
		{"leading and trailing separators", "---\nmethod: POST\n---\n", ""},
		{"two trailing separators", "method: POST\n---\n---\n", ""},
		{"trailing separator then blank lines", "method: POST\n---\n\n\n", ""},
		{"stray line splits the mapping", "method: POST\n// stray\nurl: https://e.com\n", "line 2"},
		{"deliberate second document", "method: POST\n---\nurl: https://e.com\n", "line 2"},
		{"second document is an explicit null", "method: POST\n---\nnull\n", "line 2"},
		{"split after a leading separator", "---\nmethod: POST\n// stray\n", "line 3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSingleYAMLDoc("req.hk.yaml", []byte(tt.content))
			if tt.wantLine == "" {
				if err != nil {
					t.Fatalf("want no error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("want an error, got nil")
			}
			if !strings.Contains(err.Error(), "req.hk.yaml") {
				t.Errorf("error should name the file, got %q", err)
			}
			if !strings.Contains(err.Error(), tt.wantLine) {
				t.Errorf("error should name %s, got %q", tt.wantLine, err)
			}
		})
	}
}

package utils

import (
	"path/filepath"
	"testing"
)

func TestSiblingPath(t *testing.T) {
	tests := []struct {
		name        string
		currentFile string
		arg         string
		want        string
		wantOK      bool
	}{
		{
			name:        "gql sibling from .hk.yaml",
			currentFile: filepath.Join("genesis", "getUser.hk.yaml"),
			arg:         "*.gql",
			want:        filepath.Join("genesis", "getUser.gql"),
			wantOK:      true,
		},
		{
			name:        "json sibling",
			currentFile: filepath.Join("genesis", "getUser.hk.yaml"),
			arg:         "*.json",
			want:        filepath.Join("genesis", "getUser.json"),
			wantOK:      true,
		},
		{
			name:        "multi-dot extension keeps everything after star",
			currentFile: filepath.Join("a", "b", "backup.hk.yaml"),
			arg:         "*.tar.gz",
			want:        filepath.Join("a", "b", "backup.tar.gz"),
			wantOK:      true,
		},
		{
			name:        "hk.yml suffix stripped",
			currentFile: "getUser.hk.yml",
			arg:         "*.gql",
			want:        "getUser.gql",
			wantOK:      true,
		},
		{
			name:        "plain yaml suffix stripped",
			currentFile: "getUser.yaml",
			arg:         "*.gql",
			want:        "getUser.gql",
			wantOK:      true,
		},
		{
			name:        "yml suffix stripped",
			currentFile: "getUser.yml",
			arg:         "*.gql",
			want:        "getUser.gql",
			wantOK:      true,
		},
		{
			name:        "longest suffix wins: .hk.yaml not .yaml",
			currentFile: "report.hk.yaml",
			arg:         "*.gql",
			want:        "report.gql",
			wantOK:      true,
		},
		{
			name:        "arbitrary literal suffix after star",
			currentFile: "getUser.hk.yaml",
			arg:         "*-response.json",
			want:        "getUser-response.json",
			wantOK:      true,
		},
		{
			name:        "non-star arg is not a sibling",
			currentFile: "getUser.hk.yaml",
			arg:         "genesis/other.gql",
			want:        "",
			wantOK:      false,
		},
		{
			name:        "mid-path star is literal, not a sibling",
			currentFile: "getUser.hk.yaml",
			arg:         "sub/*.gql",
			want:        "",
			wantOK:      false,
		},
		{
			name:        "bare star resolves to stem with no extension",
			currentFile: filepath.Join("genesis", "getUser.hk.yaml"),
			arg:         "*",
			want:        filepath.Join("genesis", "getUser"),
			wantOK:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := SiblingPath(tt.currentFile, tt.arg)
			if ok != tt.wantOK {
				t.Fatalf("SiblingPath(%q, %q) ok = %v, want %v", tt.currentFile, tt.arg, ok, tt.wantOK)
			}
			if got != tt.want {
				t.Errorf("SiblingPath(%q, %q) = %q, want %q", tt.currentFile, tt.arg, got, tt.want)
			}
		})
	}
}

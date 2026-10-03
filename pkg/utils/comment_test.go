package utils

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestF1_TemplateCommentIsNotAVariableReference(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    []string
	}{
		{
			"comment only",
			"url: http://example.com\nheaders:\n  X: \"{{/* staging .token */}}ok\"\n",
			nil,
		},
		{
			"comment beside a real reference",
			"url: \"{{.baseUrl}}\"\nheaders:\n  X: \"{{/* .ignored */}}{{.token}}\"\n",
			[]string{"baseUrl", "token"},
		},
		{
			"trim markers on a comment",
			"url: http://example.com\nheaders:\n  X: \"{{- /* .token */ -}}ok\"\n",
			nil,
		},
		{
			"a glob in a quoted argument does not make the action a comment",
			"url: \"{{.baseUrl}}\"\nheaders:\n  X: \"{{printf \\\"a/*b\\\" .token}}\"\n",
			[]string{"baseUrl", "token"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			p := filepath.Join(dir, "c.hk.yaml")
			if err := os.WriteFile(p, []byte(tc.content), 0o600); err != nil {
				t.Fatal(err)
			}

			env, _, err := RequestVariables(p)
			if err != nil {
				t.Fatalf("RequestVariables: %v", err)
			}
			if !slices.Equal(env, tc.want) {
				t.Errorf("env vars = %v, want %v", env, tc.want)
			}

			wantEnvLoad := len(tc.want) > 0
			if got := FileHasTemplateVars(p); got != wantEnvLoad {
				t.Errorf("FileHasTemplateVars = %v, want %v", got, wantEnvLoad)
			}
		})
	}
}

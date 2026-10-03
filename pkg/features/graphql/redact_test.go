package graphql

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestL11_ProcessGraphQLFileRedactsThePreflightURL(t *testing.T) {
	const secret = "super-secret-client-value"

	dir := t.TempDir()
	path := filepath.Join(dir, "broken.hk.yaml")
	content := "kind: GraphQL\nmethod: POST\nurl: \"not a url {{.api_token}}\"\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := ProcessGraphQLFile(path, map[string]any{"api_token": secret})
	if err == nil {
		t.Fatal("want a validation error for an invalid URL, got nil")
	}
	if strings.Contains(err.Error(), secret) {
		t.Errorf("pre-flight error leaked the resolved secret: %s", err.Error())
	}
}

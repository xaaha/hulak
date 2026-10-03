package graphql

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xaaha/hulak/pkg/utils/testutil"
)

func TestL11_ProcessGraphQLFileRedactsThePreflightURL(t *testing.T) {
	const secret = "super-secret-client-value"

	for name, content := range map[string]string{
		"invalid url":    "kind: GraphQL\nmethod: POST\nurl: \"not a url {{.api_token}}\"\n",
		"invalid method": "kind: GraphQL\nmethod: \"{{.api_token}}\"\nurl: \"https://api.example.com\"\n",
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "broken.hk.yaml")
			if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}

			_, err := ProcessGraphQLFile(path, map[string]any{"api_token": secret})
			if err == nil {
				t.Fatal("want a validation error, got nil")
			}
			testutil.AssertNoSecretForm(t, "GraphQL pre-flight error", err.Error(), secret)
		})
	}
}

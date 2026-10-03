package features

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/utils/testutil"
)

func TestAuth2RedactorFollowsShow(t *testing.T) {
	const secret = "super-secret-client-value"
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, utils.EnvironmentFolder), utils.DirPer); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	path := filepath.Join(dir, "auth.hk.yaml")
	if err := os.WriteFile(path, []byte(
		"kind: Auth\nmethod: POST\nurl: \"https://api.example.com/token\"\n"+
			"urlparams:\n  client_secret: \"{{.client_secret}}\"\n"), utils.FilePer,
	); err != nil {
		t.Fatal(err)
	}
	secrets := map[string]any{"client_secret": secret}

	masked, err := auth2Redactor(path, secrets, false)
	if err != nil {
		t.Fatalf("auth2Redactor: %v", err)
	}
	if strings.Contains(masked.Redact("secret="+secret), secret) {
		t.Error("without --show the auth2 echo must stay masked")
	}

	shown, err := auth2Redactor(path, secrets, true)
	if err != nil {
		t.Fatalf("auth2Redactor: %v", err)
	}
	if shown != nil {
		t.Error("--show must reach the auth2 path and mask nothing")
	}
}

// auth2ProjectFile writes an auth request into a fresh project root.
func auth2ProjectFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, utils.EnvironmentFolder), utils.DirPer); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	path := filepath.Join(dir, "auth.hk.yaml")
	if err := os.WriteFile(path, []byte(content), utils.FilePer); err != nil {
		t.Fatal(err)
	}
	return path
}

// The auth2 flow parses the file twice and builds the request from it, all
// before the redactor used to exist. goccy embeds the substituted source line
// in a decode error, so the whole resolved value reached the user.
func TestL14_Auth2PreflightErrorsAreRedacted(t *testing.T) {
	const secret = "super-secret-client-value"
	const header = "kind: Auth\nmethod: POST\n" +
		"url: \"https://api.example.com/authorize\"\n" +
		"auth:\n  type: oauth2\n" +
		"  access_token_url: \"https://api.example.com/token\"\n"
	const undecodable = header + "urlparams: \"{{.client_secret}}\"\n"
	secrets := map[string]any{"client_secret": secret}

	codeRequest := func(t *testing.T, path string, show bool) error {
		t.Helper()
		return SendAPIRequestForAuth2(context.Background(), secrets, path, false, show, nil)
	}
	tokenExchange := func(t *testing.T, path string, show bool) error {
		t.Helper()
		redact, err := auth2Redactor(path, secrets, show)
		if err != nil {
			t.Fatalf("auth2Redactor: %v", err)
		}
		return exchangeAuth2Code(context.Background(), secrets, path, "the-code", false, redact)
	}

	// Only files that fail before the browser step belong on codeRequest: a
	// file that parses would bind the callback port and open a browser.
	for name, tc := range map[string]struct {
		content string
		call    func(t *testing.T, path string, show bool) error
	}{
		"code request decode error":   {undecodable, codeRequest},
		"token exchange decode error": {undecodable, tokenExchange},
	} {
		t.Run(name, func(t *testing.T) {
			path := auth2ProjectFile(t, tc.content)

			err := tc.call(t, path, false)
			if err == nil {
				t.Fatal("expected a pre-flight error")
			}
			testutil.AssertNoSecretForm(t, "auth2 pre-flight error", err.Error(), secret)

			shown := tc.call(t, path, true)
			if shown == nil || !strings.Contains(shown.Error(), secret) {
				t.Errorf("--show must leave the auth2 error alone: %v", shown)
			}
		})
	}
}

package features

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xaaha/hulak/pkg/utils"
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

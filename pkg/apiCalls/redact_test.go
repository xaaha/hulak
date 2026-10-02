package apicalls

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/yamlparser"
)

// chdirToProject moves into a fresh project root so vault.DetectStore sees
// only what the test puts there.
func chdirToProject(t *testing.T, withVault bool) {
	t.Helper()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Fatal(err)
		}
	})

	tmpDir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("eval symlinks: %v", err)
	}
	if withVault {
		hulakDir := filepath.Join(tmpDir, utils.HiddenProjectName)
		if err := os.Mkdir(hulakDir, utils.DirPer); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(
			filepath.Join(hulakDir, utils.StoreFile), []byte("encrypted"), utils.SecretPer,
		); err != nil {
			t.Fatal(err)
		}
	} else if err := os.Mkdir(filepath.Join(tmpDir, utils.EnvironmentFolder), utils.DirPer); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
}

func TestD1_1_SecretProvenanceFollowsVaultStore(t *testing.T) {
	const plainValue = "https://api.example.com/v1/users"
	const secretValue = "super-secret-client-value"
	secrets := map[string]any{"base_url": plainValue, "client_secret": secretValue}

	t.Run("vault store makes every resolved value a secret", func(t *testing.T) {
		chdirToProject(t, true)
		got := NewSecretRedactor(secrets).Redact(plainValue + " " + secretValue)
		if strings.Contains(got, plainValue) {
			t.Errorf("vault-backed value left in clear: %q", got)
		}
		if strings.Contains(got, secretValue) {
			t.Errorf("vault-backed secret left in clear: %q", got)
		}
	})

	t.Run("plain env files fall back to the key name", func(t *testing.T) {
		chdirToProject(t, false)
		got := NewSecretRedactor(secrets).Redact(plainValue + " " + secretValue)
		if !strings.Contains(got, plainValue) {
			t.Errorf("non-secret key must not be masked without a vault: %q", got)
		}
		if strings.Contains(got, secretValue) {
			t.Errorf("secret-named key must still be masked: %q", got)
		}
	})
}

func TestD1_2_ValueMaskingAddsToHeaderNameMasking(t *testing.T) {
	const headerToken = "ya29.a0AfH6SMB-never-in-the-secrets-map"
	const bodySecret = "super-secret-client-value"
	info := &yamlparser.APIInfo{
		Method:    "POST",
		URL:       "https://api.example.com/token",
		URLParams: map[string]string{"client_secret": bodySecret},
		Headers: map[string]string{
			"Authorization": "Bearer " + headerToken,
			"Content-Type":  "application/x-www-form-urlencoded",
		},
		Body: strings.NewReader("grant_type=client_credentials&client_secret=" + bodySecret),
	}
	redactor := utils.NewValueRedactor(map[string]any{"client_secret": bodySecret}, true)

	out, err := FormatDryRun(info, false, redactor)
	if err != nil {
		t.Fatalf("FormatDryRun: %v", err)
	}

	if strings.Contains(out, headerToken) {
		t.Errorf("header-name masking must still cover a token absent from the secrets map:\n%s", out)
	}
	if !strings.Contains(out, "Authorization: "+utils.MaskedValue+"\n") {
		t.Errorf("Authorization should stay masked by header name:\n%s", out)
	}
	if strings.Contains(out, bodySecret) {
		t.Errorf("resolved secret leaked into the body or query string:\n%s", out)
	}
	if !strings.Contains(out, utils.MaskedValue+"(25 chars, #") {
		t.Errorf("expected the value mask in the output:\n%s", out)
	}
}

package apicalls

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xaaha/hulak/pkg/httpclient"
	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/yamlparser"
)

// leakyValues exercise the transforms a rendered request applies to a value:
// percent-encoding in a query string or urlencoded body, and JSON string
// escaping in a JSON body.
var leakyValues = []string{
	"super-secret-client-value",
	"Zm9vYmFy/c2VjcmV0+dmFsdWU=",
	`pa$$w"rd-1234567890`,
	`with space and \ backslash`,
}

// assertNoSecretForm fails when out carries value in any spelling a rendered
// request can produce, not just the verbatim one.
func assertNoSecretForm(t *testing.T, where, out, value string) {
	t.Helper()
	quoted, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for form, text := range map[string]string{
		"raw":         value,
		"queryEscape": url.QueryEscape(value),
		"pathEscape":  url.PathEscape(value),
		"jsonEscape":  string(quoted[1 : len(quoted)-1]),
	} {
		if strings.Contains(out, text) {
			t.Errorf("%s leaked the %s form of the secret:\n%s", where, form, out)
		}
	}
}

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
	for _, bodySecret := range leakyValues {
		t.Run(bodySecret, func(t *testing.T) {
			info := &yamlparser.APIInfo{
				Method:    "POST",
				URL:       "https://api.example.com/token",
				URLParams: map[string]string{"client_secret": bodySecret},
				Headers: map[string]string{
					"Authorization": "Bearer " + headerToken,
					"Content-Type":  "application/x-www-form-urlencoded",
				},
				Body: strings.NewReader(url.Values{
					"grant_type":    {"client_credentials"},
					"client_secret": {bodySecret},
				}.Encode()),
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
			assertNoSecretForm(t, "dry run", out, bodySecret)
			if !strings.Contains(out, fmt.Sprintf("%s(%d chars, #", utils.MaskedValue, len(bodySecret))) {
				t.Errorf("expected the value mask in the output:\n%s", out)
			}
		})
	}
}

func TestD1_3_DebugMasksRequestAndShowReveals(t *testing.T) {
	client := &MockHTTPClient{
		DoFunc: func(_ *http.Request) (*http.Response, error) {
			return NewMockResponse(200, `{"ok":true}`), nil
		},
	}
	for _, secret := range leakyValues {
		t.Run(secret, func(t *testing.T) {
			newInfo := func() yamlparser.APIInfo {
				return yamlparser.APIInfo{
					Method:    "POST",
					URL:       "https://api.example.com/token",
					URLParams: map[string]string{"client_secret": secret},
					Headers:   map[string]string{"Authorization": "Bearer " + secret},
					Body: strings.NewReader(
						url.Values{"client_secret": {secret}}.Encode(),
					),
				}
			}
			redactor := utils.NewValueRedactor(map[string]any{"client_secret": secret}, true)

			masked, err := StandardCallWithClient(
				context.Background(), newInfo(), true, redactor, client,
			)
			if err != nil {
				t.Fatalf("StandardCallWithClient: %v", err)
			}
			if masked.Request == nil {
				t.Fatal("debug call must carry request info")
			}
			for field, got := range map[string]string{
				"url":           masked.Request.URL,
				"authorization": masked.Request.Headers["Authorization"],
				"body":          fmt.Sprint(masked.Request.Body),
			} {
				assertNoSecretForm(t, "debug "+field, got, secret)
			}

			shown, err := StandardCallWithClient(context.Background(), newInfo(), true, nil, client)
			if err != nil {
				t.Fatalf("StandardCallWithClient: %v", err)
			}
			if !strings.Contains(fmt.Sprint(shown.Request.Body), url.QueryEscape(secret)) {
				t.Errorf("show must reveal the body, got %q", shown.Request.Body)
			}
			if !strings.Contains(shown.Request.Headers["Authorization"], secret) {
				t.Errorf("show must reveal the header, got %q", shown.Request.Headers["Authorization"])
			}
		})
	}
}

func TestD1_4_EmptyResolvedVariablesListedInFooter(t *testing.T) {
	newInfo := func() *yamlparser.APIInfo {
		return &yamlparser.APIInfo{
			Method:  "POST",
			URL:     "https://api.example.com/token",
			Headers: map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			Body:    strings.NewReader("client_secret=&tenant_id="),
		}
	}

	t.Run("names every variable that resolved empty", func(t *testing.T) {
		redactor := utils.NewValueRedactor(map[string]any{
			"client_secret": "",
			"tenant_id":     "",
			"api_token":     "a-value-that-resolved",
		}, true)
		out, err := FormatDryRun(newInfo(), false, redactor)
		if err != nil {
			t.Fatalf("FormatDryRun: %v", err)
		}
		if !strings.HasSuffix(out, "// unresolved: client_secret, tenant_id\n") {
			t.Errorf("expected the unresolved footer last, got:\n%s", out)
		}
	})

	t.Run("stays quiet when everything resolved", func(t *testing.T) {
		redactor := utils.NewValueRedactor(
			map[string]any{"api_token": "a-value-that-resolved"}, true,
		)
		out, err := FormatDryRun(newInfo(), false, redactor)
		if err != nil {
			t.Fatalf("FormatDryRun: %v", err)
		}
		if strings.Contains(out, "unresolved") {
			t.Errorf("no variable resolved empty, footer should be absent:\n%s", out)
		}
	})
}

func TestLeak3_TransportAndBuildErrorsAreRedacted(t *testing.T) {
	const secret = "super-secret-client-value"
	redactor := func() *utils.ValueRedactor {
		return utils.NewValueRedactor(map[string]any{"client_secret": secret}, true)
	}

	t.Run("unreachable host", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, err := StandardCallWithClient(ctx, yamlparser.APIInfo{
			Method:    "GET",
			URL:       "http://127.0.0.1:1/token",
			URLParams: map[string]string{"client_secret": secret},
		}, false, redactor(), httpclient.New())
		if err == nil {
			t.Fatal("expected a transport error from an unreachable host")
		}
		assertNoSecretForm(t, "transport error", err.Error(), secret)
	})

	t.Run("unbuildable request", func(t *testing.T) {
		_, err := StandardCallWithClient(context.Background(), yamlparser.APIInfo{
			Method: "GET",
			URL:    "http://api.example.com/\x7f" + secret,
		}, false, redactor(), httpclient.New())
		if err == nil {
			t.Fatal("expected a request-build error from an unparseable URL")
		}
		assertNoSecretForm(t, "build error", err.Error(), secret)
	})

	t.Run("show leaves the error alone", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, err := StandardCallWithClient(ctx, yamlparser.APIInfo{
			Method:    "GET",
			URL:       "http://127.0.0.1:1/token",
			URLParams: map[string]string{"client_secret": secret},
		}, false, nil, httpclient.New())
		if err == nil {
			t.Fatal("expected a transport error from an unreachable host")
		}
		if !strings.Contains(err.Error(), url.QueryEscape(secret)) {
			t.Errorf("a nil redactor must leave the error untouched: %v", err)
		}
	})
}

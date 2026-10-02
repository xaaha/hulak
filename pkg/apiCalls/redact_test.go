package apicalls

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/xaaha/hulak/pkg/httpclient"
	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/yamlparser"
)

// leakyValues carry characters a rendered request percent- or JSON-escapes.
var leakyValues = []string{
	"super-secret-client-value",
	"Zm9vYmFy/c2VjcmV0+dmFsdWU=",
	`pa$$w"rd-1234567890`,
	`with space and \ backslash`,
}

// assertNoSecretForm rejects every spelling of value, not just the verbatim one.
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

// chdirToProject moves into a fresh project root for vault.DetectStore.
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

// writeRequestFile drops a request referencing every name, for footer scoping.
func writeRequestFile(t *testing.T, names ...string) string {
	t.Helper()
	content := "kind: API\nmethod: GET\nurl: \"https://api.example.com\"\nurlparams:\n"
	for i, name := range names {
		content += fmt.Sprintf("  p%d: \"{{.%s}}\"\n", i, name)
	}
	path := filepath.Join(t.TempDir(), "request.hk.yaml")
	if err := os.WriteFile(path, []byte(content), utils.FilePer); err != nil {
		t.Fatal(err)
	}
	return path
}

func newSecretRedactor(t *testing.T, path string, secrets map[string]any) *utils.ValueRedactor {
	t.Helper()
	r, err := NewSecretRedactor(path, secrets)
	if err != nil {
		t.Fatalf("NewSecretRedactor: %v", err)
	}
	return r
}

func TestD1_1_SecretProvenanceFollowsVaultStore(t *testing.T) {
	const plainValue = "https://api.example.com/v1/users"
	const secretValue = "super-secret-client-value"
	secrets := map[string]any{"base_url": plainValue, "client_secret": secretValue}

	t.Run("vault store makes every resolved value a secret", func(t *testing.T) {
		chdirToProject(t, true)
		path := writeRequestFile(t, "base_url", "client_secret")
		got := newSecretRedactor(t, path, secrets).Redact(plainValue + " " + secretValue)
		if strings.Contains(got, plainValue) {
			t.Errorf("vault-backed value left in clear: %q", got)
		}
		if strings.Contains(got, secretValue) {
			t.Errorf("vault-backed secret left in clear: %q", got)
		}
	})

	t.Run("plain env files fall back to the key name", func(t *testing.T) {
		chdirToProject(t, false)
		path := writeRequestFile(t, "base_url", "client_secret")
		got := newSecretRedactor(t, path, secrets).Redact(plainValue + " " + secretValue)
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
			redactor := utils.NewValueRedactor(map[string]any{"client_secret": bodySecret}, true, nil)

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

// pathLeakyValues render differently under net/url's path encoding than under
// PathEscape: they combine one of / ; , with a character both escape.
var pathLeakyValues = []string{
	"pa55 word/with slash",
	"токен/значение",
	"alpha,beta gamma,delta",
}

func TestLeak4_SecretInURLPathIsMasked(t *testing.T) {
	for _, secret := range pathLeakyValues {
		t.Run(secret, func(t *testing.T) {
			info := &yamlparser.APIInfo{
				Method: "GET",
				URL:    "https://api.example.com/v1/" + secret + "/profile",
			}
			out, err := FormatDryRun(
				info, false,
				utils.NewValueRedactor(map[string]any{"client_secret": secret}, true, nil),
			)
			if err != nil {
				t.Fatalf("FormatDryRun: %v", err)
			}
			if rendered := (&url.URL{Path: secret}).EscapedPath(); strings.Contains(out, rendered) {
				t.Errorf("dry run leaked the URL-path form %q:\n%s", rendered, out)
			}
			if !strings.Contains(out, fmt.Sprintf("%s(%d chars, #", utils.MaskedValue, len(secret))) {
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
			redactor := utils.NewValueRedactor(map[string]any{"client_secret": secret}, true, nil)

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

func TestD1_4a_EmptyReferencedVariablesListedInFooter(t *testing.T) {
	newInfo := func() *yamlparser.APIInfo {
		return &yamlparser.APIInfo{
			Method:  "POST",
			URL:     "https://api.example.com/token",
			Headers: map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			Body:    strings.NewReader("client_secret=&tenant_id="),
		}
	}
	values := map[string]any{
		"client_secret": "",
		"tenant_id":     "",
		"unused_token":  "",
		"api_token":     "a-value-that-resolved",
	}

	t.Run("names every referenced variable that resolved empty", func(t *testing.T) {
		redactor := utils.NewValueRedactor(
			values, true, []string{"api_token", "client_secret", "tenant_id"},
		)
		out, err := FormatDryRun(newInfo(), false, redactor)
		if err != nil {
			t.Fatalf("FormatDryRun: %v", err)
		}
		if !strings.HasSuffix(out, "// unresolved: client_secret, tenant_id\n") {
			t.Errorf("expected the unresolved footer last, got:\n%s", out)
		}
	})

	t.Run("classification does not gate the footer", func(t *testing.T) {
		redactor := utils.NewValueRedactor(
			values, false, []string{"api_token", "client_secret", "tenant_id"},
		)
		out, err := FormatDryRun(newInfo(), false, redactor)
		if err != nil {
			t.Fatalf("FormatDryRun: %v", err)
		}
		if !strings.HasSuffix(out, "// unresolved: client_secret, tenant_id\n") {
			t.Errorf("tenant_id matches no secret hint but is still unresolved, got:\n%s", out)
		}
	})

	t.Run("stays quiet when everything referenced resolved", func(t *testing.T) {
		redactor := utils.NewValueRedactor(values, true, []string{"api_token"})
		out, err := FormatDryRun(newInfo(), false, redactor)
		if err != nil {
			t.Fatalf("FormatDryRun: %v", err)
		}
		if strings.Contains(out, "unresolved") {
			t.Errorf("an empty variable the request never references must not be named:\n%s", out)
		}
	})
}

func TestLeak3_TransportAndBuildErrorsAreRedacted(t *testing.T) {
	const secret = "super-secret-client-value"
	redactor := func() *utils.ValueRedactor {
		return utils.NewValueRedactor(map[string]any{"client_secret": secret}, true, nil)
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

// A getValueOf-class token never reaches the secrets map: only names cover it.
func TestD1_2_DebugMasksHeaderNamesToo(t *testing.T) {
	const headerToken = "ya29.a0AfH6SMB-never-in-the-secrets-map"
	const bodySecret = "super-secret-client-value"
	client := &MockHTTPClient{
		DoFunc: func(_ *http.Request) (*http.Response, error) {
			return NewMockResponse(200, `{"ok":true}`), nil
		},
	}
	info := yamlparser.APIInfo{
		Method: "POST",
		URL:    "https://api.example.com/token",
		Headers: map[string]string{
			"Authorization": "Bearer " + headerToken,
			"Cookie":        "session=" + headerToken,
			"Content-Type":  "application/x-www-form-urlencoded",
		},
		Body: strings.NewReader(url.Values{"client_secret": {bodySecret}}.Encode()),
	}
	redactor := utils.NewValueRedactor(map[string]any{"client_secret": bodySecret}, true, nil)

	resp, err := StandardCallWithClient(context.Background(), info, true, redactor, client)
	if err != nil {
		t.Fatalf("StandardCallWithClient: %v", err)
	}
	if resp.Request == nil {
		t.Fatal("debug call must carry request info")
	}
	for _, name := range []string{"Authorization", "Cookie"} {
		if got := resp.Request.Headers[name]; got != utils.MaskedValue {
			t.Errorf("%s must be masked by header name, got %q", name, got)
		}
	}
	if resp.Request.Headers["Content-Type"] != "application/x-www-form-urlencoded" {
		t.Errorf("a non-sensitive header must survive: %q", resp.Request.Headers["Content-Type"])
	}
}

// Show must turn value masking off, not just header-name masking.
func TestD1_3_ShowRevealsAtTheRedactorSeam(t *testing.T) {
	const secret = "super-secret-client-value"
	chdirToProject(t, true)
	path := writeRequestFile(t, "client_secret")
	opts := RequestOptions{
		Secrets: map[string]any{"client_secret": secret},
		Path:    path,
	}

	masked, err := outputRedactor(opts)
	if err != nil {
		t.Fatalf("outputRedactor: %v", err)
	}
	if masked == nil {
		t.Fatal("without Show the output must be masked")
	}
	if strings.Contains(masked.Redact("secret="+secret), secret) {
		t.Error("without Show the resolved secret must not survive")
	}

	opts.Show = true
	shown, err := outputRedactor(opts)
	if err != nil {
		t.Fatalf("outputRedactor: %v", err)
	}
	if shown != nil {
		t.Error("Show must mask nothing")
	}
}

func TestD1_3_ShowRevealsThroughDryRun(t *testing.T) {
	const secret = "super-secret-client-value"
	chdirToProject(t, true)
	path := writeRequestFile(t, "client_secret")

	out, err := DryRun(RequestOptions{
		Secrets: map[string]any{"client_secret": secret},
		Path:    path,
		Show:    true,
	})
	if err != nil {
		t.Fatalf("DryRun: %v", err)
	}
	if !strings.Contains(out, url.QueryEscape(secret)) {
		t.Errorf("Show must leave the resolved secret in the clear:\n%s", out)
	}
}

// An empty value inside a name-masked header is byte-identical to a resolved one.
func TestD1_4a_DebugSurfaceCarriesUnresolved(t *testing.T) {
	client := &MockHTTPClient{
		DoFunc: func(_ *http.Request) (*http.Response, error) {
			return NewMockResponse(200, `{"ok":true}`), nil
		},
	}
	info := yamlparser.APIInfo{
		Method:  "POST",
		URL:     "https://api.example.com/token",
		Headers: map[string]string{"Authorization": "Bearer "},
		Body:    strings.NewReader("client_secret="),
	}
	redactor := utils.NewValueRedactor(map[string]any{
		"client_secret": "",
		"tenant_id":     "",
		"unused_token":  "",
	}, true, []string{"client_secret", "tenant_id"})

	resp, err := StandardCallWithClient(context.Background(), info, true, redactor, client)
	if err != nil {
		t.Fatalf("StandardCallWithClient: %v", err)
	}
	if resp.Request == nil {
		t.Fatal("debug call must carry request info")
	}
	want := []string{"client_secret", "tenant_id"}
	if !slices.Equal(resp.Request.Unresolved, want) {
		t.Errorf("Request.Unresolved = %v, want %v", resp.Request.Unresolved, want)
	}

	shown, err := StandardCallWithClient(context.Background(), info, true, nil, client)
	if err != nil {
		t.Fatalf("StandardCallWithClient: %v", err)
	}
	if shown.Request.Unresolved != nil {
		t.Errorf("a nil redactor reports nothing, got %v", shown.Request.Unresolved)
	}
}

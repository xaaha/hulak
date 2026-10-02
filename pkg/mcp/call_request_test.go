package mcp

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xaaha/hulak/pkg/utils/testutil"
)

func TestHandleCallRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	newProjectWithReq := func(t *testing.T) (string, *Server) {
		t.Helper()
		api := projectDir(t)
		writeFileAt(t, filepath.Join(api, "ping.hk.yaml"),
			"kind: API\nmethod: GET\nurl: "+srv.URL+"\n")
		s, err := NewServer(map[string]string{"api": api}, "v")
		if err != nil {
			t.Fatal(err)
		}
		return api, s
	}

	ctx := context.Background()

	t.Run("sends and returns response", func(t *testing.T) {
		_, s := newProjectWithReq(t)
		_, out, err := s.handleCallRequest(ctx, nil, callRequestInput{Name: "ping", Env: "global"})
		if err != nil {
			t.Fatal(err)
		}
		if out.Status != "200 OK" {
			t.Errorf("status = %q, want 200 OK", out.Status)
		}
		if !strings.Contains(out.Body, `"ok"`) {
			t.Errorf("body should contain the response, got: %s", out.Body)
		}
	})

	t.Run("does not save by default", func(t *testing.T) {
		api, s := newProjectWithReq(t)
		if _, _, err := s.handleCallRequest(ctx, nil, callRequestInput{Name: "ping", Env: "global"}); err != nil {
			t.Fatal(err)
		}
		if matches, _ := filepath.Glob(filepath.Join(api, "*_response.*")); len(matches) != 0 {
			t.Errorf("agent call should not write a response file by default, found: %v", matches)
		}
	})

	t.Run("save=true writes the response file", func(t *testing.T) {
		api, s := newProjectWithReq(t)
		if _, _, err := s.handleCallRequest(ctx, nil, callRequestInput{Name: "ping", Env: "global", Save: true}); err != nil {
			t.Fatal(err)
		}
		if matches, _ := filepath.Glob(filepath.Join(api, "*_response.*")); len(matches) == 0 {
			t.Error("save=true should write a response file")
		}
	})

	t.Run("env is required", func(t *testing.T) {
		_, s := newProjectWithReq(t)
		if _, _, err := s.handleCallRequest(ctx, nil, callRequestInput{Name: "ping"}); err == nil {
			t.Error("expected error when env is missing")
		}
	})
}

func TestHandleCallRequest_Debug(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	api := projectDir(t)
	writeFileAt(t, filepath.Join(api, "ping.hk.yaml"), "kind: API\nmethod: GET\nurl: "+srv.URL+"\n")
	s, err := NewServer(map[string]string{"api": api}, "v")
	if err != nil {
		t.Fatal(err)
	}

	_, out, err := s.handleCallRequest(context.Background(), nil,
		callRequestInput{Name: "ping", Env: "global", Debug: true})
	if err != nil {
		t.Fatal(err)
	}
	// Debug body is the full CustomResponse: includes request + http_info keys.
	for _, want := range []string{`"request"`, `"http_info"`} {
		if !strings.Contains(out.Body, want) {
			t.Errorf("debug body missing %s, got:\n%s", want, out.Body)
		}
	}
}

func TestHandleCallRequest_Timeout(t *testing.T) {
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
		_, _ = w.Write([]byte("ok"))
	}))
	defer slow.Close()

	api := projectDir(t)
	writeFileAt(t, filepath.Join(api, "slow.hk.yaml"), "kind: API\nmethod: GET\nurl: "+slow.URL+"\n")
	s, err := NewServer(map[string]string{"api": api}, "v")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	t.Run("short timeout aborts the call", func(t *testing.T) {
		_, _, err := s.handleCallRequest(ctx, nil,
			callRequestInput{Name: "slow", Env: "global", Timeout: "50ms"})
		if err == nil {
			t.Error("expected timeout error for a slow request")
		}
	})

	t.Run("ample timeout succeeds", func(t *testing.T) {
		_, out, err := s.handleCallRequest(ctx, nil,
			callRequestInput{Name: "slow", Env: "global", Timeout: "5s"})
		if err != nil {
			t.Fatalf("expected success with ample timeout, got: %v", err)
		}
		if out.Status == "" {
			t.Error("expected a status")
		}
	})

	t.Run("invalid timeout string errors", func(t *testing.T) {
		_, _, err := s.handleCallRequest(ctx, nil,
			callRequestInput{Name: "slow", Env: "global", Timeout: "nope"})
		if err == nil {
			t.Error("expected error for invalid timeout")
		}
	})
}

// One tool call saves a fresh auth response, the next one reads the token out
// of it with getValueOf. The server is a single process serving both calls, so
// a getValueOf result held across calls sends the expired token forever.
func TestHandleCallRequest_SavedTokenVisibleToNextCall(t *testing.T) {
	currentToken := "fresh-token"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/auth" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"access_token":"` + currentToken + `"}`))
			return
		}
		if r.Header.Get("Authorization") != "Bearer "+currentToken {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"stale token"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"plan":"pro"}`))
	}))
	defer srv.Close()

	api := projectDir(t)
	writeFileAt(t, filepath.Join(api, "getAuth.hk.yaml"),
		"kind: API\nmethod: GET\nurl: "+srv.URL+"/auth\n")
	writeFileAt(t, filepath.Join(api, "getUserPlan.hk.yaml"),
		"kind: API\nmethod: GET\nurl: "+srv.URL+"/plan\nheaders:\n"+
			"  Authorization: 'Bearer {{getValueOf \"access_token\" \"getAuth.hk_response.json\"}}'\n")
	writeFileAt(t, filepath.Join(api, "getAuth.hk_response.json"),
		`{"access_token":"expired-token"}`)

	s, err := NewServer(map[string]string{"api": api}, "v")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	// Reads the expired token that is on disk.
	_, out, err := s.handleCallRequest(ctx, nil, callRequestInput{Name: "getUserPlan", Env: "global"})
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != "401 Unauthorized" {
		t.Fatalf("first call: status = %q, want 401 Unauthorized", out.Status)
	}

	// Saves the fresh token over that file.
	if _, _, err := s.handleCallRequest(
		ctx, nil, callRequestInput{Name: "getAuth", Env: "global", Save: true},
	); err != nil {
		t.Fatal(err)
	}

	_, out, err = s.handleCallRequest(ctx, nil, callRequestInput{Name: "getUserPlan", Env: "global"})
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != "200 OK" {
		t.Errorf("call after the token was saved: status = %q, want 200 OK", out.Status)
	}
}

// leakyValues carry characters a rendered request percent- or JSON-escapes.
// The last five separate net/url's escaping tables from each other.
var leakyValues = []string{
	"super-secret-client-value",
	"Zm9vYmFy/c2VjcmV0+dmFsdWU=",
	`pa$$w"rd-1234567890`,
	"pa55 word/with slash",
	"токен/значение",
	"st?te tok!n Value",
	"p@ss;w0rd=Secret1",
	"p#ss word-1234",
}

// headerToken is never in the secrets map, so only header-name masking hides it.
const headerToken = "ya29.a0AfH6SMB-never-in-the-secrets-map"

// requestFixtures build the same secret into each place a request carries one,
// so no position is covered only where an older fixture happened to put it.
func requestFixtures() map[string]string {
	auth := "headers:\n  Authorization: \"Bearer " + headerToken + "\"\n"
	return map[string]string{
		"url path": "kind: API\nmethod: GET\n" +
			"url: \"{{.baseUrl}}/v1/{{.client_secret}}/profile\"\n" + auth,
		"url param": "kind: API\nmethod: POST\nurl: \"{{.baseUrl}}\"\n" + auth +
			"urlparams:\n  client_secret: \"{{.client_secret}}\"\n",
		"json body": "kind: API\nmethod: POST\nurl: \"{{.baseUrl}}\"\n" + auth +
			"  Content-Type: application/json\n" +
			"body:\n  raw: '{\"client_secret\": \"{{.client_secret}}\"}'\n",
		"form body": "kind: API\nmethod: POST\nurl: \"{{.baseUrl}}\"\n" + auth +
			"body:\n  urlencodedformdata:\n    client_secret: \"{{.client_secret}}\"\n",
		"url fragment": "kind: API\nmethod: GET\n" +
			"url: \"{{.baseUrl}}/cb#{{.client_secret}}\"\n" + auth,
		"url host": "kind: API\nmethod: GET\n" +
			"url: \"http://{{.client_secret}}.invalid/v1/me\"\n" + auth,
		"url userinfo": "kind: API\nmethod: GET\n" +
			"url: \"http://admin:{{.client_secret}}@{{.baseHost}}/v1/me\"\n" + auth,
	}
}

// Neither MCP tool that renders a request may hand the agent a secret in clear.
func TestD1_3_MCPMasksSecretsOnBothSurfaces(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	for _, secret := range leakyValues {
		t.Run(secret, func(t *testing.T) {
			for position, content := range requestFixtures() {
				t.Run(position, func(t *testing.T) {
					api := projectDir(t)
					writeFileAt(t, filepath.Join(api, "env", "staging.env"),
						"baseUrl="+srv.URL+"\nbaseHost="+strings.TrimPrefix(srv.URL, "http://")+
							"\nclient_secret='"+secret+"'\n")
					writeFileAt(t, filepath.Join(api, "token.hk.yaml"), content)

					s, err := NewServer(map[string]string{"api": api}, "v")
					if err != nil {
						t.Fatal(err)
					}
					ctx := context.Background()
					wantMask := fmt.Sprintf("(%d chars, #", len(secret))

					t.Run("dry_run", func(t *testing.T) {
						_, out, err := s.handleDryRun(ctx, nil, dryRunInput{Name: "token", Env: "staging"})
						if err != nil {
							testutil.AssertNoSecretForm(t, "dry_run error "+position, err.Error(), secret)
							return
						}
						testutil.AssertNoSecretForm(t, "dry_run "+position, out.Request, secret)
						if strings.Contains(out.Request, headerToken) {
							t.Errorf("dry_run leaked a token only header-name masking covers:\n%s", out.Request)
						}
						if !strings.Contains(out.Request, wantMask) {
							t.Errorf("expected a value mask in the dry_run output:\n%s", out.Request)
						}
					})

					t.Run("call_request with debug", func(t *testing.T) {
						_, out, err := s.handleCallRequest(ctx, nil,
							callRequestInput{Name: "token", Env: "staging", Debug: true})
						if err != nil {
							testutil.AssertNoSecretForm(t, "call_request error "+position, err.Error(), secret)
							return
						}
						testutil.AssertNoSecretForm(t, "call_request debug "+position, out.Body, secret)
						if strings.Contains(out.Body, headerToken) {
							t.Errorf("call_request debug leaked a token only header-name masking covers:\n%s", out.Body)
						}
						if !strings.Contains(out.Body, wantMask) {
							t.Errorf("expected a value mask in the debug output:\n%s", out.Body)
						}
					})
				})
			}
		})
	}
}

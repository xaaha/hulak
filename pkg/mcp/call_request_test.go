package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
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
		s, err := NewServer(map[string]string{"api": api}, "api", "v")
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

	t.Run("no_save skips the response file", func(t *testing.T) {
		api, s := newProjectWithReq(t)
		if _, _, err := s.handleCallRequest(ctx, nil, callRequestInput{Name: "ping", Env: "global", NoSave: true}); err != nil {
			t.Fatal(err)
		}
		if matches, _ := filepath.Glob(filepath.Join(api, "*_response.*")); len(matches) != 0 {
			t.Errorf("no_save should write no response file, found: %v", matches)
		}
	})

	t.Run("default saves the response file", func(t *testing.T) {
		api, s := newProjectWithReq(t)
		if _, _, err := s.handleCallRequest(ctx, nil, callRequestInput{Name: "ping", Env: "global"}); err != nil {
			t.Fatal(err)
		}
		if matches, _ := filepath.Glob(filepath.Join(api, "*_response.*")); len(matches) == 0 {
			t.Error("default should write a response file")
		}
	})

	t.Run("env is required", func(t *testing.T) {
		_, s := newProjectWithReq(t)
		if _, _, err := s.handleCallRequest(ctx, nil, callRequestInput{Name: "ping"}); err == nil {
			t.Error("expected error when env is missing")
		}
	})
}

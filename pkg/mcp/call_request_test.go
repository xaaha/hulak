package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

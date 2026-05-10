package vault

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xaaha/hulak/pkg/httpclient"
)

// testClient wraps an httptest TLS server's client into an httpclient.Client.
func testClient(ts *httptest.Server) *httpclient.Client {
	return &httpclient.Client{HTTP: ts.Client()}
}

func TestKeyserverKeysURL(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		username string
		want     string
	}{
		{
			name:     "github",
			baseURL:  GitHubKeysBase,
			username: "octocat",
			want:     "https://github.com/octocat.keys",
		},
		{
			name:     "no trailing slash",
			baseURL:  "https://keys.example.com",
			username: "alice",
			want:     "https://keys.example.com/alice.keys",
		},
		{
			name:     "trailing slash",
			baseURL:  "https://keys.example.com/",
			username: "alice",
			want:     "https://keys.example.com/alice.keys",
		},
		{
			name:     "multiple trailing slashes",
			baseURL:  "https://keys.example.com///",
			username: "bob",
			want:     "https://keys.example.com/bob.keys",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := KeyserverKeysURL(tt.baseURL, tt.username)
			if got != tt.want {
				t.Errorf("KeyserverKeysURL(%q, %q) = %q, want %q", tt.baseURL, tt.username, got, tt.want)
			}
		})
	}
}

func TestFetchKeysFromURL(t *testing.T) {
	ed25519Key := testSSHEd25519PubKey(t)
	rsaKey := testSSHRSAPubKey(t)

	t.Run("fetches and returns key lines", func(t *testing.T) {
		body := ed25519Key + "\n" + rsaKey + "\n"
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(body))
		}))
		defer ts.Close()

		keys, err := FetchKeysFromURL(ts.URL, testClient(ts))
		if err != nil {
			t.Fatalf("FetchKeysFromURL() error: %v", err)
		}
		if len(keys) != 2 {
			t.Fatalf("expected 2 keys, got %d: %v", len(keys), keys)
		}
		if keys[0] != ed25519Key {
			t.Errorf("keys[0] = %q, want %q", truncateKey(keys[0]), truncateKey(ed25519Key))
		}
		if keys[1] != rsaKey {
			t.Errorf("keys[1] = %q, want %q", truncateKey(keys[1]), truncateKey(rsaKey))
		}
	})

	t.Run("returns error on 404", func(t *testing.T) {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer ts.Close()

		_, err := FetchKeysFromURL(ts.URL, testClient(ts))
		if err == nil {
			t.Fatal("expected error on 404, got nil")
		}
		if !strings.Contains(err.Error(), "no keys published") {
			t.Errorf("error should mention 'no keys published', got: %v", err)
		}
	})

	t.Run("returns error on empty body", func(t *testing.T) {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		_, err := FetchKeysFromURL(ts.URL, testClient(ts))
		if err == nil {
			t.Fatal("expected error on empty body, got nil")
		}
		if !strings.Contains(err.Error(), "no keys found") {
			t.Errorf("error should mention 'no keys found', got: %v", err)
		}
	})

	t.Run("skips blank lines", func(t *testing.T) {
		body := ed25519Key + "\n\n\n" + rsaKey + "\n\n"
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(body))
		}))
		defer ts.Close()

		keys, err := FetchKeysFromURL(ts.URL, testClient(ts))
		if err != nil {
			t.Fatalf("FetchKeysFromURL() error: %v", err)
		}
		if len(keys) != 2 {
			t.Fatalf("expected 2 keys (blank lines skipped), got %d", len(keys))
		}
	})

	t.Run("rejects http URL", func(t *testing.T) {
		_, err := FetchKeysFromURL("http://example.com/user.keys", nil)
		if err == nil {
			t.Fatal("expected error for http:// URL, got nil")
		}
		if !strings.Contains(err.Error(), "HTTPS") {
			t.Errorf("error should mention HTTPS, got: %v", err)
		}
	})
}

func TestKeyserverFetchIntegration(t *testing.T) {
	ed25519Key := testSSHEd25519PubKey(t)

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the URL path matches the keyserver pattern
		if r.URL.Path != "/alice.keys" {
			t.Errorf("unexpected path %q, want /alice.keys", r.URL.Path)
		}
		_, _ = w.Write([]byte(ed25519Key + "\n"))
	}))
	defer ts.Close()

	url := KeyserverKeysURL(ts.URL, "alice")
	keys, err := FetchKeysFromURL(url, testClient(ts))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
}

func TestFilterKeysByType(t *testing.T) {
	ed25519Key := testSSHEd25519PubKey(t)
	rsaKey := testSSHRSAPubKey(t)
	ecdsaKey := "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTY="

	keys := []string{ed25519Key, rsaKey, ecdsaKey, ed25519Key}

	ed25519Keys, rsaKeys, skippedKeys := FilterKeysByType(keys)

	if len(ed25519Keys) != 2 {
		t.Errorf("expected 2 ed25519 keys, got %d", len(ed25519Keys))
	}
	if len(rsaKeys) != 1 {
		t.Errorf("expected 1 rsa key, got %d", len(rsaKeys))
	}
	if len(skippedKeys) != 1 {
		t.Errorf("expected 1 skipped key, got %d", len(skippedKeys))
	}
	if len(skippedKeys) > 0 && skippedKeys[0] != ecdsaKey {
		t.Errorf("skipped key should be ecdsa, got %q", truncateKey(skippedKeys[0]))
	}
}

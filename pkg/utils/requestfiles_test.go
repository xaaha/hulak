package utils

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestRequestStem(t *testing.T) {
	cases := map[string]string{
		"login":          "login",
		"login.hk.yaml":  "login",
		"login.hk.yml":   "login",
		"LOGIN.yml":      "login",
		"getUser.yaml":   "getuser",
		"a/b/signup.yml": "signup",
	}
	for in, want := range cases {
		if got := RequestStem(in); got != want {
			t.Errorf("RequestStem(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIsRequestFile(t *testing.T) {
	yes := []string{"login.hk.yaml", "x.yaml", "x.yml"}
	no := []string{"options.yaml", "login_response.json", "notes.md", "data.json"}
	for _, b := range yes {
		if !IsRequestFile(b) {
			t.Errorf("IsRequestFile(%q) = false, want true", b)
		}
	}
	for _, b := range no {
		if IsRequestFile(b) {
			t.Errorf("IsRequestFile(%q) = true, want false", b)
		}
	}
}

func TestFindRequestFiles(t *testing.T) {
	root := t.TempDir()
	write := func(rel string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("kind: API\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("login.hk.yaml")
	write("sub/getUser.yml")
	write("options.yaml")        // ignored
	write("login_response.json") // ignored

	t.Run("matches hk.yaml by bare name", func(t *testing.T) {
		got, err := FindRequestFiles(root, "login")
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Contains(got, filepath.Join(root, "login.hk.yaml")) || len(got) != 1 {
			t.Errorf("got %v", got)
		}
	})

	t.Run("matches nested by name", func(t *testing.T) {
		got, err := FindRequestFiles(root, "getUser")
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 {
			t.Errorf("got %v", got)
		}
	})

	t.Run("no match returns empty, not error", func(t *testing.T) {
		got, err := FindRequestFiles(root, "missing")
		if err != nil {
			t.Fatalf("should not error on no match: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("got %v, want empty", got)
		}
	})
}

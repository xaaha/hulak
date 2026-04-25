package envparser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"filippo.io/age"

	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/vault"
)

func TestTrimQuotes(t *testing.T) {
	testCases := []struct {
		output     any
		input      string
		wasTrimmed bool
	}{
		{input: "", output: "", wasTrimmed: false},
		{input: `"test's value"`, output: "test's value", wasTrimmed: true},
		{input: `"userNam2"`, output: "userNam2", wasTrimmed: true},
		{input: `22`, output: `22`, wasTrimmed: false},
		{input: `"false"`, output: `false`, wasTrimmed: true},
		{input: `199.289`, output: `199.289`, wasTrimmed: false},
		{input: `"199.289"`, output: `199.289`, wasTrimmed: true},
	}

	for _, tc := range testCases {
		resultStr, wasTrimmed := trimQuotes(tc.input)
		if resultStr != tc.output {
			t.Errorf(
				"Expected output does not match the result: \n%v \nvs \n%v",
				tc.output,
				resultStr,
			)
		}
		if tc.wasTrimmed != wasTrimmed {
			t.Errorf("Expected wasTrimmed to be %t but got %t", tc.wasTrimmed, wasTrimmed)
		}
	}
}

// create a temporary file for tetsing
func createTempEnvFile(content string) (string, error) {
	file, err := os.CreateTemp("", "*.env")
	if err != nil {
		return "", err
	}
	defer file.Close()
	if _, err := file.WriteString(content); err != nil {
		return "", err
	}
	return file.Name(), nil
}

func TestLoadEnvVars(t *testing.T) {
	content := `
# This is a comment
KEY1=value1
KEY2="value2"
KEY3='value3'
`

	filePath, err := createTempEnvFile(content)
	if err != nil {
		t.Fatalf("Failed to create temp env file: %v", err)
	}
	defer os.Remove(filePath)

	expected := map[string]string{
		"KEY1": "value1",
		"KEY2": "value2",
		"KEY3": "value3",
	}

	result, err := LoadEnvVars(filePath)
	if err != nil {
		t.Fatalf("LoadEnvVars returned error: %v", err)
	}

	if len(result) != len(expected) {
		t.Fatalf("Expected map length %d, got %d", len(expected), len(result))
	}

	for key, val := range expected {
		if result[key] != val {
			t.Errorf("Expected key %s to have value %s, got %s", key, val, result[key])
		}
	}
}

func setupVaultProject(t *testing.T, store *vault.Store) {
	t.Helper()

	tmpDir := t.TempDir()
	tmpDir, _ = filepath.EvalSymlinks(tmpDir)

	hulakDir := filepath.Join(tmpDir, utils.HiddenProjectName)
	if err := os.Mkdir(hulakDir, utils.DirPer); err != nil {
		t.Fatal(err)
	}

	configTmp := t.TempDir()
	configTmp, _ = filepath.EvalSymlinks(configTmp)
	t.Setenv("XDG_CONFIG_HOME", configTmp)

	configDir, err := utils.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(configDir, utils.DirPer); err != nil {
		t.Fatal(err)
	}

	oldWd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Fatal(err)
		}
	})

	id, _ := age.GenerateX25519Identity()
	if err := vault.SetIdentity(id.String()); err != nil {
		t.Fatalf("SetIdentity error: %v", err)
	}
	if err := vault.WriteStore(store, id.Recipient()); err != nil {
		t.Fatalf("WriteStore error: %v", err)
	}
}

func TestLoadSecretsFromVault(t *testing.T) {
	t.Run("merges global and custom env", func(t *testing.T) {
		setupVaultProject(t, &vault.Store{Envs: map[string]vault.Env{
			"global": {"URL": "https://example.com", "DEBUG": true},
			"prod":   {"URL": "https://prod.example.com", "API_KEY": "sk-xxx"},
		}})

		got, err := loadSecretsFromVault("prod")
		if err != nil {
			t.Fatalf("loadSecretsFromVault error: %v", err)
		}

		// Custom overrides global
		if got["URL"] != "https://prod.example.com" {
			t.Errorf("URL = %v, want prod override", got["URL"])
		}
		// Global preserved when not overridden
		if got["DEBUG"] != true {
			t.Errorf("DEBUG = %v, want true (from global)", got["DEBUG"])
		}
		// Custom-only key present
		if got["API_KEY"] != "sk-xxx" {
			t.Errorf("API_KEY = %v, want sk-xxx", got["API_KEY"])
		}
	})

	t.Run("returns global only when envName is global", func(t *testing.T) {
		setupVaultProject(t, &vault.Store{Envs: map[string]vault.Env{
			"global": {"URL": "https://example.com"},
		}})

		got, err := loadSecretsFromVault(utils.DefaultEnvVal)
		if err != nil {
			t.Fatalf("loadSecretsFromVault error: %v", err)
		}
		if got["URL"] != "https://example.com" {
			t.Errorf("URL = %v, want global value", got["URL"])
		}
	})

	t.Run("errors when non-global env is missing", func(t *testing.T) {
		setupVaultProject(t, &vault.Store{Envs: map[string]vault.Env{
			"global": {"URL": "https://example.com"},
		}})

		_, err := loadSecretsFromVault("staging")
		if err == nil {
			t.Fatal("expected error for missing env, got nil")
		}
		if !strings.Contains(err.Error(), "staging") {
			t.Errorf("error = %q, want it to mention 'staging'", err.Error())
		}
	})
}

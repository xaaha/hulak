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

func TestLoadEnvVarsRaw(t *testing.T) {
	t.Setenv("HULAK_TEST_RAW_TOKEN", "resolved_value")

	content := `
# comment line
TOKEN=$HULAK_TEST_RAW_TOKEN
MISSING=$HULAK_TEST_DOES_NOT_EXIST
PLAIN=hello
PORT=8080
RATE=3.14
DEBUG=true
QUOTED="some string"
`
	filePath, err := createTempEnvFile(content)
	if err != nil {
		t.Fatalf("Failed to create temp env file: %v", err)
	}
	defer os.Remove(filePath)

	raw, err := LoadEnvVarsRaw(filePath)
	if err != nil {
		t.Fatalf("LoadEnvVarsRaw error: %v", err)
	}

	t.Run("preserves dollar var as literal", func(t *testing.T) {
		if raw["TOKEN"] != "$HULAK_TEST_RAW_TOKEN" {
			t.Errorf("TOKEN = %v, want literal $HULAK_TEST_RAW_TOKEN", raw["TOKEN"])
		}
	})

	t.Run("preserves missing dollar var as literal", func(t *testing.T) {
		if raw["MISSING"] != "$HULAK_TEST_DOES_NOT_EXIST" {
			t.Errorf("MISSING = %v, want literal $HULAK_TEST_DOES_NOT_EXIST", raw["MISSING"])
		}
	})

	t.Run("plain string unchanged", func(t *testing.T) {
		if raw["PLAIN"] != "hello" {
			t.Errorf("PLAIN = %v, want hello", raw["PLAIN"])
		}
	})

	t.Run("type inference still works", func(t *testing.T) {
		if raw["PORT"] != 8080 {
			t.Errorf("PORT = %v (%T), want int 8080", raw["PORT"], raw["PORT"])
		}
		if raw["RATE"] != 3.14 {
			t.Errorf("RATE = %v (%T), want float 3.14", raw["RATE"], raw["RATE"])
		}
		if raw["DEBUG"] != true {
			t.Errorf("DEBUG = %v (%T), want bool true", raw["DEBUG"], raw["DEBUG"])
		}
	})

	t.Run("quoted string stays string", func(t *testing.T) {
		if raw["QUOTED"] != "some string" {
			t.Errorf("QUOTED = %v, want 'some string'", raw["QUOTED"])
		}
	})

	// Regression: LoadEnvVars still resolves $VAR
	t.Run("LoadEnvVars still resolves dollar var", func(t *testing.T) {
		resolved, err := LoadEnvVars(filePath)
		if err != nil {
			t.Fatalf("LoadEnvVars error: %v", err)
		}
		if resolved["TOKEN"] != "resolved_value" {
			t.Errorf("TOKEN = %v, want resolved_value", resolved["TOKEN"])
		}
		// $MISSING resolves to empty string (var not set)
		if resolved["MISSING"] != "" {
			t.Errorf("MISSING = %v, want empty string", resolved["MISSING"])
		}
	})
}

// createTempEnvFileBytes writes raw bytes to a temp .env file.
// Use this instead of createTempEnvFile when you need to write BOM bytes.
func createTempEnvFileBytes(data []byte) (string, error) {
	file, err := os.CreateTemp("", "*.env")
	if err != nil {
		return "", err
	}
	defer file.Close()
	if _, err := file.Write(data); err != nil {
		return "", err
	}
	return file.Name(), nil
}

func TestBOMHandling(t *testing.T) {
	t.Run("strips UTF-8 BOM", func(t *testing.T) {
		content := append([]byte{0xEF, 0xBB, 0xBF}, []byte("KEY1=value1\nKEY2=value2\n")...)

		path, err := createTempEnvFileBytes(content)
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(path)

		result, err := LoadEnvVars(path)
		if err != nil {
			t.Fatalf("LoadEnvVars error: %v", err)
		}
		if result["KEY1"] != "value1" {
			t.Errorf("KEY1 = %v, want value1", result["KEY1"])
		}
		if result["KEY2"] != "value2" {
			t.Errorf("KEY2 = %v, want value2", result["KEY2"])
		}
	})

	t.Run("rejects UTF-16 BE BOM", func(t *testing.T) {
		content := append([]byte{0xFE, 0xFF}, []byte("KEY=val\n")...)

		path, err := createTempEnvFileBytes(content)
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(path)

		_, err = LoadEnvVars(path)
		if err == nil {
			t.Fatal("expected error for UTF-16 BE BOM")
		}
		if !strings.Contains(err.Error(), "UTF-16 BE") {
			t.Errorf("error = %v, want mention of UTF-16 BE", err)
		}
	})

	t.Run("rejects UTF-16 LE BOM", func(t *testing.T) {
		content := append([]byte{0xFF, 0xFE}, []byte("KEY=val\n")...)

		path, err := createTempEnvFileBytes(content)
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(path)

		_, err = LoadEnvVars(path)
		if err == nil {
			t.Fatal("expected error for UTF-16 LE BOM")
		}
		if !strings.Contains(err.Error(), "UTF-16 LE") {
			t.Errorf("error = %v, want mention of UTF-16 LE", err)
		}
	})

	t.Run("rejects UTF-32 BE BOM", func(t *testing.T) {
		content := append([]byte{0x00, 0x00, 0xFE, 0xFF}, []byte("KEY=val\n")...)

		path, err := createTempEnvFileBytes(content)
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(path)

		_, err = LoadEnvVars(path)
		if err == nil {
			t.Fatal("expected error for UTF-32 BE BOM")
		}
		if !strings.Contains(err.Error(), "UTF-32 BE") {
			t.Errorf("error = %v, want mention of UTF-32 BE", err)
		}
	})

	t.Run("rejects UTF-32 LE BOM over UTF-16 LE", func(t *testing.T) {
		// UTF-32 LE starts with same 2 bytes as UTF-16 LE (\xFF\xFE).
		// Must be detected as UTF-32 LE (4-byte match wins).
		content := append([]byte{0xFF, 0xFE, 0x00, 0x00}, []byte("KEY=val\n")...)

		path, err := createTempEnvFileBytes(content)
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(path)

		_, err = LoadEnvVars(path)
		if err == nil {
			t.Fatal("expected error for UTF-32 LE BOM")
		}
		if !strings.Contains(err.Error(), "UTF-32 LE") {
			t.Errorf("error = %v, want mention of UTF-32 LE (not UTF-16 LE)", err)
		}
	})

	t.Run("no BOM works unchanged", func(t *testing.T) {
		path, err := createTempEnvFile("KEY=value\n")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(path)

		result, err := LoadEnvVars(path)
		if err != nil {
			t.Fatalf("LoadEnvVars error: %v", err)
		}
		if result["KEY"] != "value" {
			t.Errorf("KEY = %v, want value", result["KEY"])
		}
	})
}

func TestNonASCIIValues(t *testing.T) {
	t.Run("preserves UTF-8 non-ASCII in values", func(t *testing.T) {
		content := "NAME=José\nGREET=こんにちは\nEMOJI=🚀\n"
		path, err := createTempEnvFile(content)
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(path)

		result, err := LoadEnvVars(path)
		if err != nil {
			t.Fatalf("LoadEnvVars error: %v", err)
		}
		if result["NAME"] != "José" {
			t.Errorf("NAME = %v, want José", result["NAME"])
		}
		if result["GREET"] != "こんにちは" {
			t.Errorf("GREET = %v, want こんにちは", result["GREET"])
		}
		if result["EMOJI"] != "🚀" {
			t.Errorf("EMOJI = %v, want 🚀", result["EMOJI"])
		}
	})

	t.Run("CRLF line endings parse correctly", func(t *testing.T) {
		content := []byte("KEY1=val1\r\nKEY2=val2\r\n")
		path, err := createTempEnvFileBytes(content)
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(path)

		result, err := LoadEnvVars(path)
		if err != nil {
			t.Fatalf("LoadEnvVars error: %v", err)
		}
		if result["KEY1"] != "val1" {
			t.Errorf("KEY1 = %q, want val1", result["KEY1"])
		}
		if result["KEY2"] != "val2" {
			t.Errorf("KEY2 = %q, want val2", result["KEY2"])
		}
	})

	t.Run("UTF-8 BOM with non-ASCII values", func(t *testing.T) {
		content := append([]byte{0xEF, 0xBB, 0xBF}, []byte("NAME=José\n")...)
		path, err := createTempEnvFileBytes(content)
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(path)

		result, err := LoadEnvVars(path)
		if err != nil {
			t.Fatalf("LoadEnvVars error: %v", err)
		}
		if result["NAME"] != "José" {
			t.Errorf("NAME = %v, want José", result["NAME"])
		}
	})
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

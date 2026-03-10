package graphql

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xaaha/hulak/pkg/yamlparser"
)

func stubFileInfo(isDir bool) os.FileInfo {
	return fakeFileInfo{isDir: isDir}
}

type fakeFileInfo struct {
	isDir bool
}

func (f fakeFileInfo) Name() string       { return "stub" }
func (f fakeFileInfo) Size() int64        { return 0 }
func (f fakeFileInfo) Mode() os.FileMode  { return 0 }
func (f fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (f fakeFileInfo) IsDir() bool        { return f.isDir }
func (f fakeFileInfo) Sys() any           { return nil }

func TestLoadSchemasFromDirectory(t *testing.T) {
	loader := schemaLoader{
		stat: func(path string) (os.FileInfo, error) {
			if path != "/tmp/apis" {
				t.Fatalf("unexpected path: %s", path)
			}
			return stubFileInfo(true), nil
		},
		getwd: func() (string, error) { return "", nil },
		findGraphQLFiles: func(_ string) (map[string]string, bool, error) {
			return map[string]string{
				"https://b.test/graphql": "/tmp/apis/b.yaml",
				"https://a.test/graphql": "/tmp/apis/a.yaml",
			}, true, nil
		},
		validateGraphQLFile: func(string) (string, bool, error) {
			t.Fatal("validateGraphQLFile should not be called for directories")
			return "", false, nil
		},
		getSecretsForEnv: func(urlToFileMap map[string]string, needsEnv bool, env string) map[string]any {
			if !needsEnv {
				t.Fatal("expected needsEnv=true")
			}
			if env != "dev" {
				t.Fatalf("unexpected env: %s", env)
			}
			if len(urlToFileMap) != 2 {
				t.Fatalf("expected 2 urls, got %d", len(urlToFileMap))
			}
			return map[string]any{"token": "x"}
		},
		processFilesConcurrent: func(filePaths []string, _ map[string]any) []ProcessResult {
			want := []string{"/tmp/apis/a.yaml", "/tmp/apis/b.yaml"}
			if !reflect.DeepEqual(filePaths, want) {
				t.Fatalf("unexpected file paths: got %v want %v", filePaths, want)
			}
			return []ProcessResult{
				{APIInfo: yamlparser.APIInfo{URL: "https://b.test/graphql"}},
				{APIInfo: yamlparser.APIInfo{URL: "https://a.test/graphql"}},
			}
		},
		fetchAndParseSchema: func(apiInfo yamlparser.APIInfo) (Schema, error) {
			return Schema{
				Queries: []Operation{{Name: apiInfo.URL}},
			}, nil
		},
		getWorkers: func(*int) int { return 1 },
	}

	prepared, err := loader.Prepare("/tmp/apis", "dev")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prepared.Cancelled {
		t.Fatal("expected non-cancelled result")
	}
	result, err := loader.Fetch(prepared)
	if err != nil {
		t.Fatalf("unexpected fetch error: %v", err)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", result.Warnings)
	}
	if len(result.Endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(result.Endpoints))
	}
	if result.Endpoints[0].URL != "https://a.test/graphql" {
		t.Fatalf("expected sorted endpoints, got first=%s", result.Endpoints[0].URL)
	}
	if got := result.Endpoints[1].Schema.Queries[0].Name; got != "https://b.test/graphql" {
		t.Fatalf("unexpected schema payload: %s", got)
	}
}

func TestLoadSchemasFromFile(t *testing.T) {
	loader := schemaLoader{
		stat:  func(string) (os.FileInfo, error) { return stubFileInfo(false), nil },
		getwd: func() (string, error) { return "", nil },
		findGraphQLFiles: func(string) (map[string]string, bool, error) {
			t.Fatal("findGraphQLFiles should not be called for files")
			return nil, false, nil
		},
		validateGraphQLFile: func(path string) (string, bool, error) {
			if path != "/tmp/query.yaml" {
				t.Fatalf("unexpected file path: %s", path)
			}
			return "https://api.test/graphql", false, nil
		},
		getSecretsForEnv: func(_ map[string]string, needsEnv bool, env string) map[string]any {
			if needsEnv {
				t.Fatal("expected needsEnv=false")
			}
			if env != "prod" {
				t.Fatalf("unexpected env: %s", env)
			}
			return map[string]any{}
		},
		processFilesConcurrent: func(filePaths []string, _ map[string]any) []ProcessResult {
			if !reflect.DeepEqual(filePaths, []string{"/tmp/query.yaml"}) {
				t.Fatalf("unexpected file paths: %v", filePaths)
			}
			return []ProcessResult{
				{APIInfo: yamlparser.APIInfo{URL: "https://api.test/graphql"}},
			}
		},
		fetchAndParseSchema: func(_ yamlparser.APIInfo) (Schema, error) {
			return Schema{Mutations: []Operation{{Name: "mut"}}}, nil
		},
		getWorkers: func(*int) int { return 1 },
	}

	prepared, err := loader.Prepare("/tmp/query.yaml", "prod")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result, err := loader.Fetch(prepared)
	if err != nil {
		t.Fatalf("unexpected fetch error: %v", err)
	}
	if len(result.Endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(result.Endpoints))
	}
	if result.Endpoints[0].URL != "https://api.test/graphql" {
		t.Fatalf("unexpected endpoint url: %s", result.Endpoints[0].URL)
	}
}

func TestLoadSchemasReturnsWarningsForPartialFailures(t *testing.T) {
	loader := schemaLoader{
		stat:  func(string) (os.FileInfo, error) { return stubFileInfo(true), nil },
		getwd: func() (string, error) { return "", nil },
		findGraphQLFiles: func(string) (map[string]string, bool, error) {
			return map[string]string{"https://good.test/graphql": "/tmp/good.yaml"}, false, nil
		},
		validateGraphQLFile: func(string) (string, bool, error) { return "", false, nil },
		getSecretsForEnv:    func(map[string]string, bool, string) map[string]any { return map[string]any{} },
		processFilesConcurrent: func([]string, map[string]any) []ProcessResult {
			return []ProcessResult{
				{
					APIInfo: yamlparser.APIInfo{URL: "https://bad-file.test/graphql"},
					Error:   errors.New("invalid yaml"),
				},
				{
					APIInfo: yamlparser.APIInfo{URL: "https://good.test/graphql"},
				},
				{
					APIInfo: yamlparser.APIInfo{URL: "https://bad-fetch.test/graphql"},
				},
			}
		},
		fetchAndParseSchema: func(apiInfo yamlparser.APIInfo) (Schema, error) {
			if strings.Contains(apiInfo.URL, "bad-fetch") {
				return Schema{}, errors.New("forbidden")
			}
			return Schema{Queries: []Operation{{Name: "ok"}}}, nil
		},
		getWorkers: func(*int) int { return 1 },
	}

	prepared, err := loader.Prepare("/tmp/apis", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result, err := loader.Fetch(prepared)
	if err != nil {
		t.Fatalf("unexpected fetch error: %v", err)
	}
	if len(result.Endpoints) != 1 {
		t.Fatalf("expected 1 successful endpoint, got %d", len(result.Endpoints))
	}
	if len(result.Warnings) != 2 {
		t.Fatalf("expected 2 warnings, got %d (%v)", len(result.Warnings), result.Warnings)
	}
	if !strings.Contains(result.Warnings[0], "bad-") || !strings.Contains(result.Warnings[1], "bad-") {
		t.Fatalf("expected warnings to mention failing endpoints, got %v", result.Warnings)
	}
}

func TestLoadSchemasReturnsErrorWhenAllEndpointsFail(t *testing.T) {
	loader := schemaLoader{
		stat:             func(string) (os.FileInfo, error) { return stubFileInfo(false), nil },
		getwd:            func() (string, error) { return "", nil },
		findGraphQLFiles: func(string) (map[string]string, bool, error) { return nil, false, nil },
		validateGraphQLFile: func(string) (string, bool, error) {
			return "https://api.test/graphql", false, nil
		},
		getSecretsForEnv: func(map[string]string, bool, string) map[string]any { return map[string]any{} },
		processFilesConcurrent: func([]string, map[string]any) []ProcessResult {
			return []ProcessResult{
				{
					APIInfo: yamlparser.APIInfo{URL: "https://api.test/graphql"},
					Error:   errors.New("template failure"),
				},
			}
		},
		fetchAndParseSchema: func(apiInfo yamlparser.APIInfo) (Schema, error) {
			t.Fatalf("fetch should not be called when processing already failed for %s", apiInfo.URL)
			return Schema{}, nil
		},
		getWorkers: func(*int) int { return 1 },
	}

	prepared, err := loader.Prepare("/tmp/query.yaml", "")
	if err != nil {
		t.Fatalf("unexpected prepare error: %v", err)
	}
	_, err = loader.Fetch(prepared)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "all schema fetches failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadSchemasReturnsCancelledWhenEnvSelectionCancelled(t *testing.T) {
	loader := schemaLoader{
		stat:  func(string) (os.FileInfo, error) { return stubFileInfo(true), nil },
		getwd: func() (string, error) { return "", nil },
		findGraphQLFiles: func(string) (map[string]string, bool, error) {
			return map[string]string{"https://api.test/graphql": "/tmp/query.yaml"}, true, nil
		},
		validateGraphQLFile: func(string) (string, bool, error) { return "", false, nil },
		getSecretsForEnv:    func(map[string]string, bool, string) map[string]any { return nil },
		processFilesConcurrent: func([]string, map[string]any) []ProcessResult {
			t.Fatal("processFilesConcurrent should not be called when env selection is cancelled")
			return nil
		},
		fetchAndParseSchema: func(yamlparser.APIInfo) (Schema, error) {
			t.Fatal("fetchAndParseSchema should not be called when env selection is cancelled")
			return Schema{}, nil
		},
		getWorkers: func(*int) int { return 1 },
	}

	result, err := loader.Prepare("/tmp/apis", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Cancelled {
		t.Fatal("expected cancelled result")
	}
	if len(result.Results) != 0 {
		t.Fatalf("expected no results, got %d", len(result.Results))
	}
}

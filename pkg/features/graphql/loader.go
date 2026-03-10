package graphql

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/yamlparser"
)

// LoadedEndpoint contains one successfully fetched schema keyed by endpoint URL.
type LoadedEndpoint struct {
	URL    string
	Schema Schema
}

// LoadResult is the reusable output for GraphQL schema loading.
// Warnings contains non-fatal processing or fetch errors for individual endpoints.
type LoadResult struct {
	Endpoints []LoadedEndpoint
	Warnings  []string
	Cancelled bool
}

// PreparedLoad is the non-interactive fetch input produced after resolving
// file or directory targets and any environment-backed templates.
type PreparedLoad struct {
	Results   []ProcessResult
	Cancelled bool
	Env       string
}

type schemaLoader struct {
	stat                   func(string) (os.FileInfo, error)
	getwd                  func() (string, error)
	findGraphQLFiles       func(string) (map[string]string, bool, error)
	validateGraphQLFile    func(string) (string, bool, error)
	resolveSecretsForEnv   func(map[string]string, bool, string) (map[string]any, string, bool)
	processFilesConcurrent func([]string, map[string]any) []ProcessResult
	fetchAndParseSchema    func(yamlparser.APIInfo) (Schema, error)
	getWorkers             func(*int) int
}

func newSchemaLoader() schemaLoader {
	return schemaLoader{
		stat:                   os.Stat,
		getwd:                  os.Getwd,
		findGraphQLFiles:       FindGraphQLFiles,
		validateGraphQLFile:    ValidateGraphQLFile,
		resolveSecretsForEnv:   ResolveSecretsForEnv,
		processFilesConcurrent: ProcessFilesConcurrent,
		fetchAndParseSchema:    FetchAndParseSchema,
		getWorkers:             utils.GetWorkers,
	}
}

// LoadSchemas resolves a GraphQL file or directory target, processes matching
// files, fetches introspection schemas, and returns successful endpoints plus
// non-fatal warnings. If all endpoints fail, an error is returned.
func LoadSchemas(path, env string) (LoadResult, error) {
	loader := newSchemaLoader()
	prepared, err := loader.Prepare(path, env)
	if err != nil {
		return LoadResult{}, err
	}
	if prepared.Cancelled {
		return LoadResult{Cancelled: true}, nil
	}
	return loader.Fetch(prepared)
}

// Prepare resolves a GraphQL input path into processed request definitions.
// This may invoke interactive environment selection when needed.
func PrepareSchemaLoad(path, env string) (PreparedLoad, error) {
	return newSchemaLoader().Prepare(path, env)
}

// FetchPreparedSchemas fetches and parses schemas for previously prepared inputs.
func FetchPreparedSchemas(prepared PreparedLoad) (LoadResult, error) {
	if prepared.Cancelled {
		return LoadResult{Cancelled: true}, nil
	}
	return newSchemaLoader().Fetch(prepared)
}

func (l schemaLoader) Prepare(path, env string) (PreparedLoad, error) {
	resolved, err := l.resolvePath(path)
	if err != nil {
		return PreparedLoad{}, err
	}

	results, selectedEnv, cancelled, err := l.loadProcessResults(resolved, env)
	if err != nil {
		return PreparedLoad{}, err
	}
	if cancelled {
		return PreparedLoad{Cancelled: true}, nil
	}

	return PreparedLoad{Results: results, Env: selectedEnv}, nil
}

func (l schemaLoader) Fetch(prepared PreparedLoad) (LoadResult, error) {
	if prepared.Cancelled {
		return LoadResult{Cancelled: true}, nil
	}
	return l.fetchSchemas(prepared.Results)
}

func (l schemaLoader) resolvePath(path string) (string, error) {
	if path == "." {
		cwd, err := l.getwd()
		if err != nil {
			return "", fmt.Errorf("error getting current directory: %w", err)
		}
		return cwd, nil
	}
	return filepath.Clean(path), nil
}

func (l schemaLoader) loadProcessResults(path, env string) ([]ProcessResult, string, bool, error) {
	info, err := l.stat(path)
	if err != nil {
		return nil, "", false, fmt.Errorf("cannot access %q: %w", path, err)
	}

	if info.IsDir() {
		return l.loadFromDirectory(path, env)
	}
	return l.loadFromFile(path, env)
}

func (l schemaLoader) loadFromDirectory(dir, env string) ([]ProcessResult, string, bool, error) {
	urlToFileMap, needsEnv, err := l.findGraphQLFiles(dir)
	if err != nil {
		return nil, "", false, err
	}

	filePaths := make([]string, 0, len(urlToFileMap))
	for _, fp := range urlToFileMap {
		filePaths = append(filePaths, fp)
	}
	sort.Strings(filePaths)

	secretsMap, selectedEnv, cancelled := l.resolveSecretsForEnv(urlToFileMap, needsEnv, env)
	if cancelled {
		return nil, "", true, nil
	}

	return l.processFilesConcurrent(filePaths, secretsMap), selectedEnv, false, nil
}

func (l schemaLoader) loadFromFile(filePath, env string) ([]ProcessResult, string, bool, error) {
	rawURL, needsEnv, err := l.validateGraphQLFile(filePath)
	if err != nil {
		return nil, "", false, err
	}

	urlToFileMap := map[string]string{rawURL: filePath}
	secretsMap, selectedEnv, cancelled := l.resolveSecretsForEnv(urlToFileMap, needsEnv, env)
	if cancelled {
		return nil, "", true, nil
	}

	return l.processFilesConcurrent([]string{filePath}, secretsMap), selectedEnv, false, nil
}

func (l schemaLoader) fetchSchemas(results []ProcessResult) (LoadResult, error) {
	type fetchResult struct {
		url    string
		schema Schema
		err    error
	}

	var warnings []string
	uniqueFetchResults := make(map[string]ProcessResult)
	for _, result := range results {
		if result.Error != nil {
			warnings = append(warnings, l.formatProcessWarning(&result))
			continue
		}
		url := strings.TrimSpace(result.APIInfo.URL)
		if url == "" {
			warnings = append(warnings, l.formatProcessWarning(&ProcessResult{
				FilePath: result.FilePath,
				APIInfo:  result.APIInfo,
				Error:    fmt.Errorf("missing resolved URL for schema fetch"),
			}))
			continue
		}
		if _, exists := uniqueFetchResults[url]; !exists {
			uniqueFetchResults[url] = result
		}
	}

	if len(uniqueFetchResults) == 0 {
		if len(warnings) == 0 {
			return LoadResult{}, nil
		}
		return LoadResult{}, fmt.Errorf("all schema fetches failed:\n  %s", strings.Join(warnings, "\n  "))
	}

	endpointResults := make([]ProcessResult, 0, len(uniqueFetchResults))
	for _, result := range uniqueFetchResults {
		endpointResults = append(endpointResults, result)
	}
	sort.Slice(endpointResults, func(i, j int) bool {
		if endpointResults[i].APIInfo.URL == endpointResults[j].APIInfo.URL {
			return endpointResults[i].FilePath < endpointResults[j].FilePath
		}
		return endpointResults[i].APIInfo.URL < endpointResults[j].APIInfo.URL
	})

	jobs := make(chan ProcessResult, len(endpointResults))
	fetched := make(chan fetchResult, len(endpointResults))
	endpointResultsLen := len(endpointResults)
	workerCount := l.getWorkers(&endpointResultsLen)

	var wg sync.WaitGroup
	for range workerCount {
		wg.Go(func() {
			for result := range jobs {
				schema, err := l.fetchAndParseSchema(result.APIInfo)
				fetched <- fetchResult{
					url:    result.APIInfo.URL,
					schema: schema,
					err:    err,
				}
			}
		})
	}

	for _, result := range endpointResults {
		jobs <- result
	}
	close(jobs)

	wg.Wait()
	close(fetched)

	var loaded []LoadedEndpoint
	for result := range fetched {
		if result.err != nil {
			warnings = append(warnings, fmt.Sprintf("%s: %v", result.url, result.err))
			continue
		}
		loaded = append(loaded, LoadedEndpoint{
			URL:    result.url,
			Schema: result.schema,
		})
	}

	sort.Slice(loaded, func(i, j int) bool {
		return loaded[i].URL < loaded[j].URL
	})
	sort.Strings(warnings)

	if len(loaded) == 0 && len(warnings) > 0 {
		return LoadResult{}, fmt.Errorf("all schema fetches failed:\n  %s", strings.Join(warnings, "\n  "))
	}

	return LoadResult{
		Endpoints: loaded,
		Warnings:  warnings,
	}, nil
}

func (l schemaLoader) formatProcessWarning(result *ProcessResult) string {
	label := l.relativeLabel(result.FilePath)
	if label == "" {
		label = strings.TrimSpace(result.APIInfo.URL)
	}
	if label == "" {
		label = "(unknown source)"
	}
	return fmt.Sprintf("%s: %v", label, result.Error)
}

func (l schemaLoader) relativeLabel(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if !filepath.IsAbs(path) {
		return path
	}
	cwd, err := l.getwd()
	if err != nil || cwd == "" {
		return path
	}
	rel, err := filepath.Rel(cwd, path)
	if err != nil || rel == "" {
		return path
	}
	return rel
}

// Package runner contains the API request execution pipeline.
// It is imported by both the run subcommand and main's interactive mode.
package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	apicalls "github.com/xaaha/hulak/pkg/apiCalls"
	"github.com/xaaha/hulak/pkg/envparser"
	"github.com/xaaha/hulak/pkg/features"
	"github.com/xaaha/hulak/pkg/tui/envselect"
	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/vault"
	"github.com/xaaha/hulak/pkg/yamlparser"
)

// envSelector is the function used to show the interactive environment picker.
// Package-level var so tests can replace it without TUI dependencies.
var envSelector = envselect.RunEnvSelector

// Flags holds parsed CLI flags needed by the execution pipeline.
type Flags struct {
	Env      string
	EnvSet   bool
	FilePath string
	File     string
	Debug    bool
	Dir      string
	Dirseq   string
}

// Execute runs the full pipeline: discover files, resolve env, execute requests.
func Execute(f *Flags) {
	fileList, concurrentDir, sequentialDir := discoverFilePaths(
		f.File,
		f.FilePath,
		f.Dir,
		f.Dirseq,
		f.Dir != "" || f.Dirseq != "",
	)

	allPaths := slices.Concat(fileList, concurrentDir, sequentialDir)

	var envMap map[string]any
	if containsTemplateVars(allPaths) {
		if !utils.IsHulakProject() {
			utils.PanicRedAndExit("fatal: not a hulak project \n\nRun 'hulak init' to set up")
		}
		if !f.EnvSet {
			selectedEnv, err := envSelector()
			if err != nil {
				utils.PanicRedAndExit("Environment selector error: %v", err)
			}
			if selectedEnv == "" {
				os.Exit(0)
			}
			f.Env = selectedEnv
		}
		envMap = InitializeProject(f.Env, true)
	}

	handleAPIRequests(
		envMap,
		f.Debug,
		append(fileList, concurrentDir...),
		sequentialDir,
		f.FilePath,
	)
}

// ExecuteSingleFile runs a single file through the pipeline.
// Used by interactive mode where the file is already known.
func ExecuteSingleFile(envMap map[string]any, debug bool, filePath string) {
	handleAPIRequests(envMap, debug, []string{filePath}, nil, filePath)
}

// containsTemplateVars returns true if any file in the list uses template vars.
func containsTemplateVars(paths []string) bool {
	return slices.ContainsFunc(paths, utils.FileHasTemplateVars)
}

// InitializeProject creates the env setup and returns the secrets map.
// In vault mode (.hulak/store.age), skips creating the legacy env/ folder.
func InitializeProject(env string, isCli bool) map[string]any {
	if vault.DetectStore() != vault.StoreAge {
		if err := envparser.CreateDefaultEnvs(nil); err != nil {
			utils.PanicRedAndExit("%v", err)
		}
	}
	envMap, err := envparser.GenerateSecretsMap(env, isCli)
	if err != nil {
		panic(err)
	}
	return envMap
}

// discoverFilePaths collects all file paths from -f, -fp, -dir, and -dirseq flags.
func discoverFilePaths(
	fileName, fp, dir, dirseq string,
	hasDirFlags bool,
) (fileList, concurrentDir, sequentialDir []string) {
	if fp != "" || fileName != "" {
		var err error
		fileList, err = generateFilePathList(fileName, fp)
		if err != nil {
			if !hasDirFlags {
				utils.PanicRedAndExit("%v", err)
			}
			utils.PrintWarningStderr(fmt.Sprintf("file flags: %v", err))
		}
	}

	if hasDirFlags {
		dirPaths, err := apicalls.ListDirPaths(dir, dirseq)
		if err != nil {
			utils.PrintErrorStderr(fmt.Sprintf("processing directories: %v", err))
		} else {
			concurrentDir = dirPaths.Concurrent
			sequentialDir = dirPaths.Sequential
		}
	}

	return fileList, concurrentDir, sequentialDir
}

// outcome captures the result of executing one request file. Used to render
// the per-file outcome line and the end-of-run summary.
type outcome struct {
	path     string        // request file path
	ok       bool          // true if request returned without error (any status code)
	status   string        // HTTP status string, e.g. "200 OK"; empty for non-API kinds and pre-flight errors
	duration time.Duration // wall-clock time for the request itself
	err      error         // non-nil on failure
}

// handleAPIRequests processes API requests from pre-discovered file lists.
func handleAPIRequests(
	secrets map[string]any,
	debug bool,
	concurrentFiles []string,
	sequentialFiles []string,
	fp string,
) {
	totalFiles := len(concurrentFiles) + len(sequentialFiles)
	// Per-file outcome lines only help when multiple files are running — they
	// let the user track which file finished. With a single file there's
	// exactly one outcome and the line is just noise; suppress it.
	multiFile := totalFiles > 1

	overallStart := time.Now()
	var outcomes []outcome

	if len(concurrentFiles) > 0 {
		if len(concurrentFiles) > 1 || len(sequentialFiles) > 0 {
			utils.PrintInfoStderr(
				fmt.Sprintf("Processing %d files concurrently...", len(concurrentFiles)),
			)
		}
		outcomes = append(outcomes, runTasks(concurrentFiles, secrets, debug, fp, multiFile)...)
	}
	if len(sequentialFiles) > 0 {
		utils.PrintInfoStderr(
			fmt.Sprintf("Processing %d files sequentially...", len(sequentialFiles)),
		)
		outcomes = append(
			outcomes,
			processFilesSequentially(sequentialFiles, secrets, debug, multiFile)...)
	}

	if totalFiles == 0 {
		utils.PrintWarningStderr(
			"No files were processed. Please check your path or directory arguments.",
		)
		return
	}

	if multiFile {
		printRunSummary(outcomes, time.Since(overallStart))
	}
}

// printRunSummary prints "✓ N succeeded, ✗ M failed in T" plus a list of
// failed file paths. Only invoked for multi-file runs — single-file outcome
// is already obvious from the per-file outcome line.
func printRunSummary(outcomes []outcome, total time.Duration) {
	succeeded := 0
	var failed []outcome
	for _, o := range outcomes {
		if o.ok {
			succeeded++
		} else {
			failed = append(failed, o)
		}
	}

	utils.PrintInfoStderr(fmt.Sprintf(
		"%d succeeded, %d failed in %s", succeeded, len(failed), formatDuration(total),
	))
	for _, f := range failed {
		utils.PrintInfoStderr("  ✗ " + filepath.Base(f.path))
	}
}

// formatDuration renders a duration tightly: 142ms, 1.2s, 1m23s.
// time.Duration's String() can yield clutter like 1.234567s; this trims it.
func formatDuration(d time.Duration) string {
	switch {
	case d < time.Second:
		return fmt.Sprintf("%dms", d.Milliseconds())
	case d < time.Minute:
		return fmt.Sprintf("%.1fs", d.Seconds())
	default:
		m := int(d / time.Minute)
		s := int((d % time.Minute) / time.Second)
		return fmt.Sprintf("%dm%ds", m, s)
	}
}

// printOutcome renders one ✓/✗ line per file. status is empty for non-API
// kinds (Auth2, future kinds) — the line just shows the timing. Errors are
// flattened to a single line so the outcome list stays scannable; if the
// underlying error has actionable detail (e.g. a hint), it gets printed on
// a follow-up indented line so the cause is still discoverable.
func printOutcome(o outcome) {
	name := filepath.Base(o.path)
	dur := formatDuration(o.duration)
	if o.ok {
		bracket := dur
		if o.status != "" {
			bracket = o.status + ", " + dur
		}
		utils.PrintSuccessStderr(fmt.Sprintf("%s [%s]", name, bracket))
		return
	}
	// Failure path: status may still be set if the HTTP call succeeded but
	// downstream processing failed. Surface whatever we have.
	bracket := dur
	if o.status != "" {
		bracket = o.status + ", " + dur
	}
	headline, hint := splitErrorForOutcome(o.err)
	utils.PrintErrorStderr(fmt.Sprintf("%s [%s]: %s", name, bracket, headline))
	if hint != "" {
		// Two-space indent groups the hint visually under the failure line —
		// matches the summary's `  ✗ X` indent so the eye reads them as one block.
		fmt.Fprintf(os.Stderr, "  %s\n", hint)
	}
}

// splitErrorForOutcome flattens an error chain into one short headline plus
// an optional hint extracted from the deepest error message. The headline is
// wrapper-context (file path, key path) joined with " → " for readability;
// the hint is the trailing actionable sentence (e.g. "Add ... to env/X.env").
//
// Why bother: errors here are wrapped 3-4 deep ("substituting ...:
// substituting ...: key X not found in environment Y. Add ..."). Printing
// the whole chain in one line is unreadable; printing only the leaf loses
// the file/key context. We keep both, just present them tidily.
func splitErrorForOutcome(err error) (headline, hint string) {
	if err == nil {
		return "", ""
	}
	msg := err.Error()
	// Collapse any embedded newlines a wrapper may have injected (legacy
	// ColorError used to do this). Spaces preserve the message; tabs become
	// single spaces too. strings.Join keeps it compact.
	msg = strings.ReplaceAll(msg, "\n", " ")
	msg = strings.ReplaceAll(msg, "\t", " ")
	// Trim any leftover ANSI escape codes from older wrappers.
	msg = ansiInOutcome.ReplaceAllString(msg, "")
	msg = strings.TrimSpace(msg)

	// Heuristic split: the deepest error often ends with a sentence starting
	// with "Add ...", "Run ...", or "Use ..." — a hint. Pull it onto its own
	// line so the user's eye lands on it.
	for _, marker := range []string{`. Add "`, ". Run ", ". Use ", ". Try "} {
		if i := strings.LastIndex(msg, marker); i >= 0 {
			return strings.TrimSpace(msg[:i+1]), strings.TrimSpace(msg[i+2:])
		}
	}
	return msg, ""
}

// ansiInOutcome strips ANSI SGR escape sequences from error strings.
// Older wrappers (ColorError) baked colors into errors; we don't want those
// surviving into the outcome line.
var ansiInOutcome = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// runTasks manages the go tasks with a limited worker pool. Returns one
// outcome per file in the order they finished. multiFile gates whether the
// per-file outcome line is printed (skipped for single-file runs).
func runTasks(
	filePathList []string,
	secretsMap map[string]any,
	debug bool,
	fp string,
	multiFile bool,
) []outcome {
	maxWorkers := utils.GetWorkers(nil)
	maxRetries := 3
	timeout := 60 * time.Second

	if fp != "" {
		maxRetries = 1
	}

	var wg sync.WaitGroup
	taskChan := make(chan string, len(filePathList))
	resultChan := make(chan outcome, len(filePathList))

	for _, path := range filePathList {
		taskChan <- path
	}
	close(taskChan)

	for i := range maxWorkers {
		wg.Add(1)
		go func(_ int) {
			defer wg.Done()

			for path := range taskChan {
				var final outcome

				for attempt := 0; attempt < maxRetries; attempt++ {
					if attempt > 0 {
						backoffDuration := time.Duration(1<<(attempt-1)) * time.Second
						utils.PrintWarningStderr(fmt.Sprintf("Retrying %s (attempt %d/%d) after %v",
							filepath.Base(path), attempt+1, maxRetries, backoffDuration))
						time.Sleep(backoffDuration)
					}

					ctx, cancel := context.WithTimeout(context.Background(), timeout)

					resCh := make(chan outcome, 1)
					go func() {
						resCh <- processTask(path, utils.CopyEnvMap(secretsMap), debug)
					}()

					select {
					case res := <-resCh:
						final = res
					case <-ctx.Done():
						final = outcome{
							path: path,
							ok:   false,
							err:  fmt.Errorf("timeout after %v", timeout),
						}
					}
					cancel()

					if final.ok || !isRetryable(final.err) {
						break
					}
				}

				// Single-file mode: response body prints to stdout already, so
				// suppress the success outcome line on success. Failures still
				// surface — a silent error must not look like success.
				if multiFile || !final.ok {
					printOutcome(final)
				}
				resultChan <- final
			}
		}(i)
	}
	wg.Wait()
	close(resultChan)

	outcomes := make([]outcome, 0, len(filePathList))
	for o := range resultChan {
		outcomes = append(outcomes, o)
	}
	return outcomes
}

// configError marks an error that came from parsing/validating the YAML
// config (not from the network). The retry loop checks for this type and
// fails fast — retrying a malformed file or missing env var won't help and
// just delays the user seeing the real problem.
type configError struct{ err error }

func (e *configError) Error() string { return e.err.Error() }
func (e *configError) Unwrap() error { return e.err }

// isRetryable reports whether the runner should retry after this error.
// Network/transport errors and timeouts are retryable; config errors and
// unsupported-kind errors are not.
func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	var cfgErr *configError
	return !errors.As(err, &cfgErr)
}

// processTask handles a single task and returns a structured outcome.
// Wall-clock duration is measured around the dispatched call so it reflects
// the actual API latency (plus YAML parse time, which is small).
func processTask(path string, secretsMap map[string]any, debug bool) outcome {
	start := time.Now()
	config, err := yamlparser.ParseConfig(path, secretsMap)
	if err != nil {
		return outcome{path: path, ok: false, duration: time.Since(start), err: &configError{err}}
	}

	switch {
	case config.IsAuth():
		err := features.SendAPIRequestForAuth2(secretsMap, path, debug)
		return outcome{path: path, ok: err == nil, duration: time.Since(start), err: err}
	case (config.IsAPI() || config.IsGraphql()):
		status, err := apicalls.SendAndSaveAPIRequest(secretsMap, path, debug)
		return outcome{
			path:     path,
			ok:       err == nil,
			status:   status,
			duration: time.Since(start),
			err:      err,
		}
	default:
		return outcome{
			path:     path,
			ok:       false,
			duration: time.Since(start),
			err:      &configError{fmt.Errorf("unsupported kind in file: %s", path)},
		}
	}
}

// processFilesSequentially handles files one by one. Returns outcomes in
// execution order. multiFile gates the per-file outcome line — single-file
// mode keeps stderr quiet on success since the response body already prints
// to stdout; failures always surface so a silent error isn't mistaken for OK.
func processFilesSequentially(
	filePaths []string,
	secretsMap map[string]any,
	debug bool,
	multiFile bool,
) []outcome {
	outcomes := make([]outcome, 0, len(filePaths))
	for _, path := range filePaths {
		fileEnv := utils.CopyEnvMap(secretsMap)
		o := processTask(path, fileEnv, debug)
		if multiFile || !o.ok {
			printOutcome(o)
		}
		outcomes = append(outcomes, o)
	}
	return outcomes
}

// generateFilePathList returns a slice of file paths based on the flags -f and -fp.
func generateFilePathList(fileName string, fp string) ([]string, error) {
	standardErrMsg := "to send api request(s), please provide a valid file name with \n'-f fileName' flag or  \n'-fp file/path/' "

	if fileName == "" && fp == "" {
		return nil, utils.ColorError(standardErrMsg)
	}

	var filePathList []string

	if fp != "" {
		filePathList = append(filePathList, fp)
	}

	if fileName != "" {
		if matchingPaths, err := utils.ListMatchingFiles(fileName); err != nil {
			utils.PrintErrorStderr(utils.ErrFilePathCollection + ": " + err.Error())
		} else {
			filePathList = append(filePathList, matchingPaths...)
		}
	}

	if len(filePathList) == 0 {
		return nil, utils.ColorError(standardErrMsg)
	}
	return filePathList, nil
}

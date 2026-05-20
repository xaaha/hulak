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
	"github.com/xaaha/hulak/pkg/tui"
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
	// Timeout overrides the default 60s per-request timeout. Zero means
	// "fall back to HULAK_TIMEOUT, then the 60s default". A YAML `timeout:`
	// field on the request file wins over this flag.
	Timeout time.Duration
	// Quiet suppresses the end-of-run summary table for multi-file runs.
	Quiet bool
	// SSHIdentity overrides the SSH key path for vault decryption.
	// Propagated via HULAK_SSH_IDENTITY for the duration of this execution.
	SSHIdentity string
}

// DefaultTimeout is the per-request timeout used when no override is set
// (no YAML `timeout:` field, no --timeout flag, no HULAK_TIMEOUT env var).
const DefaultTimeout = 60 * time.Second

// HulakTimeoutEnv is the env var users set to override the default timeout
// for a session without editing request files or passing --timeout.
const HulakTimeoutEnv = "HULAK_TIMEOUT"

// Execute runs the full pipeline: discover files, resolve env, execute requests.
// Returns an error if any request file failed — callers should propagate it
// so the top-level exit code is non-zero on partial success. A nil error means
// every dispatched request succeeded.
func Execute(f *Flags) error {
	// If --ssh-identity is set and the env var isn't already set by the shell,
	// propagate it so ResolveIdentity picks it up for vault decryption.
	// Uses env var (same mechanism as HULAK_MASTER_KEY) rather than threading
	// through 4 function signatures across 3 packages. Safe because Setenv
	// runs before any goroutines launch and Unsetenv runs after all complete.
	if f.SSHIdentity != "" && os.Getenv(utils.SSHIdentityEnvVar) == "" {
		if err := os.Setenv(utils.SSHIdentityEnvVar, f.SSHIdentity); err != nil {
			return fmt.Errorf("failed to set %s: %w", utils.SSHIdentityEnvVar, err)
		}
		defer os.Unsetenv(utils.SSHIdentityEnvVar)
	}

	// Resolve the flag/env layer of the timeout chain up front so a malformed
	// HULAK_TIMEOUT fails fast before any request work begins.
	baseTimeout, err := resolveBaseTimeout(f.Timeout)
	if err != nil {
		return err
	}

	fileList, concurrentDir, sequentialDir, err := discoverFilePaths(
		f.File,
		f.FilePath,
		f.Dir,
		f.Dirseq,
		f.Dir != "" || f.Dirseq != "",
	)
	if err != nil {
		return err
	}

	allPaths := slices.Concat(fileList, concurrentDir, sequentialDir)

	var envMap map[string]any
	if containsTemplateVars(allPaths) {
		if !utils.IsHulakProject() {
			return fmt.Errorf("not a hulak project — run 'hulak init' to set up")
		}
		if !f.EnvSet {
			selectedEnv, err := envSelector()
			if err != nil {
				return fmt.Errorf("environment selector: %w", err)
			}
			if selectedEnv == "" {
				return nil // user cancelled the picker; not an error
			}
			f.Env = selectedEnv
		}
		var err error
		envMap, err = InitializeProject(f.Env, true)
		if err != nil {
			return err
		}
	}

	return handleAPIRequests(
		envMap,
		f.Debug,
		f.Quiet,
		append(fileList, concurrentDir...),
		sequentialDir,
		baseTimeout,
	)
}

// ExecuteSingleFile runs a single file through the pipeline.
// Used by interactive mode where the file is already known.
//
// flagTimeout is the value of --timeout (zero if unset). HULAK_TIMEOUT and
// the default 60s still apply through resolveBaseTimeout.
func ExecuteSingleFile(
	envMap map[string]any,
	debug bool,
	filePath string,
	flagTimeout time.Duration,
) error {
	baseTimeout, err := resolveBaseTimeout(flagTimeout)
	if err != nil {
		return err
	}
	return handleAPIRequests(envMap, debug, false, []string{filePath}, nil, baseTimeout)
}

// resolveBaseTimeout combines the --timeout flag and HULAK_TIMEOUT env var
// into a single duration the runner uses when no per-file YAML override is
// set. Precedence: flag > env > DefaultTimeout. A non-empty but invalid env
// var returns an error so the user sees the typo instead of getting a silent
// fallback.
func resolveBaseTimeout(flagT time.Duration) (time.Duration, error) {
	if flagT > 0 {
		return flagT, nil
	}
	raw := os.Getenv(HulakTimeoutEnv)
	if raw == "" {
		return DefaultTimeout, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q: %w", HulakTimeoutEnv, raw, err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("%s must be positive, got %q", HulakTimeoutEnv, raw)
	}
	return d, nil
}

// containsTemplateVars returns true if any file in the list uses template vars.
func containsTemplateVars(paths []string) bool {
	return slices.ContainsFunc(paths, utils.FileHasTemplateVars)
}

// InitializeProject creates the env setup and returns the secrets map.
// In vault mode (.hulak/store.age), skips creating the legacy env/ folder.
func InitializeProject(env string, isCli bool) (map[string]any, error) {
	if vault.DetectStore() != vault.StoreAge {
		if err := envparser.CreateDefaultEnvs(nil); err != nil {
			return nil, err
		}
	}
	envMap, err := envparser.GenerateSecretsMap(env, isCli)
	if err != nil {
		return nil, err
	}
	return envMap, nil
}

// discoverFilePaths collects all file paths from -f, -fp, -dir, and -dirseq flags.
//
// When the user provides only single-file flags (no -dir/-dirseq) and the
// lookup fails, the error is fatal — there's nothing else to run, so it
// returns up to the caller. When dir flags are also present the file-flag
// failure becomes a warning so the directory work can still proceed.
// Directory-listing failures are always reported as warnings; an empty
// directory list isn't fatal on its own (callers may still have file-flag
// matches to execute).
func discoverFilePaths(
	fileName, fp, dir, dirseq string,
	hasDirFlags bool,
) ([]string, []string, []string, error) {
	var fileList, concurrentDir, sequentialDir []string

	if fp != "" || fileName != "" {
		list, err := generateFilePathList(fileName, fp)
		if err != nil {
			if !hasDirFlags {
				return nil, nil, nil, err
			}
			utils.PrintWarningStderr(fmt.Sprintf("file flags: %v", err))
		} else {
			fileList = list
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

	return fileList, concurrentDir, sequentialDir, nil
}

// outcome captures the result of executing one request file. Used to render
// the per-file outcome line and the end-of-run summary.
type outcome struct {
	path     string        // request file path
	ok       bool          // true if request returned without error (any status code)
	status   string        // HTTP status string, e.g. "200 OK"; empty for non-API kinds and pre-flight errors
	duration time.Duration // wall-clock time for the request itself
	err      error         // non-nil on failure
	// respBytes holds the serialized response body for the caller to print
	// after the spinner clears (single-file mode) or inline (multi-file mode).
	// Empty for non-API kinds, pre-flight errors, and transport failures.
	respBytes []byte
}

// handleAPIRequests processes API requests from pre-discovered file lists.
// Returns nil if every dispatched request succeeded; otherwise an error
// summarizing the failures so the call site can propagate a non-zero exit.
//
// baseTimeout is the resolved flag/env timeout (or DefaultTimeout). It is
// the floor used for each request unless the file's YAML overrides it.
func handleAPIRequests(
	secrets map[string]any,
	debug bool,
	quiet bool,
	concurrentFiles []string,
	sequentialFiles []string,
	baseTimeout time.Duration,
) error {
	totalFiles := len(concurrentFiles) + len(sequentialFiles)
	// Per-file outcome lines only help when multiple files are running — they
	// let the user track which file finished. With a single file there's
	// exactly one outcome and the line is just noise; suppress it.
	multiFile := totalFiles > 1

	overallStart := time.Now()
	var outcomes []outcome

	switch {
	case totalFiles == 1 && !quiet:
		// Single-file fast path: wrap the request in a stderr spinner so the
		// user has visible progress on slow APIs. The spinner self-suppresses
		// when stderr is piped or redirected, so scripted callers still get
		// a clean output stream. --quiet also bypasses the spinner via the
		// case guard above.
		outcomes = append(outcomes, runSingleWithSpinner(
			firstNonEmpty(concurrentFiles, sequentialFiles),
			secrets, debug, baseTimeout,
		))
	default:
		if len(concurrentFiles) > 0 {
			if len(concurrentFiles) > 1 || len(sequentialFiles) > 0 {
				utils.PrintInfoStderr(
					fmt.Sprintf("Processing %d files concurrently...", len(concurrentFiles)),
				)
			}
			outcomes = append(
				outcomes,
				runTasks(concurrentFiles, secrets, debug, baseTimeout)...)
		}
		if len(sequentialFiles) > 0 {
			utils.PrintInfoStderr(
				fmt.Sprintf("Processing %d files sequentially...", len(sequentialFiles)),
			)
			outcomes = append(
				outcomes,
				processFilesSequentially(sequentialFiles, secrets, debug, multiFile, baseTimeout)...)
		}
	}

	if totalFiles == 0 {
		utils.PrintWarningStderr(
			"No files were processed. Please check your path or directory arguments.",
		)
		return nil
	}

	if multiFile && !quiet {
		printRunSummary(outcomes, time.Since(overallStart))
	}

	// Aggregate failures into a single error so the exit code reflects them.
	// Per-file detail has already been printed by printOutcome; the error
	// returned here is just a short headline a top-level handler can surface
	// without duplicating what's already on screen.
	failed := 0
	for _, o := range outcomes {
		if !o.ok {
			failed++
		}
	}
	if failed > 0 {
		return &runFailureError{failed: failed, total: totalFiles}
	}
	return nil
}

// runFailureError signals "n of m files failed" so the exit code flips
// non-zero. The top-level error handler can recognize this type and skip
// printing — the per-file outcome lines and the summary have already
// communicated the failures to the user.
type runFailureError struct {
	failed int
	total  int
}

func (e *runFailureError) Error() string {
	// "request failed" is intentionally generic — current call sites all
	// gate printing on IsRunFailure() and skip the message entirely (the
	// per-file outcome line already explained what went wrong). It only
	// surfaces if a future caller forgets the IsRunFailure check; in that
	// case, generic-but-correct beats a stale chained message.
	if e.total == 1 {
		return "request failed"
	}
	return fmt.Sprintf("%d of %d files failed", e.failed, e.total)
}

// IsRunFailure reports whether err originated from a runner pipeline failure
// (one or more request files failed). Callers use this to suppress redundant
// "error: ..." printing — printOutcome already showed the detail.
func IsRunFailure(err error) bool {
	var rf *runFailureError
	return errors.As(err, &rf)
}

// printRunSummary prints a table of all outcomes followed by a totals line.
// Only invoked for multi-file runs — single-file outcome is already obvious
// from the per-file outcome line.
func printRunSummary(outcomes []outcome, total time.Duration) {
	headers := []string{"FILE", "RESULT", "STATUS", "DURATION"}
	var rows [][]string

	succeeded := 0
	failed := 0
	for _, o := range outcomes {
		result := utils.Green + utils.CheckMark + utils.ColorReset
		name := utils.Blue + filepath.Base(o.path) + utils.ColorReset
		errMsg := ""
		if o.ok {
			succeeded++
		} else {
			failed++
			result = utils.Red + utils.CrossMark + utils.ColorReset
			headline, _ := splitErrorForOutcome(o.err)
			errMsg = headline
		}
		rows = append(rows, []string{
			name,
			result,
			o.status,
			formatDuration(o.duration),
			errMsg,
		})
	}
	if failed > 0 {
		headers = append(headers, "ERROR")
	}
	fmt.Fprintln(os.Stderr)
	_ = utils.PrintTable(os.Stderr, headers, rows, 0)
	fmt.Fprintln(os.Stderr)
	utils.PrintInfoStderr(fmt.Sprintf(
		"%d succeeded, %d failed in %s", succeeded, failed, formatDuration(total),
	))
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
func printOutcome(o *outcome) {
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
	headline, detail := splitErrorForOutcome(o.err)
	utils.PrintErrorStderr(fmt.Sprintf("%s [%s]: %s", name, bracket, headline))
	if detail != "" {
		// Two-space indent groups the detail block visually under the failure
		// line. Matches the summary's `  ✗ X` indent so the eye reads them as
		// one unit. Newlines in detail are preserved (YAML decoder context,
		// remediation steps, etc.) so each gets its own indented line.
		for line := range strings.SplitSeq(detail, "\n") {
			fmt.Fprintf(os.Stderr, "  %s\n", line)
		}
	}
}

// splitErrorForOutcome splits an error message into a one-line headline plus
// an optional multi-line detail block. The headline is the first line of the
// error (after ANSI stripping). The detail is everything after the first
// newline, with newlines preserved so callers can print the block indented
// under the headline.
//
// Why bother: wrapped errors (envparser missing-key with remediation steps,
// YAML decoder errors with caret-arrow context, etc.) carry information on
// multiple lines. Flattening to one line loses the context arrows; printing
// the whole chain inline makes the outcome row unreadable. Splitting keeps
// the row scannable and the detail discoverable.
func splitErrorForOutcome(err error) (headline, detail string) {
	if err == nil {
		return "", ""
	}
	msg := err.Error()
	msg = ansiInOutcome.ReplaceAllString(msg, "")
	msg = strings.TrimRight(msg, " \n\t")

	if first, rest, ok := strings.Cut(msg, "\n"); ok {
		headline = strings.TrimSpace(first)
		// Tabs widen unpredictably in terminals; normalize so the indented
		// detail block lines up regardless of the source error formatting.
		detail = strings.ReplaceAll(rest, "\t", "  ")
		detail = strings.TrimRight(detail, " \n\t")
		return headline, detail
	}
	return strings.TrimSpace(msg), ""
}

// ansiInOutcome strips ANSI SGR escape sequences from error strings.
// Defense-in-depth: errors should be plain text post-#180, but a third-party
// library could still emit ANSI and we don't want those surviving into the
// outcome line.
var ansiInOutcome = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// runTasks manages the go tasks with a limited worker pool. Returns one
// outcome per file in the order they finished. Per-file outcome lines are
// not printed here — the summary table handles concurrent results.
//
// baseTimeout is the per-request timeout when a file has no YAML `timeout:`
// override. processTask resolves the YAML override internally (single
// ParseConfig call) and threads the resulting context into the HTTP client
// for real cancellation — no leaked goroutines on timeout.
func runTasks(
	filePathList []string,
	secretsMap map[string]any,
	debug bool,
	baseTimeout time.Duration,
) []outcome {
	maxWorkers := utils.GetWorkers(nil)

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
				final := processTask(path, utils.CopyEnvMap(secretsMap), debug, baseTimeout)
				if final.respBytes != nil {
					apicalls.PrintRespBytes(final.respBytes)
				}
				if !final.ok {
					printOutcome(&final)
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

// runSingleWithSpinner wraps a single processTask call with a stderr spinner.
// TTY detection lives in tui.RunWithSpinnerOnStderr — when stderr is not a
// terminal (piped, redirected, CI) the wrapper falls through to the task
// directly and emits nothing during the wait. Failures are printed via the
// usual printOutcome path so the multi-line detail block surfaces.
func runSingleWithSpinner(
	path string,
	secrets map[string]any,
	debug bool,
	baseTimeout time.Duration,
) outcome {
	msg := fmt.Sprintf("Running '%s'...", filepath.Base(path))
	result, _ := tui.RunWithSpinnerOnStderr(msg, func() (any, error) {
		return processTask(path, utils.CopyEnvMap(secrets), debug, baseTimeout), nil
	})
	o := result.(outcome)
	// Spinner has cleared by this point. Print the response now so it lands
	// on a clean stderr line instead of overlapping with the spinner frame.
	if o.respBytes != nil {
		apicalls.PrintRespBytes(o.respBytes)
	}
	if !o.ok {
		printOutcome(&o)
	}
	return o
}

// firstNonEmpty returns the first element of the first non-empty slice. The
// single-file fast path uses this because the file might arrive via either
// the concurrent list (the typical `hulak run foo.yaml` flow) or the
// sequential list (`hulak run path/ --sequential` with a one-file directory).
//
// Returns "" when both slices are empty. The current call site guards on
// totalFiles == 1 so this branch is unreachable today, but the defensive
// return keeps a future caller from index-out-of-range if that guard drifts.
func firstNonEmpty(a, b []string) string {
	if len(a) > 0 {
		return a[0]
	}
	if len(b) > 0 {
		return b[0]
	}
	return ""
}

// processTask handles a single task and returns a structured outcome.
// Wall-clock duration is measured around the dispatched call so it reflects
// the actual API latency (plus YAML parse time, which is small).
//
// baseTimeout is the resolved flag/env timeout. The YAML `timeout:` field
// on the parsed config wins over base. The context created here flows into
// the HTTP client so the request is truly cancelled on deadline — no leaked
// goroutines.
func processTask(
	path string,
	secretsMap map[string]any,
	debug bool,
	baseTimeout time.Duration,
) outcome {
	start := time.Now()
	config, err := yamlparser.ParseConfig(path, secretsMap)
	if err != nil {
		return outcome{path: path, ok: false, duration: time.Since(start), err: err}
	}

	// Resolve per-file timeout: YAML wins over base.
	timeout := baseTimeout
	if d, _ := config.ParsedTimeout(); d > 0 {
		timeout = d
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	switch {
	case config.IsAuth():
		err := features.SendAPIRequestForAuth2(ctx, secretsMap, path, debug)
		return outcome{path: path, ok: err == nil, duration: time.Since(start), err: err}
	case config.IsAPI() || config.IsGraphql():
		respBytes, status, err := apicalls.SendAndSaveAPIRequest(ctx, secretsMap, path, debug)
		return outcome{
			path:      path,
			ok:        err == nil,
			status:    status,
			duration:  time.Since(start),
			err:       err,
			respBytes: respBytes,
		}
	default:
		return outcome{
			path:     path,
			ok:       false,
			duration: time.Since(start),
			err:      fmt.Errorf("unsupported kind in file: %s", path),
		}
	}
}

// processFilesSequentially handles files one by one. Returns outcomes in
// execution order. multiFile gates the per-file outcome line — single-file
// mode keeps stderr quiet on success since the response body already prints
// to stdout; failures always surface so a silent error isn't mistaken for OK.
//
// baseTimeout is the per-request timeout when a file has no YAML override.
// processTask resolves the YAML override internally and threads the context
// into the HTTP client — same cancellation path as runTasks.
func processFilesSequentially(
	filePaths []string,
	secretsMap map[string]any,
	debug bool,
	multiFile bool,
	baseTimeout time.Duration,
) []outcome {
	outcomes := make([]outcome, 0, len(filePaths))
	for _, path := range filePaths {
		o := processTask(path, utils.CopyEnvMap(secretsMap), debug, baseTimeout)
		if o.respBytes != nil {
			apicalls.PrintRespBytes(o.respBytes)
		}
		if multiFile || !o.ok {
			printOutcome(&o)
		}
		outcomes = append(outcomes, o)
	}
	return outcomes
}

// generateFilePathList returns a slice of file paths based on the flags -f and -fp.
func generateFilePathList(fileName string, fp string) ([]string, error) {
	standardErrMsg := "to send api request(s), please provide a valid file name with \n'-f fileName' flag or  \n'-fp file/path/' "

	if fileName == "" && fp == "" {
		return nil, errors.New(standardErrMsg)
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
		return nil, errors.New(standardErrMsg)
	}
	return filePathList, nil
}

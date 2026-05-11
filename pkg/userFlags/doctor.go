package userflags

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/vault"
)

// severity ranks a doctor finding from informational to blocking.
type severity int

const (
	sevInfo  severity = iota // context-only, does not affect exit code
	sevOk                    // check passed
	sevWarn                  // potential problem
	sevError                 // must fix before vault is usable
)

func (s severity) String() string {
	switch s {
	case sevInfo:
		return "info"
	case sevOk:
		return "ok"
	case sevWarn:
		return "warn"
	case sevError:
		return "error"
	default:
		return "unknown"
	}
}

// finding is a single doctor check result.
type finding struct {
	check    string       // stable ID, e.g. "identity-mode"
	severity severity     // one of sevInfo / sevOk / sevWarn / sevError
	message  string       // human-readable description
	fix      string       // remediation advice (always informational)
	auto     func() error // non-nil iff --fix can safely repair this
}

// doctorReport collects all findings for a single run.
type doctorReport struct {
	project  string
	backend  string
	findings []finding
}

// summary counts findings by severity.
type summary struct {
	Ok    int `json:"ok"`
	Warn  int `json:"warn"`
	Error int `json:"error"`
	Info  int `json:"info"`
}

func (r *doctorReport) summary() summary {
	var s summary
	for _, f := range r.findings {
		switch f.severity {
		case sevInfo:
			s.Info++
		case sevOk:
			s.Ok++
		case sevWarn:
			s.Warn++
		case sevError:
			s.Error++
		}
	}
	return s
}

// exitCode returns 0 for ok/info-only, 1 for warnings, 2 for errors.
func (r *doctorReport) exitCode() int {
	s := r.summary()
	if s.Error > 0 {
		return 2
	}
	if s.Warn > 0 {
		return 1
	}
	return 0
}

// --- human output (stderr) -------------------------------------------------

func (r *doctorReport) printHuman() {
	utils.PrintInfoStderr(fmt.Sprintf("Project: %s", r.project))
	utils.PrintInfoStderr(fmt.Sprintf("Backend: %s\n", r.backend))

	for _, f := range r.findings {
		printFinding(f)
	}

	s := r.summary()
	utils.PrintInfoStderr(fmt.Sprintf("\n%d ok, %d warning, %d error", s.Ok, s.Warn, s.Error))
}

func printFinding(f finding) {
	switch f.severity {
	case sevOk:
		// ✔ message
		fmt.Fprintf(os.Stderr, "%s%s%s %s\n", utils.Green, utils.CheckMark, utils.ColorReset, f.message)
	case sevWarn:
		// ⚠ warning: message
		fmt.Fprintf(os.Stderr, "%s%s warning:%s %s\n", utils.Yellow, utils.WarningMark, utils.ColorReset, f.message)
	case sevError:
		// ✖ error: message
		fmt.Fprintf(os.Stderr, "%s%s error:%s %s\n", utils.Red, utils.CrossMark, utils.ColorReset, f.message)
	case sevInfo:
		// ℹ info: message
		fmt.Fprintf(os.Stderr, "%s%s info:%s %s\n", utils.Blue, utils.InfoMark, utils.ColorReset, f.message)
	}
	if f.fix != "" {
		utils.PrintInfoStderr(fmt.Sprintf("  Fix: %s", f.fix))
	}
}

// --- JSON output (stdout) --------------------------------------------------

type jsonFinding struct {
	Check       string `json:"check"`
	Severity    string `json:"severity"`
	Message     string `json:"message"`
	Fix         string `json:"fix,omitempty"`
	AutoFixable bool   `json:"auto_fixable"`
}

type jsonReport struct {
	Project  string        `json:"project"`
	Backend  string        `json:"backend"`
	Findings []jsonFinding `json:"findings"`
	Summary  summary       `json:"summary"`
}

func (r *doctorReport) printJSON() {
	jf := make([]jsonFinding, len(r.findings))
	for i, f := range r.findings {
		jf[i] = jsonFinding{
			Check:       f.check,
			Severity:    f.severity.String(),
			Message:     f.message,
			Fix:         f.fix,
			AutoFixable: f.auto != nil,
		}
	}
	jr := jsonReport{
		Project:  r.project,
		Backend:  r.backend,
		Findings: jf,
		Summary:  r.summary(),
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(jr) // stdout write failure is unrecoverable
}

// --- --fix flow ------------------------------------------------------------

func (r *doctorReport) runFixes(autoYes bool) {
	for i, f := range r.findings {
		if f.auto == nil {
			continue
		}
		printFinding(f)

		if !autoYes {
			ok, err := utils.ConfirmAction("  Fix? [Y/n] ")
			if err != nil || !ok {
				continue
			}
		}

		if err := f.auto(); err != nil {
			utils.PrintErrorStderr(fmt.Sprintf("  failed: %v", err))
			continue
		}

		// re-check shows green on success
		r.findings[i].severity = sevOk
		r.findings[i].fix = ""
		r.findings[i].auto = nil
		utils.PrintSuccessStderr("  fixed")
	}
}

// --- orchestrator ----------------------------------------------------------

type doctorOpts struct {
	fix     bool
	yes     bool
	jsonOut bool
}

func runDoctor(opts doctorOpts) {
	projectRoot, found := utils.FindProjectRoot()
	storeType := vault.DetectStore()

	if opts.fix && opts.jsonOut {
		utils.PrintErrorStderr("--fix and --json are mutually exclusive")
		os.Exit(1)
	}

	if !found || storeType == vault.StoreNone {
		utils.PrintInfoStderr("Not a hulak project. Run 'hulak init' to start one.")
		return
	}

	report := &doctorReport{project: projectRoot}

	switch storeType {
	case vault.StoreAge:
		report.backend = "vault (.hulak/store.age)"
		report.findings = collectVaultFindings()
	case vault.StoreClassic:
		report.backend = "classic (env/)"
		report.findings = collectClassicFindings(projectRoot)
	}

	if opts.jsonOut {
		report.printJSON()
		os.Exit(report.exitCode())
	}

	if opts.fix {
		report.printHuman()
		report.runFixes(opts.yes)
	} else {
		report.printHuman()
	}

	os.Exit(report.exitCode())
}

// collectVaultFindings runs all vault-aware checks.
func collectVaultFindings() []finding {
	return append([]finding{},
		// Bootstrap chain (checks 1-12)
		checkIdentityPresent(),
		checkIdentityMode(),
		checkIdentityNotInGit(),
		checkIdentityLeakedInProject(),
		checkConfigDirMode(),
		checkStoreMode(),
		checkStoreEncrypted(),
		checkStoreDecrypts(),
		checkRecipientsExist(),
		checkRecipientsValid(),
		checkRecipientsMode(),
		checkRecipientsCommitted(),
		// Drift + remaining checks (13-18)
		checkRecipientDrift(),
		checkStoreNotGitignored(),
		checkLegacyKeyPub(),
		checkDualBackend(),
		checkDualIdentity(),
		checkStoreSize(),
	)
}

// collectClassicFindings wraps legacy checks into findings and adds
// an info-level suggestion to migrate to the vault backend.
func collectClassicFindings(projectRoot string) []finding {
	envPath := filepath.Join(projectRoot, utils.EnvironmentFolder)

	// Print inventory header
	printInventory(envPath)

	// Run legacy checks and convert to findings
	warnings := collectWarnings(envPath)

	var findings []finding
	if len(warnings) == 0 {
		findings = append(findings, finding{
			check:    "classic-health",
			severity: sevOk,
			message:  "classic backend looks healthy",
		})
	}
	for _, w := range warnings {
		findings = append(findings, finding{
			check:    "classic-issue",
			severity: sevWarn,
			message:  w.message,
			fix:      w.fix,
		})
	}

	// Suggest migration
	findings = append(findings, finding{
		check:    "migrate-suggestion",
		severity: sevInfo,
		message:  "consider migrating to the encrypted vault with 'hulak secrets migrate'",
	})

	return findings
}

func printInventory(envPath string) {
	entries, err := os.ReadDir(envPath)
	if err != nil {
		return
	}

	utils.PrintInfoStderr(utils.EnvironmentFolder + "/")
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		utils.PrintInfoStderr("  " + e.Name())
	}
	utils.PrintInfoStderr("")
}

// --- legacy helpers (classic backend checks) ---

type warning struct {
	message string
	fix     string
}

func collectWarnings(envPath string) []warning {
	var warnings []warning
	warnings = append(warnings, checkGitignore()...)
	warnings = append(warnings, checkEnvPermissions(envPath)...)
	warnings = append(warnings, checkGitHistory()...)
	return warnings
}

func checkGitignore() []warning {
	gitignorePath, err := utils.CreatePath(".gitignore")
	if err != nil {
		return []warning{{
			message: fmt.Sprintf(
				"%s/ is not gitignored — secrets may be committed",
				utils.EnvironmentFolder,
			),
			fix: fmt.Sprintf(
				"echo \"%s/\" >> .gitignore",
				utils.EnvironmentFolder,
			),
		}}
	}

	if !utils.FileExists(gitignorePath) {
		return []warning{{
			message: fmt.Sprintf(
				"%s/ is not gitignored — secrets may be committed",
				utils.EnvironmentFolder,
			),
			fix: fmt.Sprintf(
				"echo \"%s/\" >> .gitignore",
				utils.EnvironmentFolder,
			),
		}}
	}

	file, err := os.Open(gitignorePath)
	if err != nil {
		return []warning{{
			message: "could not read .gitignore",
		}}
	}
	defer file.Close()

	entry := utils.EnvironmentFolder + "/"
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == entry || line == utils.EnvironmentFolder {
			return nil
		}
	}

	return []warning{{
		message: fmt.Sprintf(
			"%s/ is not gitignored — secrets may be committed",
			utils.EnvironmentFolder,
		),
		fix: fmt.Sprintf(
			"echo \"%s/\" >> .gitignore",
			utils.EnvironmentFolder,
		),
	}}
}

func checkEnvPermissions(envPath string) []warning {
	entries, err := os.ReadDir(envPath)
	if err != nil {
		return nil
	}

	var warnings []warning
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), utils.DefaultEnvFileSuffix) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.Mode().Perm()&0o077 != 0 {
			warnings = append(warnings, warning{
				message: fmt.Sprintf(
					"%s has loose permissions (%o)",
					e.Name(), info.Mode().Perm(),
				),
				fix: fmt.Sprintf(
					"chmod 600 %s/%s",
					utils.EnvironmentFolder, e.Name(),
				),
			})
		}
	}
	return warnings
}

func checkGitHistory() []warning {
	if _, err := exec.LookPath("git"); err != nil {
		return nil
	}

	gitGlob := utils.EnvironmentFolder + "/*" + utils.DefaultEnvFileSuffix
	cmd := exec.Command(
		"git", "log", "--all", "--diff-filter=A",
		"--name-only", "--pretty=format:", "--", gitGlob,
	)
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	leaked := strings.TrimSpace(string(output))
	if leaked == "" {
		return nil
	}

	var files []string
	for f := range strings.SplitSeq(leaked, "\n") {
		if trimmed := strings.TrimSpace(f); trimmed != "" {
			files = append(files, trimmed)
		}
	}
	if len(files) == 0 {
		return nil
	}

	return []warning{{
		message: fmt.Sprintf(
			"%s files found in git history: %s",
			utils.DefaultEnvFileSuffix, strings.Join(files, ", "),
		),
		fix: "consider removing with git filter-repo or BFG Repo-Cleaner",
	}}
}

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

// --- severity ----------------------------------------------------------------

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

// sevDisplay maps a severity to its colored mark for table output.
type sevDisplay struct {
	color string
	mark  string
}

var sevDisplays = [4]sevDisplay{
	sevInfo:  {utils.Blue, utils.InfoMark},
	sevOk:    {utils.Green, utils.CheckMark},
	sevWarn:  {utils.Yellow, utils.WarningMark},
	sevError: {utils.Red, utils.CrossMark},
}

// --- finding -----------------------------------------------------------------

// finding is a single doctor check result.
type finding struct {
	check    string       // stable ID, e.g. "identity-mode"
	severity severity     // one of sevInfo / sevOk / sevWarn / sevError
	message  string       // human-readable description
	fix      string       // remediation advice (always informational)
	auto     func() error // non-nil iff --fix can safely repair this
}

func skipFinding(check, reason string) finding {
	return finding{check: check, severity: sevInfo, message: reason}
}

func okFinding(check, msg string) finding {
	return finding{check: check, severity: sevOk, message: msg}
}

// --- report ------------------------------------------------------------------

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

// --- human output (stderr) ---------------------------------------------------

func (r *doctorReport) printHuman() {
	utils.PrintInfoStderr(fmt.Sprintf("Project: %s", r.project))
	utils.PrintInfoStderr(fmt.Sprintf("Backend: %s\n", r.backend))

	headers := []string{"RESULT", "MESSAGE"}
	hasFix := false
	for _, f := range r.findings {
		if f.fix != "" {
			hasFix = true
			break
		}
	}
	if hasFix {
		headers = append(headers, "FIX")
	}

	rows := make([][]string, len(r.findings))
	for i, f := range r.findings {
		d := sevDisplays[f.severity]
		row := []string{d.color + d.mark + utils.ColorReset, f.message}
		if hasFix {
			row = append(row, f.fix)
		}
		rows[i] = row
	}

	_ = utils.PrintTable(os.Stderr, headers, rows, 0)

	s := r.summary()
	utils.PrintInfoStderr(fmt.Sprintf("\n%d ok, %d warning, %d error", s.Ok, s.Warn, s.Error))
}

// --- JSON output (stdout) ----------------------------------------------------

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

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(jsonReport{
		Project:  r.project,
		Backend:  r.backend,
		Findings: jf,
		Summary:  r.summary(),
	})
}

// --- --fix flow --------------------------------------------------------------

// runFixes applies auto-fixable findings. Findings were already printed by
// printHuman, so this only shows the fix prompt and result.
func (r *doctorReport) runFixes(autoYes bool) {
	for i, f := range r.findings {
		if f.auto == nil {
			continue
		}

		if !autoYes {
			ok, err := utils.ConfirmAction(fmt.Sprintf("  Fix %s (%s)? [Y/n] ", f.check, f.fix))
			if err != nil || !ok {
				continue
			}
		}

		if err := f.auto(); err != nil {
			utils.PrintErrorStderr(fmt.Sprintf("  fix %s failed: %v", f.check, err))
			continue
		}

		r.findings[i].severity = sevOk
		r.findings[i].fix = ""
		r.findings[i].auto = nil
		utils.PrintSuccessStderr(fmt.Sprintf("  fixed: %s", f.check))
	}
}

// --- orchestrator ------------------------------------------------------------

type doctorOpts struct {
	fix     bool
	yes     bool
	jsonOut bool
}

// runDoctor runs all health checks and returns the exit code:
// 0 = ok/info only, 1 = warnings, 2 = errors.
func runDoctor(opts doctorOpts) int {
	projectRoot, found := utils.FindProjectRoot()
	storeType := vault.DetectStore()

	if opts.fix && opts.jsonOut {
		utils.PrintErrorStderr("--fix and --json are mutually exclusive")
		return 1
	}

	if !found || storeType == vault.StoreNone {
		utils.PrintInfoStderr("Not a hulak project. Run 'hulak init' to start one.")
		return 0
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
		return report.exitCode()
	}

	if opts.fix {
		report.printHuman()
		report.runFixes(opts.yes)
	} else {
		report.printHuman()
	}

	return report.exitCode()
}

// collectVaultFindings runs all vault-aware checks.
func collectVaultFindings() []finding {
	return []finding{
		// Identity chain
		checkIdentityPresent(),
		checkIdentityMode(),
		checkIdentityNotInGit(),
		checkIdentityLeakedInProject(),
		// Store
		checkStoreMode(),
		checkStoreEncrypted(),
		checkStoreDecrypts(),
		// Recipients
		checkRecipientsExist(),
		checkRecipientsValid(),
		checkRecipientsCommitted(),
		// Drift + misc
		checkRecipientDrift(),
		checkStoreNotGitignored(),
		checkLegacyKeyPub(),
		checkDualBackend(),
		checkDualIdentity(),
		checkStoreSize(),
	}
}

// collectClassicFindings runs classic-backend checks and suggests migration.
func collectClassicFindings(projectRoot string) []finding {
	envPath := filepath.Join(projectRoot, utils.EnvironmentFolder)
	printInventory(envPath)

	var findings []finding
	findings = append(findings, checkGitignore()...)
	findings = append(findings, checkEnvPermissions(envPath)...)
	findings = append(findings, checkGitHistory()...)

	if len(findings) == 0 {
		findings = append(findings, okFinding("classic-health", "classic backend looks healthy"))
	}

	findings = append(findings, finding{
		check:    "migrate-suggestion",
		severity: sevInfo,
		message:  "consider migrating to the encrypted vault with 'hulak secrets migrate'",
	})

	return findings
}

// --- classic checks ----------------------------------------------------------

func checkGitignore() []finding {
	notIgnored := finding{
		check:    "classic-gitignore",
		severity: sevWarn,
		message:  fmt.Sprintf("%s/ is not gitignored — secrets may be committed", utils.EnvironmentFolder),
		fix:      fmt.Sprintf("echo \"%s/\" >> .gitignore", utils.EnvironmentFolder),
	}

	gitignorePath, err := utils.CreatePath(".gitignore")
	if err != nil || !utils.FileExists(gitignorePath) {
		return []finding{notIgnored}
	}

	file, err := os.Open(gitignorePath)
	if err != nil {
		return []finding{{
			check:    "classic-gitignore",
			severity: sevWarn,
			message:  "could not read .gitignore",
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

	return []finding{notIgnored}
}

func checkEnvPermissions(envPath string) []finding {
	entries, err := os.ReadDir(envPath)
	if err != nil {
		return nil
	}

	var findings []finding
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), utils.DefaultEnvFileSuffix) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.Mode().Perm()&0o077 != 0 {
			findings = append(findings, finding{
				check:    "classic-permissions",
				severity: sevWarn,
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
	return findings
}

func checkGitHistory() []finding {
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

	return []finding{{
		check:    "classic-git-history",
		severity: sevWarn,
		message: fmt.Sprintf(
			"%s files found in git history: %s",
			utils.DefaultEnvFileSuffix, strings.Join(files, ", "),
		),
		fix: "consider removing with git filter-repo or BFG Repo-Cleaner",
	}}
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

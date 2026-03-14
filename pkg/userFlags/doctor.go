package userflags

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/xaaha/hulak/pkg/utils"
)

type warning struct {
	message string
	fix     string
}

func runDoctor() {
	envPath, err := utils.CreatePath(utils.EnvironmentFolder)
	if err != nil {
		utils.PrintWarning(fmt.Sprintf(
			"%s/ could not be resolved", utils.EnvironmentFolder,
		))
		return
	}

	info, statErr := os.Stat(envPath)
	if statErr != nil || !info.IsDir() {
		utils.PrintWarning(fmt.Sprintf(
			"%s/ directory not found. Create it:\n    hulak init", utils.EnvironmentFolder,
		))
		return
	}

	printInventory(envPath)

	warnings := collectWarnings(envPath)
	if len(warnings) == 0 {
		utils.PrintGreen(fmt.Sprintf("  %s No issues found", utils.CheckMark))
		return
	}
	for _, w := range warnings {
		msg := fmt.Sprintf("  %s %s", utils.CrossMark, w.message)
		if w.fix != "" {
			msg += "\n    " + w.fix
		}
		utils.PrintWarning(msg)
	}
}

func printInventory(envPath string) {
	entries, err := os.ReadDir(envPath)
	if err != nil {
		return
	}

	fmt.Println(utils.EnvironmentFolder + "/")
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		fmt.Println("  " + e.Name())
	}
	fmt.Println()
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
	for _, f := range strings.Split(leaked, "\n") {
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

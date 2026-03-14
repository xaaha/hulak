package userflags

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/xaaha/hulak/pkg/utils"
)

type checkResult struct {
	passed  bool
	message string
}

func runDoctor() {
	results := []checkResult{
		checkEnvDir(),
		checkGitignore(),
		checkEnvPermissions(),
		checkGitHistory(),
		checkEnvFileList(),
	}

	for _, r := range results {
		if r.passed {
			utils.PrintGreen(fmt.Sprintf("  %s %s", utils.CheckMark, r.message))
		} else {
			utils.PrintWarning(fmt.Sprintf("  %s", r.message))
		}
	}
}

func checkEnvDir() checkResult {
	envPath, err := utils.CreatePath(utils.EnvironmentFolder)
	if err != nil {
		return checkResult{false, fmt.Sprintf("could not resolve %s/ path", utils.EnvironmentFolder)}
	}
	info, err := os.Stat(envPath)
	if err != nil || !info.IsDir() {
		return checkResult{
			false,
			fmt.Sprintf("%s/ directory not found — run 'hulak init'", utils.EnvironmentFolder),
		}
	}
	return checkResult{true, fmt.Sprintf("%s/ directory found", utils.EnvironmentFolder)}
}

func checkGitignore() checkResult {
	gitignorePath, err := utils.CreatePath(".gitignore")
	if err != nil {
		return checkResult{false, "could not resolve .gitignore path"}
	}

	if !utils.FileExists(gitignorePath) {
		return checkResult{
			false,
			fmt.Sprintf("no .gitignore found — %s/ secrets may be committed", utils.EnvironmentFolder),
		}
	}

	file, err := os.Open(gitignorePath)
	if err != nil {
		return checkResult{false, "could not read .gitignore"}
	}
	defer file.Close()

	entry := utils.EnvironmentFolder + "/"
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == entry || line == utils.EnvironmentFolder {
			return checkResult{true, fmt.Sprintf("%s/ is in .gitignore", utils.EnvironmentFolder)}
		}
	}

	return checkResult{
		false,
		fmt.Sprintf("%s/ is not in .gitignore — secrets may be committed", utils.EnvironmentFolder),
	}
}

func checkEnvPermissions() checkResult {
	envPath, err := utils.CreatePath(utils.EnvironmentFolder)
	if err != nil {
		return checkResult{false, fmt.Sprintf("could not resolve %s/ path", utils.EnvironmentFolder)}
	}

	entries, err := os.ReadDir(envPath)
	if err != nil {
		return checkResult{true, "no env files to check permissions"}
	}

	var loose []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), utils.DefaultEnvFileSuffix) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		perm := info.Mode().Perm()
		if perm&0o077 != 0 {
			loose = append(loose, fmt.Sprintf("%s (%o)", e.Name(), perm))
		}
	}

	if len(loose) > 0 {
		return checkResult{
			false,
			fmt.Sprintf(
				"env files with loose permissions (should be 600): %s",
				strings.Join(loose, ", "),
			),
		}
	}
	return checkResult{true, "env file permissions are restrictive"}
}

func checkGitHistory() checkResult {
	_, err := exec.LookPath("git")
	if err != nil {
		return checkResult{true, "git not found, skipping history check"}
	}

	gitGlob := utils.EnvironmentFolder + "/*" + utils.DefaultEnvFileSuffix
	cmd := exec.Command("git", "log", "--all", "--diff-filter=A", "--name-only", "--pretty=format:", "--", gitGlob)
	output, err := cmd.Output()
	if err != nil {
		return checkResult{true, "not a git repo, skipping history check"}
	}

	leaked := strings.TrimSpace(string(output))
	if leaked != "" {
		files := strings.Split(leaked, "\n")
		var nonEmpty []string
		for _, f := range files {
			if strings.TrimSpace(f) != "" {
				nonEmpty = append(nonEmpty, strings.TrimSpace(f))
			}
		}
		if len(nonEmpty) > 0 {
			return checkResult{
				false,
				fmt.Sprintf(".env files found in git history: %s", strings.Join(nonEmpty, ", ")),
			}
		}
	}

	return checkResult{true, "no .env files found in git history"}
}

func checkEnvFileList() checkResult {
	envFiles, err := utils.GetEnvFiles()
	if err != nil || len(envFiles) == 0 {
		return checkResult{true, "no environment files found"}
	}

	var names []string
	for _, f := range envFiles {
		name := strings.TrimSuffix(f, utils.DefaultEnvFileSuffix)
		names = append(names, name)
	}

	envPath, _ := utils.CreatePath(utils.EnvironmentFolder)
	return checkResult{
		true,
		fmt.Sprintf(
			"%d environment(s) in %s: %s",
			len(names),
			filepath.Base(envPath),
			strings.Join(names, ", "),
		),
	}
}

// Package userflags have everything related to user's flags & subcommands
package userflags

import (
	"bufio"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xaaha/hulak/pkg/envparser"
	"github.com/xaaha/hulak/pkg/utils"
)

//go:embed apiOptions.hk.yaml
var embeddedFiles embed.FS

// InitClassicProject sets up the plaintext env/ layout: env/ directory,
// global.env, the apiOptions.hk.yaml example, and an env/ entry in .gitignore.
//
// Refuses to run if .hulak/ already exists in the current directory — that
// signals the user has already initialized the encrypted vault, and bolting
// a parallel plaintext layout next to it would create two sources of truth
// for environment values. The user has to remove .hulak/ explicitly to opt
// out of the vault.
func InitClassicProject() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not determine current directory: %w", err)
	}
	if utils.DirExists(filepath.Join(cwd, utils.HiddenProjectName)) {
		return fmt.Errorf(
			"refusing to create plaintext env/ layout: %s/ already exists "+
				"(this project is using the encrypted vault) — "+
				"remove %s/ first if you really want to switch",
			utils.HiddenProjectName, utils.HiddenProjectName,
		)
	}

	if err := envparser.CreateDefaultEnvs(nil); err != nil {
		return err
	}

	if err := ensureGitignoreEntry(); err != nil {
		utils.PrintWarningStderr(fmt.Sprintf("could not update .gitignore: %v", err))
	}

	root, err := utils.CreatePath(utils.APIOptions)
	if err != nil {
		return err
	}

	// Don't clobber a customized example file. `hulak init` is designed to be
	// safe to re-run; overwriting user-edited content would defeat that.
	if utils.FileExists(root) {
		utils.PrintWarningStderr(
			fmt.Sprintf("Kept existing '%s' (delete it to regenerate)", utils.APIOptions),
		)
		utils.PrintSuccessStderr("Done")
		return nil
	}

	content, err := embeddedFiles.ReadFile(utils.APIOptions)
	if err != nil {
		return err
	}

	if err := os.WriteFile(root, content, utils.FilePer); err != nil {
		return fmt.Errorf("error on writing '%s' file: %s", utils.APIOptions, err)
	}

	utils.PrintSuccessStderr(fmt.Sprintf("Created '%s'", utils.APIOptions))
	utils.PrintSuccessStderr("Done")
	return nil
}

// ensureGitignoreEntry adds env/ to .gitignore if not already present.
func ensureGitignoreEntry() error {
	gitignorePath, err := utils.CreatePath(".gitignore")
	if err != nil {
		return fmt.Errorf("could not resolve .gitignore path: %w", err)
	}

	// .gitignored uses forward / for path
	entry := utils.EnvironmentFolder + "/"

	if utils.FileExists(gitignorePath) {
		file, err := os.Open(gitignorePath)
		if err != nil {
			return fmt.Errorf("could not read .gitignore: %w", err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == entry || line == utils.EnvironmentFolder {
				return nil
			}
		}
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("error reading .gitignore: %w", err)
		}
	}

	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, utils.FilePer)
	if err != nil {
		return fmt.Errorf("could not open .gitignore for writing: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("could not stat .gitignore: %w", err)
	}

	prefix := ""
	if info.Size() > 0 {
		content, err := os.ReadFile(gitignorePath)
		if err != nil {
			return fmt.Errorf("could not read .gitignore: %w", err)
		}
		if len(content) > 0 && content[len(content)-1] != '\n' {
			prefix = "\n"
		}
	}

	if _, err := fmt.Fprintf(f, "%s%s\n", prefix, entry); err != nil {
		return fmt.Errorf("could not write to .gitignore: %w", err)
	}

	return nil
}

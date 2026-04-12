// Package userflags have everything related to user's flags & subcommands
package userflags

import (
	"bufio"
	"embed"
	"fmt"
	"os"
	"strings"

	"github.com/xaaha/hulak/pkg/envparser"
	"github.com/xaaha/hulak/pkg/utils"
)

//go:embed apiOptions.hk.yaml
var embeddedFiles embed.FS

// InitDefaultProject performs the default hulak project initialization:
// creates env/ directory, global.env, and the apiOptions.hk.yaml example file.
// It also adds env/ to .gitignore if not already present.
func InitDefaultProject() error {
	if err := envparser.CreateDefaultEnvs(nil); err != nil {
		return err
	}

	if err := ensureGitignoreEntry(); err != nil {
		utils.PrintWarning(fmt.Sprintf("could not update .gitignore: %v", err))
	}

	content, err := embeddedFiles.ReadFile(utils.APIOptions)
	if err != nil {
		return err
	}

	root, err := utils.CreatePath(utils.APIOptions)
	if err != nil {
		return err
	}

	if err := os.WriteFile(root, content, utils.FilePer); err != nil {
		return fmt.Errorf("error on writing '%s' file: %s", utils.APIOptions, err)
	}

	utils.PrintGreen(fmt.Sprintf("Created '%s': %s", utils.APIOptions, utils.CheckMark))
	utils.PrintGreen("Done " + utils.CheckMark)
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

// Package userflags have everything related to user's flags & subcommands
package userflags

import (
	"fmt"
	"os"

	"github.com/xaaha/hulak/pkg/envparser"
	"github.com/xaaha/hulak/pkg/utils"
)

// InitDefaultProject performs the default hulak project initialization:
// creates env/ directory, global.env, and the apiOptions.hk.yaml example file.
func InitDefaultProject() error {
	if err := envparser.CreateDefaultEnvs(nil); err != nil {
		return err
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

func handleInit() error {
	err := initialize.Parse(os.Args[2:])
	if err != nil {
		return fmt.Errorf("\n invalid subcommand %v", err)
	}
	// Check if -env flag is present
	if *createEnvs {
		envs := initialize.Args()
		if len(envs) > 0 {
			for _, env := range envs {
				if err := envparser.CreateDefaultEnvs(&env); err != nil {
					utils.PrintRed(err.Error())
				}
			}
		} else {
			utils.PrintWarning("No environment names provided after -env flag")
		}
	} else {
		return InitDefaultProject()
	}
	return nil
}

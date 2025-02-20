package envparser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xaaha/hulak/pkg/utils"
)

/*
Creates environment folder and a default global.env file in it.
Optional: File names as a *string
*/
func CreateDefaultEnvs(envName *string) error {
	defaultEnv := utils.DefaultEnvVal
	if envName != nil {
		lowerCasedEnv := strings.ToLower(*envName)
		defaultEnv = lowerCasedEnv
	}
	projectRoot, err := os.Getwd()
	if err != nil {
		return err
	}

	// create an env folder in the root of the project
	envDirpath := filepath.Join(projectRoot, "env")
	envFilePath := filepath.Join(envDirpath, defaultEnv+".env") // global.env as the default
	if _, err := os.Stat(envDirpath); os.IsNotExist(err) {
		utils.PrintGreen("Created env directory \u2713")
		if err := os.Mkdir(envDirpath, 0755); err != nil {
			utils.PrintRed("Error creating env directory \u2717")
			return err
		}
	}
	if _, err := os.Stat(envFilePath); os.IsNotExist(err) {
		file, err := os.Create(envFilePath)
		if err != nil {
			utils.PrintRed("Error creating global environment \u2717")
			return err
		}
		defer file.Close()
		utils.PrintGreen(fmt.Sprintf("'%s.env' created \u2713", defaultEnv))
	}
	return nil
}

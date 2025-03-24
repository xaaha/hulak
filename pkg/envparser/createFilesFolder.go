package envparser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xaaha/hulak/pkg/utils"
)

// Creates an env directory and a fileName inside it.
// Returns envfilePath and errors associated with it
func CreateEnvDirAndFiles(fileName string) (string, error) {
	defEnvDir := utils.EnvironmentFolder
	defEnvSfx := utils.DefaultEnvFileSuffix

	envDirpath, err := utils.CreateFilePath(defEnvDir)
	if err != nil {
		utils.PrintRed("Error creating filePath")
		return "", err
	}

	envFilePath := filepath.Join(envDirpath, fileName+defEnvSfx)

	if _, err := os.Stat(envDirpath); os.IsNotExist(err) {
		if err := os.Mkdir(envDirpath, 0755); err != nil {
			utils.PrintRed("Error creating env directory \u2717")
			return "", err
		}
		utils.PrintGreen("Created env directory \u2713")
	}
	if _, err := os.Stat(envFilePath); os.IsNotExist(err) {
		file, err := os.Create(envFilePath)
		if err != nil {
			utils.PrintRed(fmt.Sprintf("Error creating %s environment \u2717", fileName))
			return "", err
		}
		defer file.Close()
		utils.PrintGreen(fmt.Sprintf("'%s%s' created \u2713", fileName, defEnvSfx))
	}

	return envFilePath, nil
}

/*
Creates environment folder and a default global.env file in it.
Optional: File names as a *string
*/
func CreateDefaultEnvs(envName *string) error {
	defEnv := utils.DefaultEnvVal

	if envName != nil {
		lowerCasedEnv := strings.ToLower(*envName)
		defEnv = lowerCasedEnv
	}
	_, err := CreateEnvDirAndFiles(defEnv)
	return err
}

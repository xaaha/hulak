// Package envparser contains environment parsing and functions around it
package envparser

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/xaaha/hulak/pkg/utils"
)

// CreateEnvDirAndFiles Creates an env directory and a fileName inside it.
// Returns envfilePath and errors associated with it
func CreateEnvDirAndFiles(fileName string) (string, error) {
	defEnvDir := utils.EnvironmentFolder
	defEnvSfx := utils.DefaultEnvFileSuffix

	envDirpath, err := utils.CreatePath(defEnvDir)
	if err != nil {
		utils.PrintRed("Error creating filePath")
		return "", err
	}

	envFilePath := filepath.Join(envDirpath, fileName+defEnvSfx)
	if err = utils.CreateDir(envDirpath); err != nil {
		return "", err
	}
	_, err = os.Stat(envFilePath)
	if os.IsNotExist(err) {
		if err = utils.CreateFile(envFilePath); err != nil {
			return "", err
		}
	}
	return envFilePath, nil
}

// CreateDefaultEnvs Creates environment folder and a default global.env file in it.
// Optional: File names as a *string
func CreateDefaultEnvs(envName *string) error {
	defEnv := utils.DefaultEnvVal

	if envName != nil {
		lowerCasedEnv := strings.ToLower(*envName)
		defEnv = lowerCasedEnv
	}
	_, err := CreateEnvDirAndFiles(defEnv)
	return err
}

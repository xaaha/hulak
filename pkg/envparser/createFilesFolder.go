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
	defEnv := utils.DefaultEnvVal
	defEnvDir := utils.EnvironmentFolder
	defEnvSfx := utils.DefaultEnvFileSuffix

	if envName != nil {
		lowerCasedEnv := strings.ToLower(*envName)
		defEnv = lowerCasedEnv
	}
	projectRoot, err := os.Getwd()
	if err != nil {
		utils.PrintRed("Error getting current working directory")
		return err
	}

	// create an env folder in the root of the project
	envDirpath := filepath.Join(projectRoot, defEnvDir)
	envFilePath := filepath.Join(envDirpath, defEnv+defEnvSfx) // global.env as the default
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
			utils.PrintRed(fmt.Sprintf("Error creating %s environment \u2717", defEnv))
			return err
		}
		defer file.Close()
		utils.PrintGreen(fmt.Sprintf("'%s%s' created \u2713", defEnv, defEnvSfx))
	}
	return nil
}

// not used yet,
func CreateFilesFolder(fileName string) error {
	defEnvDir := utils.EnvironmentFolder
	projectRoot, err := os.Getwd()
	defEnvSfx := utils.DefaultEnvFileSuffix

	if err != nil {
		utils.PrintRed("Error getting current working directory")
		return err
	}

	envDirpath := filepath.Join(projectRoot, defEnvDir)
	envFilePath := filepath.Join(envDirpath, fileName+defEnvSfx)
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
			utils.PrintRed(fmt.Sprintf("Error creating %s environment \u2717", fileName))
			return err
		}
		defer file.Close()
		utils.PrintGreen(fmt.Sprintf("'%s%s' created \u2713", fileName, defEnvSfx))
	}

	return nil
}

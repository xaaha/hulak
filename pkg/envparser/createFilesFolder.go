package envparser

import (
	"fmt"
	"os"
	"path/filepath"
)

/*
Creates environment folder and a default global.env file in it.
Optional: File names as a *string
*/
func CreateDefaultEnvs(envName *string) error {
	defaultEnv := "global"
	if envName != nil {
		defaultEnv = *envName
	}
	projectRoot, err := os.Getwd()
	if err != nil {
		return err
	}

	// create an env folder in the root of the project
	envDirpath := filepath.Join(projectRoot, "env")
	envFilePath := filepath.Join(envDirpath, defaultEnv+".env") // global.env as the default
	if _, err := os.Stat(envDirpath); os.IsNotExist(err) {
		fmt.Println("Created env directory \u2713")
		if err := os.Mkdir(envDirpath, 0755); err != nil {
			fmt.Println("Error creating env directory")
			return err
		}
	}
	if _, err := os.Stat(envFilePath); os.IsNotExist(err) {
		file, err := os.Create(envFilePath)
		if err != nil {
			fmt.Println("Error creating global environment \u2717")
			return err
		}
		defer file.Close()
		fmt.Println("Created", defaultEnv+".env", "\u2713")
	}
	return nil
}

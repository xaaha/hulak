package utils

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/xaaha/hulak/pkg/envparser"
)

// creates environment folder and a default global.env file in the folder
func createDefaultEnvs() error {
	projectRoot, err := os.Getwd()
	if err != nil {
		return err
	}
	// create a env folder in the root of the project
	envDirpath := filepath.Join(projectRoot, "env")
	envFilePath := filepath.Join(envDirpath, "global.env") // global.env as the default
	if _, err := os.Stat(envDirpath); os.IsNotExist(err) {
		fmt.Println("Creating env directory...")
		if err := os.Mkdir(envDirpath, 0755); err != nil {
			fmt.Println("Error creating env directory")
			return err
		}
	}
	if _, err := os.Stat(envFilePath); os.IsNotExist(err) {
		fmt.Println("Creating global.env for your secrets...")
		file, err := os.Create(envFilePath)
		if err != nil {
			fmt.Println("Error creating global environment \u2717")
			return err
		}
		defer file.Close()
		fmt.Println("Global env file created \u2713")
	}
	return nil
}

/*
InitializeProject() starts the project by creating envfolder and global file in it.
TBC...
*/
func InitializeProject() {
	err := createDefaultEnvs()
	if err != nil {
		panic(err)
	}
	err = envparser.SetDefaultEnv()
	if err != nil {
		panic(err)
	}
}

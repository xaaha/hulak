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

// TODO: write function to migrate the pm environemnt and pm collection
/*

package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// sample file for env

type EnvValues struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Enabled bool   `json:"enabled"`
}

type Environment struct {
	Name   string      `json:"name"`
	Values []EnvValues `json:"values"`
}

func main() {
	var env Environment
	jsonByteVal, err := os.ReadFile("./globals.json")
	if err != nil {
		fmt.Println("error occured while opening the json file", err)
	}
	err = json.Unmarshal(jsonByteVal, &env)
	if err != nil {
		fmt.Println("error occured while unmarshalling the file", err)
	}

	fmt.Println("Key = ", env.Values[0].Key)
	fmt.Println("Value \u2713 =", env.Values[0].Value)

	// hulak migrate "./globals.json"
  // only accept 1 argument and determine
  // what is it; if  postman envFile Run env otherwise run postman collection
  // automatically check whether it' postman file
  // postman has name that says where it's coming from


	// if name is empty ""  or name == "globals" then migrate things to global
	// otherwise a name in pm json file should create a new env file with the exact name if the env file does not exists
	// if the name in json exists in the env folder there is no need to create it, just migrate
	// Existing function to create folder and file for the environment
	// If it's globals then push this into global.env
	// Otherwise just create then same environment as the name
}
*/

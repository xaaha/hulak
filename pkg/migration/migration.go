// Package migration migrates colelction, variables, responses to hulak Currently it only supports postman collection and variablespackage migration
package migration

import (
	"fmt"

	"github.com/xaaha/hulak/pkg/utils"
)

// CompleteMigration processes all files for migration
func CompleteMigration(filePaths []string) error {
	if len(filePaths) == 0 {
		return utils.ColorError("please provide a valid json file for migration")
	}
	for _, path := range filePaths {
		jsonStr, err := readJSON(path)
		if err != nil {
			return err
		}

		if IsEnv(jsonStr) {
			env, err := PrepareEnvStruct(jsonStr)
			if err != nil {
				return fmt.Errorf("error converting to Environment: %w", err)
			}

			err = migrateEnv(env)
			if err != nil {
				return utils.ColorError("error migrating environment: %w", err)
			}
			utils.PrintGreen(fmt.Sprintf("migrated '%s': ", path))
		} else if isCollection(jsonStr) {
			err := migrateCollection(jsonStr)
			utils.PrintGreen(fmt.Sprintf("migrated '%s': ", path))
			if err != nil {
				utils.PrintWarning("Collection migration did not work for: " + path)
				return err
			}
		} else {
			utils.PrintWarning("Unknown Postman file format: " + path)
		}
	}

	return nil
}

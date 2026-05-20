// Package migration migrates colelction, variables, responses to hulak Currently it only supports postman collection and variablespackage migration
package migration

import (
	"errors"
	"fmt"

	"github.com/xaaha/hulak/pkg/utils"
)

// CompleteMigration processes all files for migration
func CompleteMigration(filePaths []string) error {
	if len(filePaths) == 0 {
		return errors.New("please provide a valid json file for migration")
	}
	for _, path := range filePaths {
		jsonStr, err := readJSON(path)
		if err != nil {
			return err
		}

		switch {
		case IsEnv(jsonStr):
			env, err := PrepareEnvStruct(jsonStr)
			if err != nil {
				return fmt.Errorf("error converting to Environment: %w", err)
			}

			err = migrateEnv(env)
			if err != nil {
				return fmt.Errorf("error migrating environment: %w", err)
			}
			utils.PrintSuccessStderr(fmt.Sprintf("migrated '%s'", path))
		case isCollection(jsonStr):
			err := migrateCollection(jsonStr)
			if err != nil {
				return fmt.Errorf("collection migration failed for %s: %w", path, err)
			}
			utils.PrintSuccessStderr(fmt.Sprintf("migrated '%s'", path))
		default:
			utils.PrintWarningStderr("Unknown Postman file format: " + path)
		}
	}

	return nil
}

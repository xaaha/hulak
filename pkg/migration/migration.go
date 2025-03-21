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
		jsonStr, err := ReadPmFile(path)
		if err != nil {
			return err
		}

		if IsEnv(jsonStr) {
			env, err := PrepareEnvStruct(jsonStr)
			if err != nil {
				return fmt.Errorf("error converting to Environment: %w", err)
			}

			err = MigrateEnv(env)
			if err != nil {
				return utils.ColorError("error migrating environment: %w", err)
			}
		} else if IsCollection(jsonStr) {
			// Future implementation for collection migration
			utils.PrintWarning("Collection migration coming soon for file: " + path)
		} else {
			utils.PrintWarning("Unknown Postman file format: " + path)
		}
	}

	return nil
}

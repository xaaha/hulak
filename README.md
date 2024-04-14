# Hulak

# Construction Work 🏗️

## flags

### `-env`

- **Description**: Specifies the environment configuration to be used during the program's execution. This flag accepts the name of an environment file, excluding the path to the `env` folder where the environment files are stored.
- **Arguments**: Only one argument is accepted for this flag. If multiple arguments are provided, only the first one is considered.
- **Default Behavior**: By default, the environment is set up to utilize the `global` configuration if no other environment file is specified.
- **File Existence**: The specified environment file should exist within the `env` folder. If the file does not exist, the user will be prompted to create a new environment file in the specified folder.
- **Priority**: User-created environment files take precedence over global settings. If there are any overlapping variables between the global environment and the user-specified environment, the values from the user's environment file will override the global values.

### To do

- [x] Unit Testing.
  - Ongoing: parser_test
- [ ] Remove the env value if the file creation is skipped.

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

- [ ] Strip special character besides \_ or - from the env file name. On `setEnvFile Line 77` and get `env file line 36 `
- [ ] Unit Testing

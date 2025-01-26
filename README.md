# Hulak

# Construction Work 🏗️

## Flags

| Flag   | Description                                                                                                                                                                                           | Usage                            |
| ------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------- |
| `-env` | Specify the environment file you want to use for Api Call. If the user flag is absent, it defaults to `global`.                                                                                       | `-env prod`                      |
| `-fp`  | Represents file-path for the file/directory you want to run. (Directory run is coming soon)                                                                                                           | -fp "./collection/getUsers.yaml" |
| `-f`   | Yaml/yml file to run. Hulak recurses though your directories and subdirectories, excluding hidden, from the root and finds the matching file(s). If multiple matches are found, they run concurrently | `-f graphql`                     |

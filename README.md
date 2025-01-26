# Hulak

File based, user friendly API client for the terminal.

# Construction Work 🏗️

## Flags

| Flag   | Description                                                                                                                                                                                           | Usage                            |
| ------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------- |
| `-env` | Specify the environment file you want to use for Api Call. If the user flag is absent, it defaults to `global`.                                                                                       | `-env prod`                      |
| `-fp`  | Represents file-path for the file/directory you want to run. (Directory run is coming soon)                                                                                                           | -fp "./collection/getUsers.yaml" |
| `-f`   | Yaml/yml file to run. Hulak recurses though your directories and subdirectories, excluding hidden, from the root and finds the matching file(s). If multiple matches are found, they run concurrently | `-f graphql`                     |

## Actions

Actions make it easier to retrieve values from other files.

### `.Key`

Grab the key's value from the defined environemnt files in the project's root. Key is case sensitive. So, `.key` and `.Key` are two different values.

```bash
# example directory structure
env/
  global.env    # default and required
  prod.env      # user defined, could be anything
  staging.env   # user defined
collection/
    test.yaml   # file
```

If the call is made with `-env prod`, then `.Key` should be defined in either `global` or `prod` file. Similarly, if
the call is made with `-env staging`, then the `.Key` should be present in either `global` or `staging` file.
If the `.Key` is defined in both files, `global` is ignored.

### `getValueOf`:

`getValueOf` looks for the value of the key inside the `file_name_response.json` file.
You don't need to provide the full `file_name_response.json` name. Hulak recurces through the directory and uses the first file match.
It's recommended that you use a unique name for each file.

- `"key"` and `"file_name"`: Should be surrounded by double quotes (Go template).

```yml
url: `{{getValueOf "key" "file_name"}}`
```

OR

```yaml
url: '{{getValueOf "key" "file_name"}}'
```

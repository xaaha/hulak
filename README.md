<p align="center">
  <img alt="Hulak Logo" src="./assets/logo.svg" height="140" />
  <p align="center">File based API client for terminal nerds.</p>
</p>

> [!Warning]
> ⚠️ This project is actively in development and testing. Expect rapid changes and plenty of bugs! 🚧

# Elevator Pitch

If you’ve ever wanted to manage your API workflows like a code repository — easily searching, editing, copying, and deleting request files and variables, `hulak` is the tool for you. Hulak is a fast, lightweight, file-based API client that lets you make API calls and organize requests and responses using YAML files.

```yaml
# ────────────────────────────────────────────────────
# Example: test_gql.yaml
# ────────────────────────────────────────────────────
---
method: POST

# 🚨 Keep secrets separate! Avoid hardcoding credentials.
url: "{{.graphqlUrl}}"
headers:
  Content-Type: application/json
  # 🔍 Dynamically access nested values from another file
  # using the `getValueOf` action.
  Authorization: Bearer {{getValueOf "data.access_token" "employer_auth.json"}}
body:
  graphql:
    # 📂 Store large JSON, GraphQL, XML, or HTML files separately
    # and access them using the `getFile` action.
    query: '{{getFile "e2etests/test_collection/test.graphql"}}'
    variables:
      # 🏷️ Use templating to dynamically construct values.
      name: "{{.userName}} of age {{.userAge}}"
      age: "{{.userAge}}"
```

```bash
# Run the file using secrets from staging.env file
hulak -env staging -f test_gql
```

# Getting Started

## Installation

### 1. Homebrew

```bash
brew install xaaha/tap/hulak
```

### 2. `go install`

- Run

```bash
go install github.com/xaaha/hulak@latest
```

- You need to install `go` in your system
- In order for any utility, installed with `go install`, to be available for use, you need the path from `go env GOPATH` to be in the shell’s PATH.

  • If it’s not, add the following to your shell's configuration file.

```bash
export GOPATH=$HOME/go
export PATH=$PATH:$(go env GOPATH)/bin
```

- Then source your shell configuration file `source ~/.zshrc` or `source ~/.bashrc`

### 3. Build from source

- Clone the repo
- Install required dependencies: Run `go mod tidy` in the root of the project
- Build the executable with: `go build -o hulak`
- Move the executable to the path.

  - On Mac/Linux: `sudo mv hulak /usr/local/bin/`
    - Verify the project exists in path with: `which hulak`
  - On Windows:

    - Move the `hulak.exe` binary to a folder that is in your PATH. A common location for this is `C:\Go\bin` (or another directory you've added to your PATH).
    - To add a folder to your PATH in Windows:
      Go to `Control Panel > System and Security > System > Advanced system settings`.
      Click `Environment Variables`.
      Under `System variables`, find the `Path ` variable and click `Edit`.
      Add the path to your folder (e.g., `C:\Go\bin`) and click `OK`.

## Verify Installation with

```bash
hulak help
```

---

## Initialize environment directory to store secrets

Hulak uses `env` directory to store secrets (e.g., passwords, client IDs) used in API call. It allows separation between different environments like local, test, and production environments.

### Setup

Create the `env/global.env` in the root of the hulak project by running

```bash

hulak init
# to create multiple .env files in the env directory run
hulak init -env staging prod
```

You can store all secrets in `global.env`, but for running tests with different credentials, use additional `<custom_file_name>.env` files like `staging.env` or `prod.env`.

If `env/global.env` is absent, it will prompt you to create one at runtime. For more details read this [environment documentation](./docs/environment.md).

```bash
# example directory structure
env/
  global.env    # default and required
  prod.env      # user defined, could be anything
  staging.env   # user defined
collection/
    test.yaml   # api file
```

As seen above, in a location of your choice, create a directory called `env` and put `global.env` file inside it. Global is the default and required environment. You can put all your secrets here, but in order to run the same test with multiple secrets, you would need other `.env` files like `staging` or `prod` as well.

## Create An API file

Then Basic API call looks like `test.yaml` below. See full documentation on Request Body structure [here](./docs/body.md). More request examples are [here](https://github.com/xaaha/hulak/tree/main/e2etests/test_collection).

```yaml
# test.yaml
method: Get
url: https://jsonplaceholder.typicode.com/todos/1
```

Run the file with

```bash
hulak -env global -f test
# or
hulak -env global -fp ./test.yaml
```

Since global is default environment, we don't need to specify `-env global`. So, this is the simplest way of running the file.

```bash
hulak -f test
```

File's response is be printed in the console and also saved at the same location as the calling file with `_response.json` suffix.
Read more about response in [response documentation](./docs/response.md).

```json
{
  "body": {
    "completed": false,
    "id": 1,
    "title": "delectus aut autem",
    "userId": 1
  },
  "status": "200 OK"
}
```

# Flags and Subcommands

## Flags

| Flag   | Description                                                                                                                                                                                                | Usage                            |
| ------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------- |
| `-env` | Specify the environment file you want to use for Api Call. If the user flag is absent, it defaults to `global`.                                                                                            | `-env prod`                      |
| `-fp`  | Represents file-path for the file/directory you want to run. (Directory run is coming soon)                                                                                                                | -fp "./collection/getUsers.yaml" |
| `-f`   | yaml/yml file to run. Hulak recurses though your directories and subdirectories, excluding hidden, from the root and finds the matching yaml file(s). If multiple matches are found, they run concurrently | `-f graphql`                     |

## Subcommands

| Subcommand | Description                                                              | Usage                                                               |
| ---------- | ------------------------------------------------------------------------ | ------------------------------------------------------------------- |
| help       | display help message                                                     | `hulak help`                                                        |
| init       | Initialize environment directory and files in it                         | `hulak init` or ` hulak init -env global prod staging`              |
| migrate    | migrates postman environment and collection (v2.1 only) files for hulak. | `hulak migrate "path/to/environment.json" "path/to/collection.json` |

# Actions

Actions make it easier to retrieve values from other files. See, [actions documentation](./docs/body.md) for more detailed explanation.

### `.Key`

```yaml
# example section
body:
  graphql:
    query: |
      query Hello($name: String!, $age: Int) {
        hello(person: { name: $name, age: $age })
      }
    variables:
      name: "{{.userName}} of age {{.userAge}}"
      age: "{{.userAge}}"
```

`.Key` is a variable, that is present in one of the `.env` files. It grabs the value from environemnt files in the `env/` directory in the root of the project [created above](#Initialize-Environment-Folders). The value of, `Key` is replaced during runtime.
In the example above, `.userName` and `.userAge` are examples of retrieving key from secrets stored in `env/`.

### `getValueOf`:

```yaml
# example
url: `{{getValueOf "key" "file_name" }}`
```

`getValueOf` looks for the value of the `key` inside the `file_name.json` file. Since responses of the api requests are saved in `file_name_response.json` file in the same directory, you don't need to provide `_response.json` suffix when using `getValueOf`.
If multiple `file_name.json` is found, hulak recurces through the directory and uses the first file match. So, it's recommended that you use a unique name for each file.
You can also provide the exact file location instead of `file_name` as `./e2etests/test_collection/graphql_response.json`

- `"key"` and `"file_name"`: Should be surrounded by double quotes (Go template).
- `key` you are looking for could in a nested object as well. For example, `user.name` means give me the name inside the user's object. You can esacpe the dot (.) with single curly brace like `{user.name}`. Here, `user.name` is considered a `key`.
- `file_name` could be the only file name or the entire file path from project root. If only name is provided, first match will be used.

```yaml
# name is inside the user object in the user.json file
name: '{{getValueOf "user.name" "user.json"}}'
# extract the value of name from nested object from provided json file path
name: '{{getValueOf "data.users[0].name" "./e2etests/test_collection/graphql_response.json"}}'
# where name is the key in the file
name: `{{getValueOf "name" "user.json"}}`
```

### `getFile`

Gets the file content as string and dumps the entire file content in context. It takes file path as an argument. Do not use `getFile` action to pass token in auth header.

```yaml
# example
body:
  graphql:
    query: '{{getFile "e2etests/test_collection/test.graphql"}}'
```

Learn more about these actions [here](./docs/actions.md)

# Auth2.0 (Beta)

Hualk supports auth2.0 web-application-flow. Follow the auth2.0 provider instruction to set it up. Read more [here](./docs/auth20.md)

# Planned Features

[See Features and Fixes Milestone](https://github.com/xaaha/hulak/milestone/3) to see all the upcoming, exciting features

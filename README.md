<p align="center">
  <img alt="Hulak Logo" src="./assets/logo.svg" height="140" />
  <p align="center">File based API client for terminal nerds.</p>
</p>

# Elevator Pitch

If you’ve ever wanted to manage your API workflows like a code repository — easily searching, editing, copying, and deleting request files and variables, `hulak` is the tool for you. Hulak is a fast, lightweight, file-based API client that lets you make API calls and organize requests and responses using YAML files.

```yaml
# ────────────────────────────────────────────────────
# Example: test_gql.hk.yaml
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

# Table of Contents

- [Elevator Pitch](#elevator-pitch)
- [Getting Started](#getting-started)
  - [Installation](#installation)
    - [1. Homebrew](#1-homebrew)
    - [2. go install](#2-go-install)
    - [3. Build from source](#3-build-from-source)
  - [Verify Installation with](#verify-installation-with)
  - [Initialize Project](#initialize-project)
  - [Create An API file](#create-an-api-file)
- [Flags and Subcommands](#flags-and-subcommands)
  - [Flags](#flags)
  - [Subcommands](#subcommands)
- [Schema](#schema)
- [Actions](#actions)
  - [.Key](#key)
  - [getValueOf](#getvalueof)
  - [getFile](#getfile)
- [Auth2.0 (Beta)](#auth20-beta)
- [Planned Features](#planned-features)
- [Support the Project](#support-the-project)

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
hulak version
# or
hulak help
```

---

## Initialize Project

Create a project directory and cd into it. Then Initialize the project

```bash
mkdir my_apis & cd my_apis
hulak init

```

Hulak uses `env` directory to store secrets (e.g., passwords, client IDs) used in API call. It allows separation between different environments like local, test, and production environments. The `hulak init` command above sets up the secrets directory structure `env/` and also provides an `apiOptions.yaml` file for your reference.

```bash
# to create multiple .env files in the env directory run
hulak init -env staging prod
```

You can store all secrets in `global.env`, but for running tests with different credentials, use additional `<custom_file_name>.env` files like `staging.env` or `prod.env`.

If `env/global.env` is absent, it will prompt you to create one at runtime. For more details read this [environment documentation](./docs/environment.md).

```bash
# example directory structure
env/
  global.env    # default and required created with hulak init
  prod.env      # user defined, could be anything
  staging.env   # user defined
collection/     # example directory
    test.yaml   # example api file
```

### Using OS environment variables

As you are using the `.env` file as a means to store the variables and secrets used to make the request, it's understandable that you may not want to store your secrets already in your system's OS environment into a file which you may accidentally push to your repository. To reduce the possibility of this occurring, you can set the variables to use values sourced from yoursystem's OS environment variable by using the `$` prefix for the value. 

For example, if you had an environment variable `USER=foo` set on your system, and the following was in your `<custom_file_name>.env` file.

```
exampleVar = $USER
```

Using `{{.exampleVar}}` within a request file, i.e. 

```yaml
# test.yaml
method: Get
url: http://some.api.com/tests?bar={{.exampleVar}}
```

would result in the request targeting `http://some.api.com/tests?bar=foo`


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
hulak -env global -fp test.yaml
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

| Flag      | Description                                                                                                                                                                                                                                                                                                                                                            | Usage                            |
| --------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------- |
| `-env`    | Specify the environment file you want to use for Api Call. If the user flag is absent, it defaults to `global`.                                                                                                                                                                                                                                                        | `-env prod`                      |
| `-fp`     | Represents file-path for the file/directory you want to run.                                                                                                                                                                                                                                                                                                           | -fp "./collection/getUsers.yaml" |
| `-f`      | File name (yaml/yml) to run. Hulak searches your directories and subdirectories from the root and finds the matching yaml file(s). If multiple matches are found, they run concurrently                                                                                                                                                                                | `-f graphql`                     |
| `-debug`  | Add debug boolean flag to get the entire request, response, headers, and TLS info about the api request                                                                                                                                                                                                                                                                | `-debug`                         |
| `-dir`    | Run entire directory concurrently. Only supports (.yaml or .yam) file. All files use the same provided environment                                                                                                                                                                                                                                                     | `-dir path/to/directory/`        |
| `-dirseq` | Run entire directory one file at a time. Only supports (.yaml or .yam) file. All files use the same provided environment. In nested directory, it is not guranteed that files will run as they appear in the file system. If the order matter, it's recommended to have a directory without nested directories inside it, in which case, files will run alphabetically | `-dirseq path/to/directory/`     |

## Subcommands

| Subcommand | Description                                                              | Usage                                                               |
| ---------- | ------------------------------------------------------------------------ | ------------------------------------------------------------------- |
| help       | display help message                                                     | `hulak help`                                                        |
| init       | Initialize environment directory and files in it                         | `hulak init` or ` hulak init -env global prod staging`              |
| migrate    | migrates postman environment and collection (v2.1 only) files for hulak. | `hulak migrate "path/to/environment.json" "path/to/collection.json` |

# Schema

To enable auto-completion for Hulak YAML files, you have following options:

> **Note:** You need a YAML language server for any of these options to work.

## Option 1: Declare Schema in the File

You can declare the schema at the top of your YAML file. This can either be a local schema or a schema referenced by a URL. Here are two examples:

### Local Schema

```yaml
# yaml-language-server: $schema=../../assets/schema.json
---
```

OR

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/xaaha/hulak/refs/heads/main/assets/schema.json
---
```

## Option 2: Configure Your Editor

Alternatively, you can configure your editor to enable auto-completion without needing to declare the schema in each file. For Neovim users, you can find my configuration [here](https://github.com/xaaha/dev-env/blob/7d25456e59a3a73081baedfd9060810afa4332e4/nvim/.config/nvim/lua/pratik/plugins/lsp/lspconfig.lua).
Once configured, you can simply rename your file to `yourFile.hk.yaml` to benefit from auto-completion.

## Option 3: Schema Store

A request to add the schema to the Schema Store is currently pending. For updates, please refer to the issue on GitHub: [SchemaStore Issue #4645](https://github.com/SchemaStore/schemastore/issues/4645).

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
name: '{{getValueOf "data.users[0].name" "e2etests/test_collection/graphql_response.json"}}'
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

# Support the Project

If you enjoy the project, please consider supporting it by reporting a bug, suggesting a feature request, or sponsoring the project. Your pull request contributions are also welcome—feel free to open an issue indicating your interest in tackling a bug or implementing a new feature.

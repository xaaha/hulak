<p align="center">
  <strong>
    From quick single file runs to a rich GraphQL TUI, Hulak scales and lets you manage API workflows like code.
  </strong>
</p>

### Dedicated GraphQL Explorer

<img alt="GraphQL Explorer" src="./assets/gql.gif" width="720" />

Designed for developers who explore GraphQL at scale. Browse schemas from multiple endpoints, search operations, build queries interactively, execute inline, and save results, responses, all from the terminal.

### Fast Interactive Runner

<img alt="Interactive Runner" src="./assets/fp.gif" width="720" />

Just type `hulak` → find your request file → pick an environment → get your response.

### Quick Install

```bash
brew install xaaha/tap/hulak
```

---

# Table of Contents

- [Getting Started](#getting-started)
  - [Installation](#installation)
    - [1. Homebrew](#1-homebrew)
    - [2. go install](#2-go-install)
    - [3. Build from source](#3-build-from-source)
  - [Verify Installation](#verify-installation)
  - [Initialize Project](#initialize-project)
  - [Create An API File](#create-an-api-file)
- [GraphQL Explorer](#graphql-explorer-1)
- [Actions](#actions)
  - [.Key](#key)
  - [getValueOf](#getvalueof)
  - [getFile](#getfile)
- [Flags and Subcommands](#flags-and-subcommands)
  - [Flags](#flags)
  - [Subcommands](#subcommands)
- [Schema](#schema)
- [Auth2.0 (Beta)](#auth20-beta)
- [Documentation](#documentation)
- [Planned Features](#planned-features)
- [Contributing](#contributing)
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
- In order for any utility, installed with `go install`, to be available for use, you need the path from `go env GOPATH` to be in the shell's PATH.

  • If it's not, add the following to your shell's configuration file.

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

## Verify Installation

```bash
hulak version
# or
hulak help
```

---

## Initialize Project

Create a project directory and cd into it. Then initialize the project

```bash
mkdir my_apis & cd my_apis
hulak init
```

Hulak uses `env` directory to store secrets (e.g., passwords, client IDs) used in API calls that reference environment template vars like `{{.key}}`. It allows separation between different environments like local, test, and production. The `hulak init` command above sets up the secrets directory structure `env/` and also provides an `apiOptions.hk.yaml` file for your reference.

```bash
# to create multiple .env files in the env directory run
hulak init -env staging prod
```

You can store all secrets in `global.env`, but for running tests with different credentials, use additional `<custom_file_name>.env` files like `staging.env` or `prod.env`.

If a selected request needs `{{.key}}` values and `env/global.env` is absent, Hulak will prompt you to create the project setup at runtime. Requests without environment template vars can run without `env/`. For more details read this [environment documentation](./docs/environment.md).

```bash
# example directory structure
env/
  global.env    # default env file used when environment vars are needed
  prod.env      # user defined, could be anything
  staging.env   # user defined
collection/     # example directory
    test.yaml   # example api file
```

### Using OS environment variables

If you use a `.env` file to store secrets, you might not want to duplicate secrets already stored in your system environment (for example, your shell). To avoid this, you can reference system environment variables in your `.env` file by using the `$` prefix.

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

## Create An API File

A basic API call looks like `test.yaml` below. See full documentation on Request Body structure [here](./docs/body.md). More request examples are [here](https://github.com/xaaha/hulak/tree/main/e2etests/test_collection).

```yaml
# test.yaml
method: Get
url: https://jsonplaceholder.typicode.com/todos/1
```

Run the file with

```bash
hulak -f test
```

Since global is default environment, we don't need to specify `-env global`. If the matched files do not contain environment template vars (`{{.key}}`), Hulak runs them without requiring `env/`.

File's response is printed in the console and also saved at the same location as the calling file with `_response.json` suffix.
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

Here's a more advanced example using templates, actions, and GraphQL:

```yaml
# ────────────────────────────────────────────────────
# Example: test_gql.hk.yaml
# ────────────────────────────────────────────────────
---
method: POST
url: "{{.graphqlUrl}}"
headers:
  Content-Type: application/json
  Authorization: Bearer {{getValueOf "data.access_token" "employer_auth.json"}}
body:
  graphql:
    query: '{{getFile "e2etests/test_collection/test.graphql"}}'
    variables:
      name: "{{.userName}} of age {{.userAge}}"
      age: "{{.userAge}}"
```

```bash
# Run the file using secrets from staging.env file
hulak -env staging -f test_gql
```

# GraphQL Explorer

Hulak has a dedicated GraphQL explorer for schema discovery, operation search, query building, validation, and saving generated files.

Use it when you know a query or mutation exists somewhere but do not remember which endpoint or file owns it.

```bash
# Explore one GraphQL source file
hulak gql e2etests/gql_schemas/countries.yml

# Explore all GraphQL source files in the current directory
hulak gql .

# Skip the environment selector
hulak gql -env staging ./collections/graphql
```

Directory mode auto-detects GraphQL source files by looking for `kind: GraphQL` with a non-empty `url`. Single-file mode only requires a valid file with a non-empty `url`.

The explorer gives you:

- fast operation search
- type filters with `q:`, `m:`, and `s:`
- endpoint filtering with `e:`
- interactive argument and field selection
- generated query and variables panels
- inline query execution
- response inspection and saving
- generated `.gql` and `.hk.yaml` request files

Read the full guide in [docs/graphql-explorer.md](./docs/graphql-explorer.md).

# Actions

Actions make it easier to retrieve values from other files. See [actions documentation](./docs/actions.md) for more detailed explanation.

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

`.Key` is a variable, that is present in one of the `.env` files. It grabs the value from environment files in the `env/` directory in the root of the project [created above](#initialize-project). The value of `Key` is replaced during runtime.
In the example above, `.userName` and `.userAge` are examples of retrieving key from secrets stored in `env/`.

### `getValueOf`

```yaml
# example
url: `{{getValueOf "key" "file_name" }}`
```

`getValueOf` looks for the value of the `key` inside the `file_name.json` file. Since responses of the api requests are saved in `file_name_response.json` file in the same directory, you don't need to provide `_response.json` suffix when using `getValueOf`.
If multiple `file_name.json` is found, hulak recurses through the directory and uses the first file match. So, it's recommended that you use a unique name for each file.
You can also provide the exact file location instead of `file_name` as `./e2etests/test_collection/graphql_response.json`

- `"key"` and `"file_name"`: Should be surrounded by double quotes (Go template).
- `key` you are looking for could be in a nested object as well. For example, `user.name` means give me the name inside the user's object. You can escape the dot (.) with single curly brace like `{user.name}`. Here, `user.name` is considered a `key`.
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

# Flags and Subcommands

## Flags

| Flag      | Description                                                                                                                                                                                                                                                                                                                                                              | Usage                            |
| --------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | -------------------------------- |
| `-env`    | Specify the environment file you want to use for Api Call. If the user flag is absent, it defaults to `global`.                                                                                                                                                                                                                                                          | `-env prod`                      |
| `-fp`     | Represents file-path for the file/directory you want to run.                                                                                                                                                                                                                                                                                                             | -fp "./collection/getUsers.yaml" |
| `-f`      | File name (yaml/yml) to run. Hulak searches your directories and subdirectories from the root and finds the matching yaml file(s). If multiple matches are found, they run concurrently                                                                                                                                                                                  | `-f graphql`                     |
| `-debug`  | Add debug boolean flag to get the entire request, response, headers, and TLS info about the api request                                                                                                                                                                                                                                                                  | `-debug`                         |
| `-dir`    | Run entire directory concurrently. Only supports (.yaml or .yam) file. All files use the same provided environment                                                                                                                                                                                                                                                       | `-dir path/to/directory/`        |
| `-dirseq` | Run entire directory one file at a time. Only supports (.yaml or .yam) file. All files use the same provided environment. In nested directory, it is not guaranteed that files will run as they appear in the file system. If the order matters, it's recommended to have a directory without nested directories inside it, in which case, files will run alphabetically | `-dirseq path/to/directory/`     |

Interactive mode (`hulak` with no file/directory flags) picks the request file first. If the selected file requires `{{.key}}`, Hulak then asks for environment selection. During slow file discovery, a spinner appears after a short delay.

## Subcommands

| Subcommand | Description                                                              | Usage                                                                |
| ---------- | ------------------------------------------------------------------------ | -------------------------------------------------------------------- |
| help       | display help message                                                     | `hulak help`                                                         |
| init       | Initialize environment directory and files in it                         | `hulak init` or `hulak init -env global prod staging`                |
| migrate    | migrates postman environment and collection (v2.1 only) files for hulak. | `hulak migrate "path/to/environment.json" "path/to/collection.json"` |
| gql        | open the GraphQL explorer for one file or a directory                    | `hulak gql .` or `hulak gql -env staging path/to/graphql`            |

# Schema

To enable auto-completion for Hulak YAML files, you have the following options:

> **Note:** You need a YAML language server for any of these options to work.

## Option 1: Schema Store (Recommended)

The Hulak schema is now available in the [Schema Store](https://www.schemastore.org/). If your editor supports Schema Store (most do, like VS Code and Neovim with `yaml-language-server`), auto-completion will work automatically for files ending in `.hk.yaml` or `.hk.yml`.
If Schema Store is not set up in your editor, use **Option 2** or **Option 3** below.

## Option 2: Declare Schema in the File

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

## Option 3: Configure Your Editor

Alternatively, you can configure your editor to enable auto-completion without needing to declare the schema in each file. For Neovim users, you can find my configuration [here](https://github.com/xaaha/dev-env/blob/7d25456e59a3a73081baedfd9060810afa4332e4/nvim/.config/nvim/lua/pratik/plugins/lsp/lspconfig.lua).
Once configured, you can simply rename your file to `yourFile.hk.yaml` for auto-completion.

# Auth2.0 (Beta)

Hulak supports auth2.0 web-application-flow. Follow the auth2.0 provider instruction to set it up. Read more [here](./docs/auth20.md)

# Documentation

For deeper docs, start here:

- [GraphQL Explorer](./docs/graphql-explorer.md)
- [Request Body](./docs/body.md)
- [Actions](./docs/actions.md)
- [Environment Secrets](./docs/environment.md)
- [Response Files](./docs/response.md)
- [Auth2.0](./docs/auth20.md)

# Planned Features

[See Features and Fixes Milestone](https://github.com/xaaha/hulak/milestone/3) to see all the upcoming, exciting features

# Contributing

```bash
git clone https://github.com/xaaha/hulak.git
cd hulak
mise install
```

See **[CONTRIBUTING.md](./CONTRIBUTING.md)** for the full guide covering development workflow.

# Support the Project

If you enjoy the project, please consider supporting it by reporting a bug, suggesting a feature request, or sponsoring the project. Your pull request contributions are also welcome. Feel free to open an issue indicating your interest in tackling a bug or implementing a new feature.

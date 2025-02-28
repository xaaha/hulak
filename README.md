<p align="center">
  <img alt="Hulak Logo" src="./assets/logo.svg" height="140" />
  <!-- <h2 align="center">Hulak</h2> -->
  <p align="center">User friendly API Client for terminal nerds.</p>
</p>

## Installation

### 1. Recommended `go install`

- Run

```bash
go install github.com/xaaha/hulak@latest
```

- In order for any utility, installed with `go install`, to be available for use, you need the path from `go env GOPATH` to be in the shell’s PATH.

  • If it’s not, add the following to your shell's configuration file.

```bash
export GOPATH=$HOME/go
export PATH=$PATH:$(go env GOPATH)/bin
```

### 2. Homebrew

Hulak is not yet available as a Homebrew formula due to its early-stage development, see this [section](https://docs.brew.sh/Acceptable-Formulae#niche-or-self-submitted-stuff). A Homebrew tap will be added in the future.

TODO: with go releaser

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

---

## Initialize Environment Folders

Hulak uses `env` directory to store secrets (e.g., passwords, client IDs) used in API call. It allows separation between local, test, and production environments.

### Setup

Create the env folder and the required `global.env` file in the root of the hulak project.

```bash
mkdir -p env && touch env/global.env
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

## Getting Started

Then Basic API call looks like `test.yaml`

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

Since global is default environemnt, we don't need to specify `-env global`. So, this is the simplest way of running the file.

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

## Flags

| Flag   | Description                                                                                                                                                                                           | Usage                            |
| ------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------- |
| `-env` | Specify the environment file you want to use for Api Call. If the user flag is absent, it defaults to `global`.                                                                                       | `-env prod`                      |
| `-fp`  | Represents file-path for the file/directory you want to run. (Directory run is coming soon)                                                                                                           | -fp "./collection/getUsers.yaml" |
| `-f`   | yaml/yml file to run. Hulak recurses though your directories and subdirectories, excluding hidden, from the root and finds the matching file(s). If multiple matches are found, they run concurrently | `-f graphql`                     |

## Actions

Actions make it easier to retrieve values from other files. See, [actions documentation](./docs/body.md) for more detailed explanation.

### `.Key`

`.Key` is a variable, that is present in one of the `.env` files. It grabs the value from a environemnt file in the `env/` directory in the root of the project [created above](#Initialize-Environment-Folders). The value of, `Key` is replaced during runtime.

### `getValueOf`:

```yaml
# example
url: `{{getValueOf "key" "file_name" }}`
```

`getValueOf` looks for the value of the `key` inside the `file_name.json` file. Since responses of the api requests are saved in `file_name_response.json` file in the same directory, you don't need to provide `_response.json` suffix when using `getValueOf`.
If multiple `file_name.json` is found, hulak recurces through the directory and uses the first file match. SO, it's recommended that you use a unique name for each file.
You can also provide the exact file location instead of `file_name` as `./e2etests/test_collection/graphql_response.json`

- `"key"` and `"file_name"`: Should be surrounded by double quotes (Go template).
- `key` you are looking for could in a nested example as well. For example, `user.name` means give me the name inside the user's object. You can esacpe the dot (.) with single curly brace like `{user.name}`. Here, `user.name` is considered a `key`.

```yaml
# name is inside the user object in the user.json file
name: '{{getValueOf "user.name" "user.json"}}'
# providing full path
name: '{{getValueOf "data.users[0].name" "./e2etests/test_collection/graphql_response.json"}}'
# where name is the key in the file
name: `{{getValueOf "name" "user.json"}}`
```

### Auth2.0 (Beta)

Hualk supports auth2.0 web-application-flow. Follow the auth2.0 provider instruction to set it up. Below is the example for [github](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps#web-application-flow).

```yaml
# kind is required field in hulak. For api files, kind is not required
kind: auth # value can be auth or api.
method: POST
# url that opens up in a browser. Usually ends with /authorize
url: https://github.com/login/oauth/authorize
urlparams:
  client_id: "{{.client_id}}"
  scope: repo:user
auth:
  type: OAuth2.0
  # url to retrieve access token after broswer authorization
  access_token_url: https://github.com/login/oauth/access_token
# Use appropriate headers as instructed by the auth2.0 provider, github in this case
headers:
  Accept: application/json
body:
  urlencodedformdata:
    client_secret: "{{.client_secret}}"
    client_id: "{{.client_id}}"
# code retrieved from browser is automatically inserted
```

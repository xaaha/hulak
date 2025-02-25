<p align="center">
  <img alt="Hulak Logo" src="./assets/logo.png" height="140" />
  <h2 align="center">Hulak</h2>
  <p align="center">User friendly API Client for terminal nerds.</p>
</p>

---

# Construction Work 🏗️

## Installation

Any of the following installation step work

### 1. Recommended `go install`

- Run

```shell
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

## Initialize Environment folders

We often need secrets, like passwords, client ids and secrets, and other sensitive information to make an api call. Those secrets are often seperate between your production and test and local environment. Hulak manages those secrets in a folder. The structure of the folder looks like the following.

```bash
env/
  global.env # required
  staging.env
  prod.env
collection/
  second_api.yaml
```

As seen above, in a location of your choice, create a directory called `env` and put `global.env` file inside it. Global is the default and required environment. You can put all your secrets here, but in order to run the same test with multiple secrets, you would need other `.env` files like `staging` or `prod` as well.

Create folder

```bash
mkdir -p env && touch env/global.env
```

If hulak does not find the env folder and `global.env` file inside, it asks the user to create one during runtime.

## Getting Started

Hualk is designed to be simple and intuitive.

Then Basic API call looks like `test.yaml`

```yaml
# test.yaml
method: Get
url: https://jsonplaceholder.typicode.com/todos/1
```

`method` and `url` are case ininsensitive. So, as `Method` values.

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
| `-f`   | Yaml/yml file to run. Hulak recurses though your directories and subdirectories, excluding hidden, from the root and finds the matching file(s). If multiple matches are found, they run concurrently | `-f graphql`                     |

## Actions

Actions make it easier to retrieve values from other files.

### Retrieving secrets with `.Key`

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

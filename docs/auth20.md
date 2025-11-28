# Auth2.0

Hualk supports auth2.0 web-application-flow. Follow the auth2.0 provider instruction to set it up.

## Brief Intro to Auth2.0 flow

OAuth 2.0 is a protocol for authorizing access. In a typical flow, user registers an app, hulak in our case, with the OAuth provider, say Github or Okta. During the process, the auth2.0 provider asks for redirect url to send the code to. You also obtain a `client_id` and a `client_secret`.
Once the app is registered, you can easily authorize the app to grab the token. For github, the process is listed [here](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps#web-application-flow)

- Redirect URL for hulak: `http://localhost:2982/callback`

> [!Warning]
> The feature is in beta because it has only been tested with Github.
> Also, features like refresh token has not been implemented yet.

Below is the example of how, say `auth2.yaml` file would look after registering hulak with Github [web-application-flow](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps#web-application-flow).

```yaml
# kind is required field if  the yaml is meant for auth2.0
kind: auth # required for auth2.0. If missing, this yaml will be treated as an API call
method: POST
# url that opens up in a browser and asks you to login. Usually ends with /authorize
url: https://github.com/login/oauth/authorize
urlparams:
  client_id: "{{.client_id}}" # client_id you receive while registering hulak. If you are part of an org, the admin will provide you this value
  scope: repo:user
auth:
  type: OAuth2.0
  # url to retrieve access token after browser authorization
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

- Run the file just like any other file

```
hulak -env staging -f auth2
```

The response is logged in the console and `auth2_response.json` file is saved in the same location of the caller. To use the `access_token` received in this response use `getValueOf` action. For example, this is how you can insert the access_token received from above

```yaml
method: POST
url: "{{.graphqlUrl}}"
headers:
  Content-Type: application/json
  Authorization: Bearer {{getValueOf "access_token" "auth2"}} # automatically inserted as `Bearere eyBexai...`
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

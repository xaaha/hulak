# Auth2.0

Hualk supports auth2.0 web-application-flow. Follow the auth2.0 provider instruction to set it up.

## Brief Intro to Auth2.0 flow

OAuth 2.0 is a protocol for authorizing access. In a typical flow, user registers their app with the OAuth provider, say Github or Okta, obtains a `client_id` and `secret`.
User needs to add the redirect uri, listed below, in the auth2.0 provider as well. Once the user consents, they are redirected back to hulak with an authorization `code` that can be exchanged for an access token.

- Redirect URL: `http://localhost:2982/callback`

> [!Warning]
> The feature is in beta because it has only been tested with github.
> User is responsible for registering their app and redirect url with the auth2.0 provider. Example For [github](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps#web-application-flow).

Below is the example of how Auth2.0 would look like for `auth2.yaml` file

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

The response is logged in the console and `auth2_response.json` file is saved in the same location as the calling file. To use the `access_token` received in this response use `getValueOf` action.

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

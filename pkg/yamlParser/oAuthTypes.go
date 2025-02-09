package yamlParser

import (
	"github.com/xaaha/hulak/pkg/utils"
)

// auth type is required as we need to see early which flow to call
type authtype string

// Variation of OAuth2.0 for auth.type
const (
	Oauth2type1 authtype = "OAuth2.0"
	Oauth2type2 authtype = "oauth2"
	Oauth2type3 authtype = "oauth2.0"
)

// Represents how Auth section in yaml looks like
type Auth struct {
	Type           authtype `json:"type"             yaml:"type"`
	AccessTokenUrl URL      `json:"access_token_url" yaml:"access_token_url"`
}

// check's if auth key contains type and has at least 1 item in Extras
// we need to extend this validation as we need them
func (a *Auth) IsValid() bool {
	if a == nil {
		return false
	}

	switch a.Type {
	case Oauth2type1, Oauth2type2, Oauth2type3:
		if a.AccessTokenUrl == "" || !a.AccessTokenUrl.IsValidURL() {
			return false
		}
		return true
	default:
		// Unsupported type is considered invalid
		return false
	}
}

type URLPARAMS map[string]string

// Checks if the URLPARAMS map contains the required "client_id" key.
func (u URLPARAMS) IsValid() bool {
	if u == nil {
		return false
	}
	_, ok := u["client_id"]
	return ok
}

// represents how a yaml file for Auth2.0 would look like
type AuthRequestBody struct {
	Method    HTTPMethodType    `json:"method"              yaml:"method"`
	Url       URL               `json:"url"                 yaml:"url"`
	UrlParams URLPARAMS         `json:"urlparams,omitempty" yaml:"urlparams"`
	Auth      *Auth             `json:"auth"                yaml:"auth"`
	Headers   map[string]string `json:"headers,omitempty"   yaml:"headers"`
	Body      Body
}

func (auth2Body *AuthRequestBody) IsValid() (bool, error) {
	if auth2Body == nil {
		return false, utils.ColorError("auth request body is nil")
	}

	// Validate Auth section
	if auth2Body.Auth == nil {
		return false, utils.ColorError("auth section is missing")
	}

	if valid := auth2Body.Auth.IsValid(); !valid {
		return false, utils.ColorError(
			"Invalid 'auth' section. Make sure the file '%s' contains valid auth section.",
		)
	}

	// Validate URL
	if !auth2Body.Url.IsValidURL() {
		return false, utils.ColorError("invalid URL in auth request body")
	}

	// Validate optional URL parameters
	if len(auth2Body.UrlParams) > 0 && !auth2Body.UrlParams.IsValid() {
		return false, utils.ColorError("invalid URL parameters")
	}

	// Validate Body if it exists
	if !auth2Body.Body.IsValid() {
		return false, utils.ColorError("invalid body content")
	}

	return true, nil
}

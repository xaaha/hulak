package yamlParser

// auth type is required as we need to see early which flow to call
type authtype string

// Variation of OAuth2.0 for auth.type
const (
	Oauth2type1 authtype = "OAuth2.0"
	Oauth2type2 authtype = "oauth2"
	Oauth2type3 authtype = "oauth2.0"
)

// Represents how Auth section in the Auth yaml looks like
type Auth struct {
	Type   authtype          `json:"type"   yaml:"type"`
	Extras map[string]string `json:"extras" yaml:"extras"`
}

type URLPARAMS map[string]string

// represents how a yaml file for Auth2.0 would look like
type AuthRequestBody struct {
	Method    HTTPMethodType    `json:"method"              yaml:"method"`
	Url       URL               `json:"url"                 yaml:"url"`
	UrlParams URLPARAMS         `json:"urlparams,omitempty" yaml:"urlparams"`
	Auth      *Auth             `json:"auth"                yaml:"auth"`
	Headers   map[string]string `json:"headers,omitempty"   yaml:"headers"`
}

// check's if auth key contains type and has at least 1 item in Extras
// we need to extend this validation as we need them
func (a *Auth) IsValid() bool {
	switch a.Type {
	case Oauth2type1, Oauth2type2, Oauth2type3:
		if _, ok := a.Extras["access_token_url"]; !ok {
			return false
		}
		return len(a.Extras) > 0
	default:
		return false
	}
}

// Checks if the URLPARAMS map contains the required "client_id" key.
func (u URLPARAMS) IsValid() bool {
	if u == nil {
		return false
	}
	_, ok := u["client_id"]
	return ok
}

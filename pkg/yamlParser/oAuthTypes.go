package yamlParser

// auth type is required as we need to see early which flow to call
type authtype string

// Variation of OAuth2.0 for auth.type
const (
	Oauth2type1 authtype = "OAuth2.0"
	Oauth2type2 authtype = "oauth2"
	Oauth2type3 authtype = "oauth2.0"
)

// Represents how Auth body in the AuthRequestBody looks like
type Auth struct {
	Type   authtype          `json:"type"   yaml:"type"`
	Extras map[string]string `json:"extras" yaml:"extras"`
}

// represents how a yaml file for Auth2.0 would look like
type AuthRequestBody struct {
	Method HTTPMethodType `json:"method" yaml:"method"`
	Url    URL            `json:"url"    yaml:"url"`
	Auth   *Auth          `json:"auth"   yaml:"auth"`
}

// check's if auth key contains type and has at least 1 item in Extras
// we need to extend this validation as we need them
func (a *Auth) IsValid() bool {
	switch a.Type {
	case Oauth2type1, Oauth2type2, Oauth2type3:
		return len(a.Extras) > 0
	default:
		return false
	}
}

// use checkYamlFile to get the buffer content
// Prepare URL with the key value pair... This can be done with EncodeXwwwFormUrlBody
// Then Open the URL we just prepared
// After user authorization, we'll receive a code
// Capture the token from the browser, you need to spin up the local server
// Save the token... or not... I am not sure if this
// Then, finally
// Exchange the code for an access token
// API call with POST request
// 	token, err := GetAccessToken(config, authCode)
// 	if err != nil {
// 		fmt.Printf("Error getting access token: %v\n", err)
// 		return
// 	}
//
// 	fmt.Printf("Response: %s\n", token)
// }

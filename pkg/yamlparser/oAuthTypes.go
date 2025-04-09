// Package yamlparser does everything related to yaml file for hulak, including type translation
package yamlparser

import (
	"io"

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

// Auth Represents how Auth section in yaml looks like
type Auth struct {
	Type           authtype `json:"type"             yaml:"type"`
	AccessTokenURL URL      `json:"access_token_url" yaml:"access_token_url"`
}

// IsValid check's if auth key contains type and has at least 1 item in Extras
// we need to extend this validation as we need them
func (a *Auth) IsValid() bool {
	if a == nil {
		return false
	}

	switch a.Type {
	case Oauth2type1, Oauth2type2, Oauth2type3:
		if a.AccessTokenURL == "" || !a.AccessTokenURL.IsValidURL() {
			return false
		}
		return true
	default:
		// Unsupported type is considered invalid
		return false
	}
}

// URLPARAMS is the standard url params just like in api file
type URLPARAMS map[string]string

// IsValid checks if the URLPARAMS map contains the required "client_id" key.
func (u URLPARAMS) IsValid() bool {
	if u == nil {
		return false
	}
	_, ok := u["client_id"]
	return ok
}

// Auth2Body represents the body of an Auth2.0 request, which typically contains
// URL-encoded form data as a map of string keys and values.
type Auth2Body struct {
	URLEncodedFormData map[string]string `json:"urlencodedformdata" yaml:"urlencodedformdata"`
}

// IsValid checks if the Auth2Body is valid. A valid Auth2Body meets the following conditions:
// 1. The body must not be nil.
// 2. The `UrlEncodedFormData` field must contain at least one item (i.e., it's non-empty).
//
// This function assumes that the Auth2.0 request body should mainly contain URL-encoded
// form data, as this is the expected format for most Auth2.0 implementations. Other formats,
// such as RawString or GraphQL, are considered invalid in this context (for now).
func (b *Auth2Body) IsValid() bool {
	// Ensure that the struct is not nil and has data
	if b == nil {
		return false
	}
	// Check if the UrlEncodedFormData field has at least one item
	if len(b.URLEncodedFormData) > 0 {
		return true
	}
	// If UrlEncodedFormData is empty or missing, it's considered invalid
	return false
}

// EncodeBody encodes the *Auth2Body
func (b *Auth2Body) EncodeBody(code string) (io.Reader, string, error) {
	var body io.Reader
	var contentType string

	if b == nil {
		return nil, "", nil
	}

	codeMap := make(map[string]string)
	codeMap[utils.ResponseType] = code
	mergedMap := utils.MergeMaps(b.URLEncodedFormData, codeMap)

	switch {
	case len(b.URLEncodedFormData) > 0:
		encodedBody, err := EncodeXwwwFormURLBody(mergedMap)
		if err != nil {
			return nil, "", utils.ColorError("#oAuthTypes.go", err)
		}
		body, contentType = encodedBody, "application/x-www-form-urlencoded"
	default:
		return nil, "", utils.ColorError("no valid body type provided")

	}
	return body, contentType, nil
}

// AuthRequestFile  represents how a yaml file for Auth2.0 would look like
type AuthRequestFile struct {
	Method    HTTPMethodType    `json:"method"              yaml:"method"`
	URL       URL               `json:"url"                 yaml:"url"`
	URLParams URLPARAMS         `json:"urlparams,omitempty" yaml:"urlparams"`
	Auth      *Auth             `json:"auth"                yaml:"auth"`
	Headers   map[string]string `json:"headers,omitempty"   yaml:"headers"`
	Body      *Auth2Body
}

// IsValid checks if AuthRequestBody is valid ,
// Has valid method, if missing method defaults to post.
// Has valid auth section with type and access_token_url, for auth2.0
// Has Required, and valid Url
// If UrlParams is present, client_id is required
// Valid Body is present
func (auth2Body *AuthRequestFile) IsValid() (bool, error) {
	if auth2Body == nil {
		return false, utils.ColorError("auth request body is nil")
	}

	// If method is  missing, By default, method is POST for Auth2.0
	if auth2Body.Method == "" {
		auth2Body.Method = POST
	}

	// uppercase the method
	auth2Body.Method.ToUpperCase()

	// method is required as each implementation of  Auth2.0 is different
	if !auth2Body.Method.IsValid() {
		return false, utils.ColorError("invalid HTTP method " + string(auth2Body.Method))
	}

	// Validate Auth section
	if auth2Body.Auth == nil {
		return false, utils.ColorError("when 'Kind: auth' is present, auth section is required")
	}

	if valid := auth2Body.Auth.IsValid(); !valid {
		return false, utils.ColorError(
			"invalid 'auth' section. Make sure the Auth2.0 file contains valid auth section with 'type' && access_token_url",
		)
	}

	// Validate URL
	if !auth2Body.URL.IsValidURL() {
		return false, utils.ColorError("missing or invalid URL in auth request body")
	}

	// Validate optional URL parameters, if present
	if len(auth2Body.URLParams) > 0 && !auth2Body.URLParams.IsValid() {
		return false, utils.ColorError("invalid URL parameters")
	}

	// Validate Body
	if !auth2Body.Body.IsValid() {
		return false, utils.ColorError("invalid body content")
	}

	return true, nil
}

// PrepareStruct prepars struct for the standard call
func (auth2Body *AuthRequestFile) PrepareStruct(code string) (ApiInfo, error) {
	body, contentType, err := auth2Body.Body.EncodeBody(code)
	if err != nil {
		return ApiInfo{}, utils.ColorError("#apiTypes.go", err)
	}

	if contentType != "" {
		if auth2Body.Headers == nil {
			auth2Body.Headers = make(map[string]string)
		}
		auth2Body.Headers["content-type"] = contentType
	}

	return ApiInfo{
		Method:    string(auth2Body.Method),
		Url:       string(auth2Body.Auth.AccessTokenURL),
		UrlParams: auth2Body.URLParams,
		Headers:   auth2Body.Headers,
		Body:      body,
	}, nil
}

package yamlparser

import (
	"strings"
	"testing"
)

func TestAuth_IsValid(t *testing.T) {
	tests := []struct {
		name string
		auth Auth
		want bool
	}{
		{
			name: "Valid type Oauth2type1 with valid AccessTokenUrl",
			auth: Auth{
				Type:           Oauth2type1,
				AccessTokenURL: "https://www.example.com/",
			},
			want: true,
		},
		{
			name: "Valid type Oauth2type2 without AccessTokenUrl",
			auth: Auth{
				Type:           Oauth2type2,
				AccessTokenURL: "",
			},
			want: false,
		},
		{
			name: "Valid type Oauth2type3 with empty AccessTokenUrl",
			auth: Auth{
				Type:           Oauth2type3,
				AccessTokenURL: "",
			},
			want: false,
		},
		{
			name: "Invalid type with valid AccessTokenUrl",
			auth: Auth{
				Type:           "invalid",
				AccessTokenURL: "https://www.example.com/",
			},
			want: false,
		},
		{
			name: "Invalid type with invalid AccessTokenUrl",
			auth: Auth{
				Type:           "invalid",
				AccessTokenURL: "example.com",
			},
			want: false,
		},
		{
			name: "Empty type with empty AccessTokenUrl",
			auth: Auth{
				Type:           "",
				AccessTokenURL: "",
			},
			want: false,
		},
		{
			name: "Valid type Oauth2type1 with invalid AccessTokenUrl",
			auth: Auth{
				Type:           Oauth2type1,
				AccessTokenURL: "invalid-url",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.auth.IsValid()
			if got != tt.want {
				t.Errorf("Auth.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestURLPARAMS_IsValid(t *testing.T) {
	tests := []struct {
		name      string
		urlParams URLPARAMS
		want      bool
	}{
		{
			name: "Valid UrlParams with client_id",
			urlParams: URLPARAMS{
				"client_id": "12345",
				"scope":     "read",
			},
			want: true,
		},
		{
			name: "Invalid UrlParams without client_id",
			urlParams: URLPARAMS{
				"scope": "read",
			},
			want: false,
		},
		{
			name:      "Empty UrlParams",
			urlParams: URLPARAMS{},
			want:      false,
		},
		{
			name:      "Nil UrlParams",
			urlParams: nil,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.urlParams.IsValid()
			if got != tt.want {
				t.Errorf("URLPARAMS.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAuthRequestBody_IsValid(t *testing.T) {
	tests := []struct {
		name         string
		authRequest  AuthRequestFile
		expectedBool bool
		expectedErr  string
	}{
		{
			name: "Valid request: Oauth2type1, valid URL, and valid params, and valid body",
			authRequest: AuthRequestFile{
				Method: POST,
				URL:    "https://api.example.com",
				URLParams: URLPARAMS{
					"client_id": "validClientId",
				},
				Auth: &Auth{
					Type:           Oauth2type1,
					AccessTokenURL: "https://auth.example.com/token",
				},
				Body: &Auth2Body{
					URLEncodedFormData: map[string]string{
						"client_id": "xaaha",
					},
				},
			},
			expectedBool: true,
			expectedErr:  "",
		},
		{
			name: "Valid request: Oauth2type1, valid URL, and valid params, and missing body",
			authRequest: AuthRequestFile{
				Method: POST,
				URL:    "https://api.example.com",
				URLParams: URLPARAMS{
					"client_id": "validClientId",
				},
				Auth: &Auth{
					Type:           Oauth2type1,
					AccessTokenURL: "https://auth.example.com/token",
				},
			},
			expectedBool: false, // missing body is not allowed
			expectedErr:  "invalid body content",
		},
		{
			name: "Missing auth section",
			authRequest: AuthRequestFile{
				Method: POST,
				URL:    "https://api.example.com",
				Auth:   nil,
			},
			expectedBool: false,
			expectedErr:  "auth section is required",
		},
		{
			name: "Invalid auth type with valid AccessTokenUrl",
			authRequest: AuthRequestFile{
				Method: POST,
				URL:    "https://api.example.com",
				Auth: &Auth{
					Type:           "invalid_type",
					AccessTokenURL: "https://auth.example.com/token",
				},
			},
			expectedBool: false,
			expectedErr:  "invalid 'auth' section",
		},
		{
			name: "Invalid AccessTokenUrl in Auth",
			authRequest: AuthRequestFile{
				Method: POST,
				URL:    "https://api.example.com",
				Auth: &Auth{
					Type:           Oauth2type1,
					AccessTokenURL: "invalid-url",
				},
			},
			expectedBool: false,
			expectedErr:  "invalid 'auth' section",
		},
		{
			name: "Missing URL in auth request body",
			authRequest: AuthRequestFile{
				Method: POST,
				Auth: &Auth{
					Type:           Oauth2type2,
					AccessTokenURL: "https://auth.example.com/token",
				},
			},
			expectedBool: false,
			expectedErr:  "missing or invalid URL in auth request body",
		},
		{
			name: "Invalid UrlParams without client_id",
			authRequest: AuthRequestFile{
				Method: POST,
				URL:    "https://api.example.com",
				Auth: &Auth{
					Type:           Oauth2type2,
					AccessTokenURL: "https://auth.example.com/token",
				},
				URLParams: URLPARAMS{
					"scope": "read",
				},
			},
			expectedBool: false,
			expectedErr:  "invalid URL parameters",
		},
		{
			name: "Invalid HTTP method",
			authRequest: AuthRequestFile{
				Method: "INVALID",
				URL:    "https://api.example.com",
				Auth: &Auth{
					Type:           Oauth2type2,
					AccessTokenURL: "https://auth.example.com/token",
				},
			},
			expectedBool: false,
			expectedErr:  "invalid HTTP method INVALID",
		},
		{
			name: "Valid request without UrlParams",
			authRequest: AuthRequestFile{
				Method: POST,
				URL:    "https://api.example.com",
				Auth: &Auth{
					Type:           Oauth2type1,
					AccessTokenURL: "https://auth.example.com/token",
				},
				Body: &Auth2Body{
					URLEncodedFormData: map[string]string{
						"client_id":     "my_id",
						"client_secret": "my_secret",
					},
				},
			},
			expectedBool: true,
			expectedErr:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBool, gotErr := tt.authRequest.IsValid()
			if gotBool != tt.expectedBool {
				t.Errorf(
					"AuthRequestBody.IsValid() bool = %v, expected %v",
					gotBool,
					tt.expectedBool,
				)
			}

			if (gotErr != nil && tt.expectedErr == "") || (gotErr == nil && tt.expectedErr != "") {
				t.Errorf(
					"AuthRequestBody.IsValid() error = %v, expected %v",
					gotErr,
					tt.expectedErr,
				)
			} else if gotErr != nil && !strings.Contains(strings.TrimSpace(gotErr.Error()), strings.TrimSpace(tt.expectedErr)) {
				t.Errorf("AuthRequestBody.IsValid() error = %v, expected %v", gotErr.Error(), tt.expectedErr)
			}
		})
	}
}

package yamlParser

import (
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
				AccessTokenUrl: "example.com",
			},
			want: true,
		},
		{
			name: "Valid type Oauth2type2 without AccessTokenUrl",
			auth: Auth{
				Type:           Oauth2type2,
				AccessTokenUrl: "",
			},
			want: false,
		},
		{
			name: "Valid type Oauth2type3 with empty AccessTokenUrl",
			auth: Auth{
				Type:           Oauth2type3,
				AccessTokenUrl: "",
			},
			want: false,
		},
		{
			name: "Invalid type with AccessTokenUrl",
			auth: Auth{
				Type:           "invalid",
				AccessTokenUrl: "example.com",
			},
			want: false,
		},
		{
			name: "Empty type with empty AccessTokenUrl",
			auth: Auth{
				Type:           "",
				AccessTokenUrl: "",
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

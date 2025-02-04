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
			name: "Valid type Oauth2type1 with valid extras",
			auth: Auth{
				Type: Oauth2type1,
				Extras: map[string]string{
					"access_token_url": "example.com",
					"scope":            "read",
				},
			},
			want: true,
		},
		{
			name: "Valid type Oauth2type2 without access_token_url",
			auth: Auth{
				Type: Oauth2type2,
				Extras: map[string]string{
					"state": "active",
				},
			},
			want: false,
		},
		{
			name: "Valid type Oauth2type3 with no extras",
			auth: Auth{
				Type:   Oauth2type3,
				Extras: map[string]string{},
			},
			want: false,
		},
		{
			name: "Invalid type with access_token_url in extras",
			auth: Auth{
				Type: "invalid",
				Extras: map[string]string{
					"access_token_url": "example.com",
				},
			},
			want: false,
		},
		{
			name: "Empty type with no extras",
			auth: Auth{
				Type:   "",
				Extras: map[string]string{},
			},
			want: false,
		},
		{
			name: "Valid type with nil extras",
			auth: Auth{
				Type:   Oauth2type1,
				Extras: nil,
			},
			want: false,
		},
		{
			name: "Valid type Oauth2type1 with only access_token_url",
			auth: Auth{
				Type: Oauth2type1,
				Extras: map[string]string{
					"access_token_url": "example.com",
				},
			},
			want: true,
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

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
			name: "Valid type Oauth2type1 with extras",
			auth: Auth{
				Type: Oauth2type1,
				Extras: map[string]string{
					"scope": "read",
				},
			},
			want: true,
		},
		{
			name: "Valid type Oauth2type2 with extras",
			auth: Auth{
				Type: Oauth2type2,
				Extras: map[string]string{
					"state": "active",
				},
			},
			want: true,
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
			name: "Invalid type with extras",
			auth: Auth{
				Type: "invalid",
				Extras: map[string]string{
					"client_id": "12345",
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

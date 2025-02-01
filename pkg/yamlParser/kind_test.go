package yamlParser

import (
	"reflect"
	"testing"
)

func TestConfigType_IsValid(t *testing.T) {
	tests := []struct {
		name string
		kind Kind
		want bool
	}{
		{
			name: "empty kind defaults to API",
			kind: "",
			want: true,
		},
		{
			name: "valid API uppercase",
			kind: "API",
			want: true,
		},
		{
			name: "valid API lowercase",
			kind: "api",
			want: true,
		},
		{
			name: "valid API mixed case",
			kind: "Api",
			want: true,
		},
		{
			name: "valid Auth uppercase",
			kind: "AUTH",
			want: true,
		},
		{
			name: "valid Auth lowercase",
			kind: "auth",
			want: true,
		},
		{
			name: "invalid kind",
			kind: "invalid",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := &ConfigType{Kind: tt.kind}
			if got := conf.IsValid(); got != tt.want {
				t.Errorf("ConfigType.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigType_GetKind(t *testing.T) {
	tests := []struct {
		name string
		kind Kind
		want Kind
	}{
		{
			name: "empty kind returns API",
			kind: "",
			want: KindAPI,
		},
		{
			name: "API different case returns canonical form",
			kind: "api",
			want: KindAPI,
		},
		{
			name: "Auth different case returns canonical form",
			kind: "auth",
			want: KindAuth,
		},
		{
			name: "invalid kind returns lowercase version",
			kind: "Invalid",
			want: "invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := &ConfigType{Kind: tt.kind}
			if got := conf.GetKind(); got != tt.want {
				t.Errorf("ConfigType.GetKind() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateKinds(t *testing.T) {
	tests := []struct {
		name        string
		kinds       []Kind
		wantInvalid []string
		wantValid   bool
	}{
		{
			name:        "all valid kinds",
			kinds:       []Kind{"API", "Auth"},
			wantInvalid: nil,
			wantValid:   true,
		},
		{
			name:        "mixed case valid kinds",
			kinds:       []Kind{"api", "AUTH"},
			wantInvalid: nil,
			wantValid:   true,
		},
		{
			name:        "some invalid kinds",
			kinds:       []Kind{"API", "invalid", "Auth", "unknown"},
			wantInvalid: []string{"invalid", "unknown"},
			wantValid:   false,
		},
		{
			name:        "empty list",
			kinds:       []Kind{},
			wantInvalid: nil,
			wantValid:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotInvalid, gotValid := ValidateKinds(tt.kinds)
			if !reflect.DeepEqual(gotInvalid, tt.wantInvalid) {
				t.Errorf("ValidateKinds() invalid = %v, want %v", gotInvalid, tt.wantInvalid)
			}
			if gotValid != tt.wantValid {
				t.Errorf("ValidateKinds() valid = %v, want %v", gotValid, tt.wantValid)
			}
		})
	}
}

func TestConfigType_IsAuth_IsAPI(t *testing.T) {
	tests := []struct {
		name     string
		kind     Kind
		wantAuth bool
		wantAPI  bool
	}{
		{
			name:     "empty kind is API",
			kind:     "",
			wantAuth: false,
			wantAPI:  true,
		},
		{
			name:     "API uppercase",
			kind:     "API",
			wantAuth: false,
			wantAPI:  true,
		},
		{
			name:     "Auth uppercase",
			kind:     "AUTH",
			wantAuth: true,
			wantAPI:  false,
		},
		{
			name:     "invalid kind is neither",
			kind:     "invalid",
			wantAuth: false,
			wantAPI:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := &ConfigType{Kind: tt.kind}
			if got := conf.IsAuth(); got != tt.wantAuth {
				t.Errorf("ConfigType.IsAuth() = %v, want %v", got, tt.wantAuth)
			}
			if got := conf.IsAPI(); got != tt.wantAPI {
				t.Errorf("ConfigType.IsAPI() = %v, want %v", got, tt.wantAPI)
			}
		})
	}
}

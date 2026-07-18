package utils

import "testing"

func TestCanonicalActionName(t *testing.T) {
	tests := []struct {
		token string
		want  string
		ok    bool
	}{
		{"getFile", "getFile", true},
		{"getfile", "getFile", true},
		{"GetFile", "getFile", true},
		{"GETFILE", "getFile", true},
		{"get_file", "getFile", true},
		{"Get_File", "getFile", true},
		{"getValueOf", "getValueOf", true},
		{"get_value_of", "getValueOf", true},
		{"GETVALUEOF", "getValueOf", true},
		{"basicAuth", "basicAuth", true},
		{"basic_auth", "basicAuth", true},
		{"os", "os", true},
		{"OS", "os", true},
		{"unknown", "", false},
		{"getFiles", "", false},
		{"", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.token, func(t *testing.T) {
			got, ok := CanonicalActionName(tt.token)
			if ok != tt.ok {
				t.Fatalf("CanonicalActionName(%q) ok = %v, want %v", tt.token, ok, tt.ok)
			}
			if got != tt.want {
				t.Errorf("CanonicalActionName(%q) = %q, want %q", tt.token, got, tt.want)
			}
		})
	}
}

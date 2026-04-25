package utils

import (
	"strings"
	"testing"
)

func TestValidateEnvName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// valid
		{"simple lowercase", "global", false},
		{"with digit", "prod1", false},
		{"with underscore", "team_a", false},
		{"with hyphen", "team-a", false},
		{"mixed case", "StagingV2", false},
		{"single char", "a", false},
		{"max length (64)", strings.Repeat("a", MaxEnvNameLen), false},
		{"all separators allowed", "team_a-1", false},

		// invalid
		{"empty string", "", true},
		{"contains space", "my env", true},
		{"contains dot", "my.env", true},
		{"contains slash", "team/prod", true},
		{"contains backslash", `team\prod`, true},
		{"contains shell metachar", "prod;rm", true},
		{"contains dollar", "$prod", true},
		{"unicode letter", "préprod", true},
		{"too long (65)", strings.Repeat("a", MaxEnvNameLen+1), true},
		{"path traversal", "../etc", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateEnvName(tc.input)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateEnvName(%q) error = %v, wantErr %v", tc.input, err, tc.wantErr)
			}
		})
	}
}

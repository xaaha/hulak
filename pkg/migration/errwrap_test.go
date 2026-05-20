package migration

import (
	"strings"
	"testing"
)

// TestPrepareEnvStructErrorsNoANSI guards against regression of issue #180.
// The 7 validation branches in PrepareEnvStruct previously used
// utils.ColorError which injected ANSI escapes and a leading newline. After
// migration to errors.New, error text must be plain.
func TestPrepareEnvStructErrorsNoANSI(t *testing.T) {
	cases := []struct {
		name   string
		input  map[string]any
		substr string
	}{
		{
			name:   "missing name",
			input:  map[string]any{},
			substr: "name field is missing",
		},
		{
			name:   "missing values",
			input:  map[string]any{"name": "x"},
			substr: "values field is missing",
		},
		{
			name:   "missing scope",
			input:  map[string]any{"name": "x", "values": []any{}},
			substr: "scope field is missing",
		},
		{
			name: "value not a map",
			input: map[string]any{
				"name":   "x",
				"values": []any{"not-a-map"},
			},
			substr: "value is not a valid map",
		},
		{
			name: "missing key",
			input: map[string]any{
				"name":   "x",
				"values": []any{map[string]any{}},
			},
			substr: "key field is missing",
		},
		{
			name: "missing value field",
			input: map[string]any{
				"name":   "x",
				"values": []any{map[string]any{"key": "k"}},
			},
			substr: "value field is missing",
		},
		{
			name: "missing enabled",
			input: map[string]any{
				"name":   "x",
				"values": []any{map[string]any{"key": "k", "value": "v"}},
			},
			substr: "enabled field is missing",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := PrepareEnvStruct(tc.input)
			if err == nil {
				t.Fatal("expected error")
			}
			msg := err.Error()
			if strings.Contains(msg, "\x1b") {
				t.Errorf("error contains ANSI escape: %q", msg)
			}
			if strings.HasPrefix(msg, "\n") {
				t.Errorf("error starts with newline: %q", msg)
			}
			if !strings.Contains(msg, tc.substr) {
				t.Errorf("expected %q in %q", tc.substr, msg)
			}
		})
	}
}

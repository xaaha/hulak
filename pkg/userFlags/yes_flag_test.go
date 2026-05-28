package userflags

import (
	"flag"
	"testing"
)

// TestRegisterYesFlag verifies --yes and -y both register and share the
// same underlying variable. Without that, the long form and short form
// would silently diverge: a destructive op honoring --yes but not -y is
// the kind of bug that's easy to miss in code review.
func TestRegisterYesFlag(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"long form --yes", []string{"--yes"}},
		{"short form -y", []string{"-y"}},
		{"unset stays false", nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			yes := registerYesFlag(fs, "skip confirm")
			if err := fs.Parse(tc.args); err != nil {
				t.Fatalf("Parse(%v): %v", tc.args, err)
			}
			want := len(tc.args) > 0
			if *yes != want {
				t.Errorf("yes = %v, want %v", *yes, want)
			}
		})
	}
}

// TestRegisterYesFlagAlias verifies both Lookup entries point at the same
// variable — the contract that callers rely on.
func TestRegisterYesFlagAlias(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	yes := registerYesFlag(fs, "skip confirm")

	long := fs.Lookup("yes")
	short := fs.Lookup("y")
	if long == nil || short == nil {
		t.Fatalf("expected both --yes and -y registered, got long=%v short=%v", long, short)
	}

	if err := long.Value.Set("true"); err != nil {
		t.Fatalf("set --yes: %v", err)
	}
	if !*yes {
		t.Error("setting --yes did not update the bound variable")
	}
	if short.Value.String() != "true" {
		t.Errorf("-y view of variable = %q, want true (should share storage)", short.Value.String())
	}
}

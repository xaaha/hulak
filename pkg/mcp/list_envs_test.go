package mcp

import (
	"context"
	"path/filepath"
	"slices"
	"testing"
)

func TestHandleListEnvs(t *testing.T) {
	ctx := context.Background()

	t.Run("lists env names for a classic project", func(t *testing.T) {
		api := projectDir(t)
		writeFileAt(t, filepath.Join(api, "env", "global.env"), "A=1\n")
		writeFileAt(t, filepath.Join(api, "env", "test.env"), "B=2\n")
		s, err := NewServer(map[string]string{"api": api}, "", "v")
		if err != nil {
			t.Fatal(err)
		}

		_, out, err := s.handleListEnvs(ctx, nil, listEnvsInput{})
		if err != nil {
			t.Fatal(err)
		}
		if len(out.Environments) != 1 {
			t.Fatalf("want 1 project group, got %d", len(out.Environments))
		}
		got := out.Environments[0]
		if got.Project != "api" {
			t.Errorf("project = %q, want api", got.Project)
		}
		for _, want := range []string{"global", "test"} {
			if !slices.Contains(got.Envs, want) {
				t.Errorf("envs %v missing %q", got.Envs, want)
			}
		}
	})

	t.Run("groups by project across all projects", func(t *testing.T) {
		api := projectDir(t)
		mob := projectDir(t)
		writeFileAt(t, filepath.Join(api, "env", "prod.env"), "A=1\n")
		writeFileAt(t, filepath.Join(mob, "env", "dev.env"), "B=2\n")
		s, err := NewServer(map[string]string{"api": api, "mob": mob}, "api", "v")
		if err != nil {
			t.Fatal(err)
		}

		_, out, err := s.handleListEnvs(ctx, nil, listEnvsInput{})
		if err != nil {
			t.Fatal(err)
		}
		if len(out.Environments) != 2 {
			t.Fatalf("want 2 project groups, got %d", len(out.Environments))
		}
		// projectNames sorts: api before mob.
		if out.Environments[0].Project != "api" || !slices.Contains(out.Environments[0].Envs, "prod") {
			t.Errorf("api group wrong: %+v", out.Environments[0])
		}
		if out.Environments[1].Project != "mob" || !slices.Contains(out.Environments[1].Envs, "dev") {
			t.Errorf("mob group wrong: %+v", out.Environments[1])
		}
	})

	t.Run("scopes to a single project", func(t *testing.T) {
		api := projectDir(t)
		mob := projectDir(t)
		writeFileAt(t, filepath.Join(api, "env", "prod.env"), "A=1\n")
		writeFileAt(t, filepath.Join(mob, "env", "dev.env"), "B=2\n")
		s, err := NewServer(map[string]string{"api": api, "mob": mob}, "api", "v")
		if err != nil {
			t.Fatal(err)
		}

		_, out, err := s.handleListEnvs(ctx, nil, listEnvsInput{Project: "mob"})
		if err != nil {
			t.Fatal(err)
		}
		if len(out.Environments) != 1 || out.Environments[0].Project != "mob" {
			t.Fatalf("want only mob group, got %+v", out.Environments)
		}
	})

	t.Run("unknown project errors", func(t *testing.T) {
		s, err := NewServer(map[string]string{"api": projectDir(t)}, "", "v")
		if err != nil {
			t.Fatal(err)
		}
		if _, _, err := s.handleListEnvs(ctx, nil, listEnvsInput{Project: "nope"}); err == nil {
			t.Error("expected error for unknown project")
		}
	})
}

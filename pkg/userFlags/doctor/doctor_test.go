package doctor

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xaaha/hulak/pkg/utils"
)

// chdirTemp changes to the given directory and returns a function that restores
// the original working directory. Call the returned function with defer.
func chdirTemp(t *testing.T, dir string) func() {
	t.Helper()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir to %s: %v", dir, err)
	}
	return func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	}
}

// createEnvDir creates an env/ directory inside the given parent directory.
func createEnvDir(t *testing.T, parent string) string {
	t.Helper()
	envDir := filepath.Join(parent, utils.EnvironmentFolder)
	if err := os.Mkdir(envDir, utils.DirPer); err != nil {
		t.Fatalf("failed to create env dir: %v", err)
	}
	return envDir
}

// createEnvFile creates a .env file inside the env/ directory with the given
// permissions and content.
func createEnvFile(t *testing.T, envDir, name string, perm os.FileMode, content string) {
	t.Helper()
	path := filepath.Join(envDir, name+utils.DefaultEnvFileSuffix)
	if err := os.WriteFile(path, []byte(content), perm); err != nil {
		t.Fatalf("failed to create env file %s: %v", name, err)
	}
}

// gitInit initializes a git repository in the current working directory.
func gitInit(t *testing.T) {
	t.Helper()
	cmd := exec.Command("git", "init")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, output)
	}
}

// gitAddCommit stages all files and creates a commit in the current directory.
func gitAddCommit(t *testing.T, message string) {
	t.Helper()
	add := exec.Command("git", "add", "-A")
	if output, err := add.CombinedOutput(); err != nil {
		t.Fatalf("git add failed: %v\n%s", err, output)
	}
	commit := exec.Command("git",
		"-c", "user.name=Test",
		"-c", "user.email=test@test.com",
		"commit", "--allow-empty-message", "-m", message,
	)
	if output, err := commit.CombinedOutput(); err != nil {
		t.Fatalf("git commit failed: %v\n%s", err, output)
	}
}

// findingsContain returns true if any finding message or fix contains substr.
func findingsContain(findings []finding, substr string) bool {
	for _, f := range findings {
		if strings.Contains(f.message, substr) || strings.Contains(f.fix, substr) {
			return true
		}
	}
	return false
}

func TestCheckGitignore(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(t *testing.T, dir string)
		wantFindings bool
		wantContains string
	}{
		{
			name: "no finding when gitignore contains env/",
			setup: func(t *testing.T, dir string) {
				createEnvDir(t, dir)
				content := "node_modules/\n" + utils.EnvironmentFolder + "/\n*.log\n"
				if err := os.WriteFile(
					filepath.Join(dir, ".gitignore"), []byte(content), utils.FilePer,
				); err != nil {
					t.Fatal(err)
				}
			},
			wantFindings: false,
		},
		{
			name: "no finding when gitignore contains env without trailing slash",
			setup: func(t *testing.T, dir string) {
				createEnvDir(t, dir)
				content := utils.EnvironmentFolder + "\n"
				if err := os.WriteFile(
					filepath.Join(dir, ".gitignore"), []byte(content), utils.FilePer,
				); err != nil {
					t.Fatal(err)
				}
			},
			wantFindings: false,
		},
		{
			name: "no finding when entry has surrounding whitespace",
			setup: func(t *testing.T, dir string) {
				createEnvDir(t, dir)
				content := "  " + utils.EnvironmentFolder + "/  \n"
				if err := os.WriteFile(
					filepath.Join(dir, ".gitignore"), []byte(content), utils.FilePer,
				); err != nil {
					t.Fatal(err)
				}
			},
			wantFindings: false,
		},
		{
			name: "warns when gitignore does not contain env entry",
			setup: func(t *testing.T, dir string) {
				createEnvDir(t, dir)
				content := "node_modules/\n*.log\n"
				if err := os.WriteFile(
					filepath.Join(dir, ".gitignore"), []byte(content), utils.FilePer,
				); err != nil {
					t.Fatal(err)
				}
			},
			wantFindings: true,
			wantContains: "not gitignored",
		},
		{
			name: "warns when no gitignore file exists",
			setup: func(t *testing.T, dir string) {
				createEnvDir(t, dir)
			},
			wantFindings: true,
			wantContains: "not gitignored",
		},
		{
			name: "warns with empty gitignore",
			setup: func(t *testing.T, dir string) {
				createEnvDir(t, dir)
				if err := os.WriteFile(
					filepath.Join(dir, ".gitignore"), []byte(""), utils.FilePer,
				); err != nil {
					t.Fatal(err)
				}
			},
			wantFindings: true,
			wantContains: "not gitignored",
		},
		{
			name: "includes fix command in finding",
			setup: func(t *testing.T, dir string) {
				createEnvDir(t, dir)
			},
			wantFindings: true,
			wantContains: "echo",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tc.setup(t, tmpDir)
			restore := chdirTemp(t, tmpDir)
			defer restore()

			findings := checkGitignore()
			hasFindings := len(findings) > 0
			if hasFindings != tc.wantFindings {
				t.Errorf("checkGitignore() returned %d findings, wantFindings=%v",
					len(findings), tc.wantFindings)
			}
			if tc.wantContains != "" && hasFindings && !findingsContain(findings, tc.wantContains) {
				t.Errorf("findings %+v do not contain %q", findings, tc.wantContains)
			}
		})
	}
}

func TestCheckEnvPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission checks not reliable on Windows")
	}

	tests := []struct {
		name         string
		setup        func(t *testing.T, envDir string)
		wantFindings int
		wantContains string
	}{
		{
			name: "no findings when all env files have restrictive permissions",
			setup: func(t *testing.T, envDir string) {
				createEnvFile(t, envDir, "global", utils.SecretPer, "SECRET=value")
				createEnvFile(t, envDir, "prod", utils.SecretPer, "PROD_KEY=123")
			},
			wantFindings: 0,
		},
		{
			name: "warns per file with world-readable permissions",
			setup: func(t *testing.T, envDir string) {
				createEnvFile(t, envDir, "global", 0o644, "SECRET=leaked")
			},
			wantFindings: 1,
			wantContains: "global",
		},
		{
			name: "warns per file with group-readable permissions",
			setup: func(t *testing.T, envDir string) {
				createEnvFile(t, envDir, "staging", 0o640, "KEY=value")
			},
			wantFindings: 1,
			wantContains: "staging",
		},
		{
			name:         "no findings when env dir is empty",
			setup:        func(_ *testing.T, _ string) {},
			wantFindings: 0,
		},
		{
			name: "skips non-env files",
			setup: func(t *testing.T, envDir string) {
				path := filepath.Join(envDir, "notes.txt")
				if err := os.WriteFile(path, []byte("notes"), utils.SecretPer); err != nil {
					t.Fatal(err)
				}
				if err := os.Chmod(path, 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantFindings: 0,
		},
		{
			name: "skips subdirectories",
			setup: func(t *testing.T, envDir string) {
				if err := os.Mkdir(filepath.Join(envDir, "subdir"), utils.DirPer); err != nil {
					t.Fatal(err)
				}
			},
			wantFindings: 0,
		},
		{
			name: "warns only for loose files in mixed set",
			setup: func(t *testing.T, envDir string) {
				createEnvFile(t, envDir, "global", utils.SecretPer, "OK=value")
				createEnvFile(t, envDir, "leaky", 0o644, "LEAKED=yes")
			},
			wantFindings: 1,
			wantContains: "leaky",
		},
		{
			name: "warns for each loose file individually",
			setup: func(t *testing.T, envDir string) {
				createEnvFile(t, envDir, "first", 0o644, "A=1")
				createEnvFile(t, envDir, "second", 0o640, "B=2")
			},
			wantFindings: 2,
		},
		{
			name: "includes chmod fix command",
			setup: func(t *testing.T, envDir string) {
				createEnvFile(t, envDir, "global", 0o644, "KEY=val")
			},
			wantFindings: 1,
			wantContains: "chmod 600",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			envDir := createEnvDir(t, tmpDir)
			tc.setup(t, envDir)
			restore := chdirTemp(t, tmpDir)
			defer restore()

			findings := checkEnvPermissions(envDir)
			if len(findings) != tc.wantFindings {
				t.Errorf("checkEnvPermissions() returned %d findings, want %d: %+v",
					len(findings), tc.wantFindings, findings)
			}
			if tc.wantContains != "" && len(findings) > 0 &&
				!findingsContain(findings, tc.wantContains) {
				t.Errorf("findings %+v do not contain %q", findings, tc.wantContains)
			}
		})
	}
}

func TestCheckGitHistory(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	t.Run("no findings in non-git directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		createEnvDir(t, tmpDir)
		restore := chdirTemp(t, tmpDir)
		defer restore()

		findings := checkGitHistory()
		if len(findings) != 0 {
			t.Errorf("expected no findings in non-git dir, got: %+v", findings)
		}
	})

	t.Run("no findings when no env files in history", func(t *testing.T) {
		tmpDir := t.TempDir()
		envDir := createEnvDir(t, tmpDir)
		restore := chdirTemp(t, tmpDir)
		defer restore()

		gitInit(t)

		if err := os.WriteFile(
			filepath.Join(envDir, "readme.txt"), []byte("hello"), utils.FilePer,
		); err != nil {
			t.Fatal(err)
		}
		gitAddCommit(t, "initial commit")

		findings := checkGitHistory()
		if len(findings) != 0 {
			t.Errorf("expected no findings, got: %+v", findings)
		}
	})

	t.Run("warns when env files exist in history", func(t *testing.T) {
		tmpDir := t.TempDir()
		envDir := createEnvDir(t, tmpDir)
		restore := chdirTemp(t, tmpDir)
		defer restore()

		gitInit(t)

		createEnvFile(t, envDir, "global", utils.SecretPer, "SECRET=oops")
		gitAddCommit(t, "add secrets")

		findings := checkGitHistory()
		if len(findings) == 0 {
			t.Fatal("expected findings when env files in history, got none")
		}
		if !findingsContain(findings, "git history") {
			t.Errorf("expected message about git history, got: %+v", findings)
		}
		if !findingsContain(findings, "filter-repo") {
			t.Errorf("expected fix mentioning filter-repo, got: %+v", findings)
		}
	})
}

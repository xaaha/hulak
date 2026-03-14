package userflags

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

func TestCheckEnvDir(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(t *testing.T, dir string)
		expectedPassed bool
		wantContains   string
	}{
		{
			name: "passes when env directory exists",
			setup: func(t *testing.T, dir string) {
				createEnvDir(t, dir)
			},
			expectedPassed: true,
			wantContains:   "directory found",
		},
		{
			name:           "fails when env directory does not exist",
			setup:          func(_ *testing.T, _ string) {},
			expectedPassed: false,
			wantContains:   "directory not found",
		},
		{
			name: "fails when env is a file not a directory",
			setup: func(t *testing.T, dir string) {
				f, err := os.Create(filepath.Join(dir, utils.EnvironmentFolder))
				if err != nil {
					t.Fatal(err)
				}
				f.Close()
			},
			expectedPassed: false,
			wantContains:   "directory not found",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tc.setup(t, tmpDir)
			restore := chdirTemp(t, tmpDir)
			defer restore()

			result := checkEnvDir()
			if result.passed != tc.expectedPassed {
				t.Errorf("checkEnvDir().passed = %v, want %v (message: %s)",
					result.passed, tc.expectedPassed, result.message)
			}
			if tc.wantContains != "" && !strings.Contains(result.message, tc.wantContains) {
				t.Errorf("checkEnvDir().message = %q, want substring %q",
					result.message, tc.wantContains)
			}
		})
	}
}

func TestCheckGitignore(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(t *testing.T, dir string)
		expectedPassed bool
		wantContains   string
	}{
		{
			name: "passes when gitignore contains env/",
			setup: func(t *testing.T, dir string) {
				createEnvDir(t, dir)
				content := "node_modules/\n" + utils.EnvironmentFolder + "/\n*.log\n"
				if err := os.WriteFile(
					filepath.Join(dir, ".gitignore"), []byte(content), utils.FilePer,
				); err != nil {
					t.Fatal(err)
				}
			},
			expectedPassed: true,
			wantContains:   "is in .gitignore",
		},
		{
			name: "passes when gitignore contains env without trailing slash",
			setup: func(t *testing.T, dir string) {
				createEnvDir(t, dir)
				content := utils.EnvironmentFolder + "\n"
				if err := os.WriteFile(
					filepath.Join(dir, ".gitignore"), []byte(content), utils.FilePer,
				); err != nil {
					t.Fatal(err)
				}
			},
			expectedPassed: true,
			wantContains:   "is in .gitignore",
		},
		{
			name: "passes when entry has surrounding whitespace",
			setup: func(t *testing.T, dir string) {
				createEnvDir(t, dir)
				content := "  " + utils.EnvironmentFolder + "/  \n"
				if err := os.WriteFile(
					filepath.Join(dir, ".gitignore"), []byte(content), utils.FilePer,
				); err != nil {
					t.Fatal(err)
				}
			},
			expectedPassed: true,
			wantContains:   "is in .gitignore",
		},
		{
			name: "fails when gitignore exists but does not contain env entry",
			setup: func(t *testing.T, dir string) {
				createEnvDir(t, dir)
				content := "node_modules/\n*.log\n"
				if err := os.WriteFile(
					filepath.Join(dir, ".gitignore"), []byte(content), utils.FilePer,
				); err != nil {
					t.Fatal(err)
				}
			},
			expectedPassed: false,
			wantContains:   "is not in .gitignore",
		},
		{
			name: "fails when no gitignore file exists",
			setup: func(t *testing.T, dir string) {
				createEnvDir(t, dir)
			},
			expectedPassed: false,
			wantContains:   "no .gitignore found",
		},
		{
			name: "fails with empty gitignore",
			setup: func(t *testing.T, dir string) {
				createEnvDir(t, dir)
				if err := os.WriteFile(
					filepath.Join(dir, ".gitignore"), []byte(""), utils.FilePer,
				); err != nil {
					t.Fatal(err)
				}
			},
			expectedPassed: false,
			wantContains:   "is not in .gitignore",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tc.setup(t, tmpDir)
			restore := chdirTemp(t, tmpDir)
			defer restore()

			result := checkGitignore()
			if result.passed != tc.expectedPassed {
				t.Errorf("checkGitignore().passed = %v, want %v (message: %s)",
					result.passed, tc.expectedPassed, result.message)
			}
			if tc.wantContains != "" && !strings.Contains(result.message, tc.wantContains) {
				t.Errorf("checkGitignore().message = %q, want substring %q",
					result.message, tc.wantContains)
			}
		})
	}
}

func TestCheckEnvPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission checks not reliable on Windows")
	}

	tests := []struct {
		name           string
		setup          func(t *testing.T, envDir string)
		expectedPassed bool
		wantContains   string
	}{
		{
			name: "passes when all env files have restrictive permissions",
			setup: func(t *testing.T, envDir string) {
				createEnvFile(t, envDir, "global", utils.SecretPer, "SECRET=value")
				createEnvFile(t, envDir, "prod", utils.SecretPer, "PROD_KEY=123")
			},
			expectedPassed: true,
			wantContains:   "restrictive",
		},
		{
			name: "fails when env file has world-readable permissions",
			setup: func(t *testing.T, envDir string) {
				createEnvFile(t, envDir, "global", 0o644, "SECRET=leaked")
			},
			expectedPassed: false,
			wantContains:   "loose permissions",
		},
		{
			name: "fails when env file has group-readable permissions",
			setup: func(t *testing.T, envDir string) {
				createEnvFile(t, envDir, "staging", 0o640, "KEY=value")
			},
			expectedPassed: false,
			wantContains:   "loose permissions",
		},
		{
			name:           "passes when env dir is empty",
			setup:          func(_ *testing.T, _ string) {},
			expectedPassed: true,
			wantContains:   "restrictive",
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
			expectedPassed: true,
			wantContains:   "restrictive",
		},
		{
			name: "skips subdirectories",
			setup: func(t *testing.T, envDir string) {
				if err := os.Mkdir(filepath.Join(envDir, "subdir"), utils.DirPer); err != nil {
					t.Fatal(err)
				}
			},
			expectedPassed: true,
			wantContains:   "restrictive",
		},
		{
			name: "reports only loose files when mixed permissions exist",
			setup: func(t *testing.T, envDir string) {
				createEnvFile(t, envDir, "global", utils.SecretPer, "OK=value")
				createEnvFile(t, envDir, "leaky", 0o644, "LEAKED=yes")
			},
			expectedPassed: false,
			wantContains:   "leaky",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			envDir := createEnvDir(t, tmpDir)
			tc.setup(t, envDir)
			restore := chdirTemp(t, tmpDir)
			defer restore()

			result := checkEnvPermissions()
			if result.passed != tc.expectedPassed {
				t.Errorf("checkEnvPermissions().passed = %v, want %v (message: %s)",
					result.passed, tc.expectedPassed, result.message)
			}
			if tc.wantContains != "" && !strings.Contains(result.message, tc.wantContains) {
				t.Errorf("checkEnvPermissions().message = %q, want substring %q",
					result.message, tc.wantContains)
			}
		})
	}
}

func TestCheckGitHistory(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	t.Run("passes in non-git directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		createEnvDir(t, tmpDir)
		restore := chdirTemp(t, tmpDir)
		defer restore()

		result := checkGitHistory()
		if !result.passed {
			t.Errorf("expected pass in non-git dir, got: %s", result.message)
		}
		if !strings.Contains(result.message, "not a git repo") {
			t.Errorf("expected 'not a git repo' message, got: %s", result.message)
		}
	})

	t.Run("passes when no env files in history", func(t *testing.T) {
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

		result := checkGitHistory()
		if !result.passed {
			t.Errorf("expected pass with no env files in history, got: %s", result.message)
		}
	})

	t.Run("fails when env files exist in history", func(t *testing.T) {
		tmpDir := t.TempDir()
		envDir := createEnvDir(t, tmpDir)
		restore := chdirTemp(t, tmpDir)
		defer restore()

		gitInit(t)

		createEnvFile(t, envDir, "global", utils.SecretPer, "SECRET=oops")
		gitAddCommit(t, "add secrets")

		result := checkGitHistory()
		if result.passed {
			t.Errorf("expected fail when env files in history, got: %s", result.message)
		}
		if !strings.Contains(result.message, "git history") {
			t.Errorf("expected message about git history, got: %s", result.message)
		}
	})
}

func TestCheckEnvFileList(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(t *testing.T, envDir string)
		expectedPassed bool
		wantContains   string
	}{
		{
			name: "lists multiple env files",
			setup: func(t *testing.T, envDir string) {
				createEnvFile(t, envDir, "global", utils.SecretPer, "")
				createEnvFile(t, envDir, "prod", utils.SecretPer, "")
			},
			expectedPassed: true,
			wantContains:   "2 environment(s)",
		},
		{
			name: "lists single env file",
			setup: func(t *testing.T, envDir string) {
				createEnvFile(t, envDir, "global", utils.SecretPer, "")
			},
			expectedPassed: true,
			wantContains:   "1 environment(s)",
		},
		{
			name:           "passes with no env files",
			setup:          func(_ *testing.T, _ string) {},
			expectedPassed: true,
			wantContains:   "no environment files found",
		},
		{
			name: "shows env name without suffix",
			setup: func(t *testing.T, envDir string) {
				createEnvFile(t, envDir, "staging", utils.SecretPer, "")
			},
			expectedPassed: true,
			wantContains:   "staging",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			envDir := createEnvDir(t, tmpDir)
			tc.setup(t, envDir)
			restore := chdirTemp(t, tmpDir)
			defer restore()

			result := checkEnvFileList()
			if result.passed != tc.expectedPassed {
				t.Errorf("checkEnvFileList().passed = %v, want %v (message: %s)",
					result.passed, tc.expectedPassed, result.message)
			}
			if tc.wantContains != "" && !strings.Contains(result.message, tc.wantContains) {
				t.Errorf("checkEnvFileList().message = %q, want substring %q",
					result.message, tc.wantContains)
			}
		})
	}
}

func TestRunDoctor(t *testing.T) {
	tmpDir := t.TempDir()
	envDir := createEnvDir(t, tmpDir)
	createEnvFile(t, envDir, "global", utils.SecretPer, "KEY=value")

	content := utils.EnvironmentFolder + "/\n"
	if err := os.WriteFile(
		filepath.Join(tmpDir, ".gitignore"), []byte(content), utils.FilePer,
	); err != nil {
		t.Fatal(err)
	}

	restore := chdirTemp(t, tmpDir)
	defer restore()

	runDoctor()
}

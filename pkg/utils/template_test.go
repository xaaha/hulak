package utils

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestFileHasTemplateVars(t *testing.T) {
	tempDir := t.TempDir()
	// getFile refs resolve against the project root (the working directory), so
	// mark tempDir a project and run from it — mirroring a real run.
	if err := os.Mkdir(filepath.Join(tempDir, EnvironmentFolder), DirPer); err != nil {
		t.Fatalf("failed to create env dir: %v", err)
	}
	t.Chdir(tempDir)

	gqlPath := filepath.Join(tempDir, "query.graphql")
	err := os.WriteFile(gqlPath, []byte("query { user(id: {{.userId}}) { id } }"), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test gql file: %v", err)
	}
	unquotedPath := filepath.Join(tempDir, "unquoted.graphql")
	err = os.WriteFile(unquotedPath, []byte("query { viewer { id } } {{.needsEnv}}"), 0o600)
	if err != nil {
		t.Fatalf("Failed to create unquoted gql file: %v", err)
	}

	testCases := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "env_var_in_header",
			content:  "---\nkind: GraphQL\nurl: http://example.com/graphql\nheaders:\n  Authorization: \"Bearer {{.token}}\"\n",
			expected: true,
		},
		{
			name:     "env_var_with_spaces",
			content:  "---\nkind: GraphQL\nurl: http://example.com/graphql\nheaders:\n  Authorization: \"Bearer {{ .token }}\"\n",
			expected: true,
		},
		{
			name:     "env_var_in_url",
			content:  "---\nkind: GraphQL\nurl: \"{{.graphqlUrl}}\"\n",
			expected: true,
		},
		{
			name:     "env_var_in_body",
			content:  "---\nkind: GraphQL\nurl: http://example.com/graphql\nbody:\n  graphql:\n    variables:\n      name: \"{{.userName}}\"\n",
			expected: true,
		},
		{
			name:     "only_getFile_no_env_vars",
			content:  buildGetFileContent("plain.graphql"),
			expected: false,
		},
		{
			// getFile dumps the referenced file's raw content into context and
			// hulak never re-templates it (single-pass substitution), so an env
			// var inside the referenced file can never resolve. It must not force
			// env resolution.
			name:     "getFile_body_env_vars_do_not_count",
			content:  buildGetFileContent("query.graphql"),
			expected: false,
		},
		{
			name:     "getFile_unquoted_body_env_vars_do_not_count",
			content:  buildGetFileContentNoQuotes("unquoted.graphql"),
			expected: false,
		},
		{
			// A template var living only in a YAML comment never reaches runtime
			// substitution (the decoder drops comments), so it must not force env
			// resolution.
			name:     "env_var_only_in_full_line_comment",
			content:  "---\nkind: GraphQL\nurl: http://example.com/graphql\n# Authorization: Bearer {{.token}}\n",
			expected: false,
		},
		{
			name:     "env_var_only_in_inline_comment",
			content:  "---\nkind: GraphQL\nurl: http://example.com/graphql # {{.token}}\n",
			expected: false,
		},
		{
			name:     "only_getValueOf_no_env_vars",
			content:  buildGetValueOfContent("token", "auth.json"),
			expected: false,
		},
		{
			name:     "no_templates_at_all",
			content:  "---\nkind: GraphQL\nurl: http://example.com/graphql\nmethod: POST\n",
			expected: false,
		},
		{
			name:     "mixed_env_var_and_getFile",
			content:  "---\nkind: GraphQL\nurl: \"{{.baseUrl}}\"\nbody:\n  graphql:\n    query: '{{" + TemplateFuncGetFile + " \"test.graphql\"}}'\n",
			expected: true,
		},
		{
			name:     "multiple_env_vars",
			content:  "---\nkind: GraphQL\nurl: \"https://{{.domain}}/graphql\"\nheaders:\n  Authorization: \"Bearer {{.token}}\"\n",
			expected: true,
		},
		{
			name:     "os_func_only_no_env_loading_needed",
			content:  "---\nurl: http://example.com\nheaders:\n  Authorization: '{{os \"GITHUB_TOKEN\"}}'\n",
			expected: false,
		},
		{
			name:     "os_func_with_spaces_no_env_loading_needed",
			content:  "---\nurl: http://example.com\nheaders:\n  Authorization: '{{ os \"TOKEN\" }}'\n",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filePath := filepath.Join(tempDir, tc.name+".yaml")
			if tc.name == "only_getFile_no_env_vars" {
				plainPath := filepath.Join(tempDir, "plain.graphql")
				if err := os.WriteFile(plainPath, []byte("query { health }"), 0o600); err != nil {
					t.Fatalf("Failed to create plain gql file: %v", err)
				}
			}
			err := os.WriteFile(filePath, []byte(tc.content), 0o600)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}
			result := FileHasTemplateVars(filePath)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v for content:\n%s", tc.expected, result, tc.content)
			}
		})
	}
}

func TestFileHasTemplateVars_NonexistentFile(t *testing.T) {
	result := FileHasTemplateVars("/nonexistent/path/file.yaml")
	if result != false {
		t.Errorf("Expected false for nonexistent file, got true")
	}
}

func buildGetFileContent(path string) string {
	return "---\nkind: GraphQL\nurl: http://example.com/graphql\nbody:\n  graphql:\n    query: '{{" + TemplateFuncGetFile + " \"" + path + "\"}}'\n"
}

func buildGetFileContentNoQuotes(path string) string {
	return "---\nkind: GraphQL\nurl: http://example.com/graphql\nbody:\n  graphql:\n    query: '{{" + TemplateFuncGetFile + " " + path + "}}'\n"
}

func buildGetValueOfContent(key, fileName string) string {
	return "---\nkind: GraphQL\nurl: http://example.com/graphql\nheaders:\n  Authorization: '{{" + TemplateFuncGetValueOf + " \"" + key + "\" \"" + fileName + "\"}}'\n"
}

func TestReferencedFiles(t *testing.T) {
	// getFile paths are project-root-relative, resolved against the working
	// directory (as a real run does), so mark the temp dir a project and run
	// from it. EvalSymlinks so absolute paths compare equal on macOS.
	root := t.TempDir()
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, EnvironmentFolder), DirPer); err != nil {
		t.Fatalf("failed to create env dir: %v", err)
	}
	t.Chdir(root)

	// A request in a SUB-DIRECTORY referencing a root-relative .gql. This is the
	// case that used to double the sub-dir ("user_service/user_service/...").
	writeFile(t, filepath.Join(root, "collection", "users.gql"), "query { user { id } }")
	reqSubdir := filepath.Join(root, "user_service", "getUser.hk.yaml")
	writeFile(t, reqSubdir, buildGetFileContent("collection/users.gql"))

	// A request with no getFile references.
	reqNoDeps := filepath.Join(root, "plain.hk.yaml")
	writeFile(t, reqNoDeps, "---\nkind: API\nmethod: GET\nurl: http://example.com\n")

	// A request whose .gql itself references another file (transitive), also
	// root-relative.
	writeFile(t, filepath.Join(root, "frag", "inner.gql"), "fragment F on User { id }")
	writeFile(t, filepath.Join(root, "frag", "outer.gql"),
		"query { ...F } {{"+TemplateFuncGetFile+" \"frag/inner.gql\"}}")
	reqNested := filepath.Join(root, "nested.hk.yaml")
	writeFile(t, reqNested, buildGetFileContent("frag/outer.gql"))

	// A request referencing a .gql that does not exist yet.
	reqMissing := filepath.Join(root, "missing.hk.yaml")
	writeFile(t, reqMissing, buildGetFileContent("queries/DoesNotExist.gql"))

	tests := []struct {
		name string
		path string
		want []string
	}{
		{
			name: "subdir request resolves root-relative gql",
			path: reqSubdir,
			want: []string{filepath.Join(root, "collection", "users.gql")},
		},
		{
			name: "no dependencies",
			path: reqNoDeps,
			want: nil,
		},
		{
			name: "transitive dependency is followed",
			path: reqNested,
			want: []string{
				filepath.Join(root, "frag", "outer.gql"),
				filepath.Join(root, "frag", "inner.gql"),
			},
		},
		{
			name: "missing referenced file is still surfaced",
			path: reqMissing,
			want: []string{filepath.Join(root, "queries", "DoesNotExist.gql")},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ReferencedFiles(tc.path)
			if err != nil {
				t.Fatalf("ReferencedFiles(%q): unexpected error: %v", tc.path, err)
			}
			if !slices.Equal(got, tc.want) {
				t.Errorf("ReferencedFiles(%q) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}

func TestReferencedFiles_NonexistentRequestFile(t *testing.T) {
	if _, err := ReferencedFiles("/nonexistent/path/req.hk.yaml"); err == nil {
		t.Error("expected error for nonexistent request file, got nil")
	}
}

// writeFile writes content to path, creating parent directories as needed.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), DirPer); err != nil {
		t.Fatalf("MkdirAll %q: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), FilePer); err != nil {
		t.Fatalf("WriteFile %q: %v", path, err)
	}
}

func TestResolveFilePath(t *testing.T) {
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Fatal(err)
		}
	}()

	tmpDir := t.TempDir()
	// Resolve symlinks (macOS /var -> /private/var) so paths are consistent
	tmpDir, err = filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatalf("failed to resolve symlinks: %v", err)
	}

	// Create env/ to make it a valid hulak project root
	if err := os.Mkdir(filepath.Join(tmpDir, EnvironmentFolder), DirPer); err != nil {
		t.Fatalf("failed to create env dir: %v", err)
	}

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	// Create test files
	topFile := filepath.Join(tmpDir, "top.txt")
	if err := os.WriteFile(topFile, []byte("top"), FilePer); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	subDir := filepath.Join(tmpDir, "sub")
	if err := os.Mkdir(subDir, DirPer); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}
	nestedFile := filepath.Join(subDir, "nested.txt")
	if err := os.WriteFile(nestedFile, []byte("nested"), FilePer); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "empty path returns error",
			input:   "",
			wantErr: true,
		},
		{
			name:  "resolves absolute path that exists",
			input: topFile,
			want:  topFile,
		},
		{
			name:  "resolves relative path via project root",
			input: "top.txt",
			want:  filepath.Join(tmpDir, "top.txt"),
		},
		{
			name:  "resolves nested relative path via project root",
			input: "sub/nested.txt",
			want:  filepath.Join(tmpDir, "sub", "nested.txt"),
		},
		{
			name:  "resolves absolute nested path",
			input: nestedFile,
			want:  nestedFile,
		},
		{
			name:    "nonexistent file returns error",
			input:   "does_not_exist.txt",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveProjectFile(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ResolveProjectFile(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("ResolveProjectFile(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("ResolveProjectFile(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolveFilePath_FromChildDir(t *testing.T) {
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Fatal(err)
		}
	}()

	tmpDir := t.TempDir()
	tmpDir, err = filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatalf("failed to resolve symlinks: %v", err)
	}

	if err := os.Mkdir(filepath.Join(tmpDir, EnvironmentFolder), DirPer); err != nil {
		t.Fatalf("failed to create env dir: %v", err)
	}

	// Create a file at project root
	rootFile := filepath.Join(tmpDir, "root.txt")
	if err := os.WriteFile(rootFile, []byte("root"), FilePer); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Create a child directory and cd into it
	childDir := filepath.Join(tmpDir, "child")
	if err := os.Mkdir(childDir, DirPer); err != nil {
		t.Fatalf("failed to create child dir: %v", err)
	}
	if err := os.Chdir(childDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	// Relative path "root.txt" should resolve via project root, not cwd
	got, err := ResolveProjectFile("root.txt")
	if err != nil {
		t.Fatalf("ResolveProjectFile(\"root.txt\") from child dir: unexpected error: %v", err)
	}
	if got != rootFile {
		t.Errorf("ResolveProjectFile(\"root.txt\") from child dir = %q, want %q", got, rootFile)
	}
}

// TestResolveProjectFile_ChildCollision is the issue #239 regression: a
// relative getFile path must resolve against the project root even when a
// same-named file also exists under the cwd. The old cwd-first resolver picked
// the child copy and silently returned the wrong file.
func TestResolveProjectFile_ChildCollision(t *testing.T) {
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Fatal(err)
		}
	}()

	tmpDir := t.TempDir()
	tmpDir, err = filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatalf("failed to resolve symlinks: %v", err)
	}
	if err := os.Mkdir(filepath.Join(tmpDir, EnvironmentFolder), DirPer); err != nil {
		t.Fatalf("failed to create env dir: %v", err)
	}

	// Same relative name at both the project root and a child dir.
	rootFile := filepath.Join(tmpDir, "data.json")
	if err := os.WriteFile(rootFile, []byte("root"), FilePer); err != nil {
		t.Fatalf("failed to write root file: %v", err)
	}
	childDir := filepath.Join(tmpDir, "child")
	if err := os.Mkdir(childDir, DirPer); err != nil {
		t.Fatalf("failed to create child dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(childDir, "data.json"), []byte("child"), FilePer); err != nil {
		t.Fatalf("failed to write child file: %v", err)
	}
	if err := os.Chdir(childDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	got, err := ResolveProjectFile("data.json")
	if err != nil {
		t.Fatalf("ResolveProjectFile(\"data.json\") from child dir: unexpected error: %v", err)
	}
	if got != rootFile {
		t.Errorf("ResolveProjectFile(\"data.json\") = %q, want the project-root copy %q (cwd copy wrongly picked)",
			got, rootFile)
	}
}

func TestMapHasEnvVars(t *testing.T) {
	testCases := []struct {
		name     string
		data     map[string]any
		expected bool
	}{
		{
			name:     "empty_map",
			data:     map[string]any{},
			expected: false,
		},
		{
			name:     "no_template_vars",
			data:     map[string]any{"url": "http://example.com", "method": "GET"},
			expected: false,
		},
		{
			name:     "top_level_env_var",
			data:     map[string]any{"url": "{{.baseUrl}}"},
			expected: true,
		},
		{
			name: "nested_env_var",
			data: map[string]any{
				"headers": map[string]any{
					"Authorization": "Bearer {{.token}}",
				},
			},
			expected: true,
		},
		{
			name: "array_with_env_var",
			data: map[string]any{
				"items": []any{"plain", "{{.secret}}"},
			},
			expected: true,
		},
		{
			name: "getFile_only_no_env_var",
			data: map[string]any{
				"query": "{{" + TemplateFuncGetFile + " \"test.graphql\"}}",
			},
			expected: false,
		},
		{
			name: "getValueOf_only_no_env_var",
			data: map[string]any{
				"auth": "{{" + TemplateFuncGetValueOf + " \"token\" \"auth.json\"}}",
			},
			expected: false,
		},
		{
			name: "deeply_nested_env_var",
			data: map[string]any{
				"body": map[string]any{
					"graphql": map[string]any{
						"variables": map[string]any{
							"name": "{{.userName}}",
						},
					},
				},
			},
			expected: true,
		},
		{
			name:     "os_func_in_value_no_env_loading_needed",
			data:     map[string]any{"auth": `{{os "GITHUB_TOKEN"}}`},
			expected: false,
		},
		{
			name: "os_func_nested_no_env_loading_needed",
			data: map[string]any{
				"headers": map[string]any{
					"X-Token": `{{os "CI_TOKEN"}}`,
				},
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := MapHasEnvVars(tc.data)
			if result != tc.expected {
				t.Errorf("MapHasEnvVars() = %v, want %v", result, tc.expected)
			}
		})
	}
}

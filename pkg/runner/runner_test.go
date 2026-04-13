package runner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateFilePathList_FpOnly(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.hk.yaml")
	if err := os.WriteFile(tmpFile, []byte("method: GET\nurl: http://example.com"), 0o600); err != nil {
		t.Fatal(err)
	}

	list, err := generateFilePathList("", tmpFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 1 || list[0] != tmpFile {
		t.Errorf("expected [%s], got %v", tmpFile, list)
	}
}

func TestGenerateFilePathList_BothEmpty(t *testing.T) {
	_, err := generateFilePathList("", "")
	if err == nil {
		t.Fatal("expected error when both fileName and fp are empty")
	}
}

func TestDiscoverFilePaths_FpReturnsFileList(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "req.hk.yaml")
	if err := os.WriteFile(tmpFile, []byte("method: GET\nurl: http://example.com"), 0o600); err != nil {
		t.Fatal(err)
	}

	fileList, concurrent, sequential := discoverFilePaths(
		"",      // fileName
		tmpFile, // fp
		"",      // dir
		"",      // dirseq
		false,   // hasDirFlags
	)

	if len(fileList) != 1 || fileList[0] != tmpFile {
		t.Errorf("fileList = %v, want [%s]", fileList, tmpFile)
	}
	if len(concurrent) != 0 {
		t.Errorf("concurrent should be empty, got %v", concurrent)
	}
	if len(sequential) != 0 {
		t.Errorf("sequential should be empty, got %v", sequential)
	}
}

func TestDiscoverFilePaths_DirReturnsConcurrent(t *testing.T) {
	tmpDir := t.TempDir()
	for _, name := range []string{"a.yaml", "b.yaml"} {
		content := "method: GET\nurl: http://example.com"
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	fileList, concurrent, sequential := discoverFilePaths(
		"",     // fileName
		"",     // fp
		tmpDir, // dir
		"",     // dirseq
		true,   // hasDirFlags
	)

	if len(fileList) != 0 {
		t.Errorf("fileList should be empty, got %v", fileList)
	}
	if len(concurrent) != 2 {
		t.Errorf("expected 2 concurrent files, got %d: %v", len(concurrent), concurrent)
	}
	if len(sequential) != 0 {
		t.Errorf("sequential should be empty, got %v", sequential)
	}
}

func TestDiscoverFilePaths_DirseqReturnsSequential(t *testing.T) {
	tmpDir := t.TempDir()
	for _, name := range []string{"a.yaml", "b.yaml"} {
		content := "method: GET\nurl: http://example.com"
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	fileList, concurrent, sequential := discoverFilePaths(
		"",     // fileName
		"",     // fp
		"",     // dir
		tmpDir, // dirseq
		true,   // hasDirFlags
	)

	if len(fileList) != 0 {
		t.Errorf("fileList should be empty, got %v", fileList)
	}
	if len(concurrent) != 0 {
		t.Errorf("concurrent should be empty, got %v", concurrent)
	}
	if len(sequential) != 2 {
		t.Errorf("expected 2 sequential files, got %d: %v", len(sequential), sequential)
	}
}

func TestContainsTemplateVars_NoTemplates(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "plain.yaml")
	if err := os.WriteFile(tmpFile, []byte("method: GET\nurl: http://example.com"), 0o600); err != nil {
		t.Fatal(err)
	}

	if containsTemplateVars([]string{tmpFile}) {
		t.Error("expected false for file without template vars")
	}
}

func TestContainsTemplateVars_WithTemplates(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "templated.yaml")
	if err := os.WriteFile(tmpFile, []byte("method: GET\nurl: '{{.apiUrl}}'"), 0o600); err != nil {
		t.Fatal(err)
	}

	if !containsTemplateVars([]string{tmpFile}) {
		t.Error("expected true for file with template vars")
	}
}

func TestContainsTemplateVars_EmptyList(t *testing.T) {
	if containsTemplateVars(nil) {
		t.Error("expected false for empty list")
	}
}

func TestDiscoverFilePaths_EmptyInputs(t *testing.T) {
	fileList, concurrent, sequential := discoverFilePaths(
		"", "", "", "", false,
	)

	if len(fileList) != 0 {
		t.Errorf("fileList should be empty, got %v", fileList)
	}
	if len(concurrent) != 0 {
		t.Errorf("concurrent should be empty, got %v", concurrent)
	}
	if len(sequential) != 0 {
		t.Errorf("sequential should be empty, got %v", sequential)
	}
}

func TestDiscoverFilePaths_FpAndDirTogether(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "single.yaml")
	content := "method: GET\nurl: http://example.com"
	if err := os.WriteFile(tmpFile, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	fileList, concurrent, sequential := discoverFilePaths(
		"",      // fileName
		tmpFile, // fp
		tmpDir,  // dir
		"",      // dirseq
		true,    // hasDirFlags
	)

	if len(fileList) != 1 {
		t.Errorf("fileList should have 1 entry, got %v", fileList)
	}
	if len(concurrent) != 1 {
		t.Errorf("concurrent should have 1 entry from dir, got %v", concurrent)
	}
	if len(sequential) != 0 {
		t.Errorf("sequential should be empty, got %v", sequential)
	}
}

package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateFilePath(t *testing.T) {
	// Test case with a known relative file path
	expected, _ := os.Getwd()
	expected = filepath.Join(expected, "testfile.txt")
	result, err := CreateFilePath("testfile.txt")
	if err != nil {
		t.Errorf("CreateFilePath returned an error: %v", err)
	}
	if result != expected {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

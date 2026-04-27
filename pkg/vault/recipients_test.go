package vault

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xaaha/hulak/pkg/utils"
)

func TestRecipientsFilePath(t *testing.T) {
	projectDir := setupHulakProject(t)

	got, err := RecipientsFilePath()
	if err != nil {
		t.Fatalf("RecipientsFilePath() error: %v", err)
	}

	want := filepath.Join(projectDir, utils.HiddenProjectName, utils.RecipientsFile)
	if got != want {
		t.Errorf("RecipientsFilePath() = %q, want %q", got, want)
	}
}

func TestLoadRecipients(t *testing.T) {
	t.Run("reads multiple recipients", func(t *testing.T) {
		projectDir := setupHulakProject(t)

		// Generate two keypairs and write them to recipients.txt
		key1, err := GenerateKeyPair()
		if err != nil {
			t.Fatalf("GenerateKeyPair: %v", err)
		}
		key2, err := GenerateKeyPair()
		if err != nil {
			t.Fatalf("GenerateKeyPair: %v", err)
		}

		content := "# Alice\n" + key1.Recipient.String() + "\n# Bob\n" + key2.Recipient.String() + "\n"
		recipientsPath := filepath.Join(projectDir, utils.HiddenProjectName, utils.RecipientsFile)
		if err := os.WriteFile(recipientsPath, []byte(content), utils.FilePer); err != nil {
			t.Fatalf("write recipients: %v", err)
		}

		recipients, err := LoadRecipients()
		if err != nil {
			t.Fatalf("LoadRecipients() error: %v", err)
		}
		if len(recipients) != 2 {
			t.Errorf("LoadRecipients() count = %d, want 2", len(recipients))
		}
	})

	t.Run("errors on missing file", func(t *testing.T) {
		setupHulakProject(t) // no recipients.txt created

		_, err := LoadRecipients()
		if err == nil {
			t.Error("LoadRecipients() should error on missing file")
		}
	})

	t.Run("errors on empty file with comments only", func(t *testing.T) {
		projectDir := setupHulakProject(t)

		content := "# just a comment\n# another comment\n"
		recipientsPath := filepath.Join(projectDir, utils.HiddenProjectName, utils.RecipientsFile)
		if err := os.WriteFile(recipientsPath, []byte(content), utils.FilePer); err != nil {
			t.Fatalf("write recipients: %v", err)
		}

		_, err := LoadRecipients()
		if err == nil {
			t.Error("LoadRecipients() should error when file has no recipients")
		}
	})
}

func TestParseRecipientsFileContent(t *testing.T) {
	t.Run("parses entries with names from comments", func(t *testing.T) {
		key1, _ := GenerateKeyPair()
		key2, _ := GenerateKeyPair()

		content := "# Alice (added 2026-04-27)\n" + key1.Recipient.String() + "\n" +
			"# Bob (added 2026-04-27)\n" + key2.Recipient.String() + "\n"

		entries, err := ParseRecipientsFileContent([]byte(content))
		if err != nil {
			t.Fatalf("ParseRecipientsFileContent() error: %v", err)
		}
		if len(entries) != 2 {
			t.Fatalf("ParseRecipientsFileContent() count = %d, want 2", len(entries))
		}

		if entries[0].Key != key1.Recipient.String() {
			t.Errorf("entry[0].Key = %q, want %q", entries[0].Key, key1.Recipient.String())
		}
		if entries[0].Name != "Alice (added 2026-04-27)" {
			t.Errorf("entry[0].Name = %q, want %q", entries[0].Name, "Alice (added 2026-04-27)")
		}

		if entries[1].Key != key2.Recipient.String() {
			t.Errorf("entry[1].Key = %q, want %q", entries[1].Key, key2.Recipient.String())
		}
		if entries[1].Name != "Bob (added 2026-04-27)" {
			t.Errorf("entry[1].Name = %q, want %q", entries[1].Name, "Bob (added 2026-04-27)")
		}
	})

	t.Run("key without preceding comment has empty name", func(t *testing.T) {
		key1, _ := GenerateKeyPair()

		content := key1.Recipient.String() + "\n"

		entries, err := ParseRecipientsFileContent([]byte(content))
		if err != nil {
			t.Fatalf("ParseRecipientsFileContent() error: %v", err)
		}
		if len(entries) != 1 {
			t.Fatalf("ParseRecipientsFileContent() count = %d, want 1", len(entries))
		}

		if entries[0].Key != key1.Recipient.String() {
			t.Errorf("entry[0].Key = %q, want %q", entries[0].Key, key1.Recipient.String())
		}
		if entries[0].Name != "" {
			t.Errorf("entry[0].Name = %q, want empty", entries[0].Name)
		}
	})

	t.Run("errors on empty content", func(t *testing.T) {
		_, err := ParseRecipientsFileContent([]byte(""))
		if err == nil {
			t.Error("ParseRecipientsFileContent() should error on empty content")
		}
	})
}

func TestSaveRecipients(t *testing.T) {
	t.Run("writes file and round-trips", func(t *testing.T) {
		projectDir := setupHulakProject(t)

		key1, _ := GenerateKeyPair()
		key2, _ := GenerateKeyPair()

		entries := []RecipientEntry{
			{Key: key1.Recipient.String(), Name: "Alice (added 2026-04-27)"},
			{Key: key2.Recipient.String(), Name: "Bob (added 2026-04-27)"},
		}

		if err := SaveRecipients(entries); err != nil {
			t.Fatalf("SaveRecipients() error: %v", err)
		}

		// Verify file was created
		recipientsPath := filepath.Join(projectDir, utils.HiddenProjectName, utils.RecipientsFile)
		data, err := os.ReadFile(recipientsPath)
		if err != nil {
			t.Fatalf("read recipients file: %v", err)
		}

		// Round-trip: parse it back
		parsed, err := ParseRecipientsFileContent(data)
		if err != nil {
			t.Fatalf("ParseRecipientsFileContent() round-trip error: %v", err)
		}
		if len(parsed) != 2 {
			t.Fatalf("round-trip count = %d, want 2", len(parsed))
		}
		if parsed[0].Key != key1.Recipient.String() {
			t.Errorf("round-trip[0].Key = %q, want %q", parsed[0].Key, key1.Recipient.String())
		}
		if parsed[0].Name != "Alice (added 2026-04-27)" {
			t.Errorf("round-trip[0].Name = %q, want %q", parsed[0].Name, "Alice (added 2026-04-27)")
		}
		if parsed[1].Key != key2.Recipient.String() {
			t.Errorf("round-trip[1].Key = %q, want %q", parsed[1].Key, key2.Recipient.String())
		}
		if parsed[1].Name != "Bob (added 2026-04-27)" {
			t.Errorf("round-trip[1].Name = %q, want %q", parsed[1].Name, "Bob (added 2026-04-27)")
		}
	})

	t.Run("entry without name omits comment", func(t *testing.T) {
		projectDir := setupHulakProject(t)

		key1, _ := GenerateKeyPair()

		entries := []RecipientEntry{
			{Key: key1.Recipient.String(), Name: ""},
		}

		if err := SaveRecipients(entries); err != nil {
			t.Fatalf("SaveRecipients() error: %v", err)
		}

		recipientsPath := filepath.Join(projectDir, utils.HiddenProjectName, utils.RecipientsFile)
		data, err := os.ReadFile(recipientsPath)
		if err != nil {
			t.Fatalf("read recipients file: %v", err)
		}

		content := string(data)
		if strings.Contains(content, "#") {
			t.Errorf("expected no comment line for nameless entry, got:\n%s", content)
		}
		if !strings.Contains(content, key1.Recipient.String()) {
			t.Errorf("expected key in file, got:\n%s", content)
		}
	})
}

func TestFormatRecipientName(t *testing.T) {
	t.Run("formats name with date", func(t *testing.T) {
		got := FormatRecipientName("Alice")
		if !strings.HasPrefix(got, "Alice (added ") {
			t.Errorf("FormatRecipientName(\"Alice\") = %q, want prefix \"Alice (added \"", got)
		}
		if !strings.HasSuffix(got, ")") {
			t.Errorf("FormatRecipientName(\"Alice\") = %q, want suffix \")\"", got)
		}
	})

	t.Run("empty name returns empty string", func(t *testing.T) {
		got := FormatRecipientName("")
		if got != "" {
			t.Errorf("FormatRecipientName(\"\") = %q, want empty", got)
		}
	})
}

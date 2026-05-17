package vault

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"filippo.io/age"

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

func TestAddRecipientEntry(t *testing.T) {
	t.Run("adds new key", func(t *testing.T) {
		key1, _ := GenerateKeyPair()
		key2, _ := GenerateKeyPair()

		existing := []RecipientEntry{{Key: key1.Recipient.String(), Name: "Alice"}}
		got, err := AddRecipientEntry(existing, key2.Recipient.String(), "Bob", false)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("got %d entries, want 2", len(got))
		}
	})

	t.Run("rejects duplicate key", func(t *testing.T) {
		key1, _ := GenerateKeyPair()
		existing := []RecipientEntry{{Key: key1.Recipient.String(), Name: "Alice"}}
		_, err := AddRecipientEntry(existing, key1.Recipient.String(), "Alice Again", false)
		if err == nil {
			t.Fatal("expected duplicate error")
		}
	})

	t.Run("rejects malformed key", func(t *testing.T) {
		_, err := AddRecipientEntry(nil, "not-a-valid-key", "Bad", false)
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("no name produces empty Name field", func(t *testing.T) {
		key1, _ := GenerateKeyPair()
		got, err := AddRecipientEntry(nil, key1.Recipient.String(), "", false)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		if got[0].Name != "" {
			t.Errorf("expected empty Name, got %q", got[0].Name)
		}
	})
}

func TestRemoveRecipientEntry(t *testing.T) {
	key1, _ := GenerateKeyPair()
	key2, _ := GenerateKeyPair()

	entries := []RecipientEntry{
		{Key: key1.Recipient.String(), Name: "Alice"},
		{Key: key2.Recipient.String(), Name: "Bob"},
	}

	t.Run("removes by key", func(t *testing.T) {
		got, removed, err := RemoveRecipientEntry(entries, key1.Recipient.String())
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		if !removed {
			t.Fatal("expected removed=true")
		}
		if len(got) != 1 {
			t.Fatalf("got %d, want 1", len(got))
		}
		if got[0].Key != key2.Recipient.String() {
			t.Error("wrong entry removed")
		}
	})

	t.Run("removes by name", func(t *testing.T) {
		got, removed, err := RemoveRecipientEntry(entries, "Bob")
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		if !removed {
			t.Fatal("expected removed=true")
		}
		if len(got) != 1 {
			t.Fatalf("got %d, want 1", len(got))
		}
	})

	t.Run("refuses to remove last recipient", func(t *testing.T) {
		single := []RecipientEntry{{Key: key1.Recipient.String(), Name: "Alice"}}
		_, _, err := RemoveRecipientEntry(single, key1.Recipient.String())
		if err == nil {
			t.Fatal("expected error when removing last recipient")
		}
	})

	t.Run("no-op for unknown query", func(t *testing.T) {
		got, removed, err := RemoveRecipientEntry(entries, "unknown-query")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if removed {
			t.Fatal("expected removed=false")
		}
		if len(got) != 2 {
			t.Fatal("entries should be unchanged")
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

func TestSwapRecipientKey(t *testing.T) {
	id1, _ := age.GenerateX25519Identity()
	id2, _ := age.GenerateX25519Identity()
	id3, _ := age.GenerateX25519Identity()
	oldKey := id1.Recipient().String()
	newKey := id2.Recipient().String()
	otherKey := id3.Recipient().String()

	t.Run("swaps single matching entry", func(t *testing.T) {
		entries := []RecipientEntry{
			{Key: oldKey, Name: "me (added 2026-04-01)"},
		}
		got, count, err := SwapRecipientKey(entries, oldKey, newKey, "me")
		if err != nil {
			t.Fatalf("SwapRecipientKey error: %v", err)
		}
		if count != 1 {
			t.Errorf("replaced count = %d, want 1", count)
		}
		if len(got) != 1 {
			t.Fatalf("len = %d, want 1", len(got))
		}
		if got[0].Key != newKey {
			t.Errorf("key = %q, want %q", got[0].Key, newKey)
		}
		if !strings.Contains(got[0].Name, "me") {
			t.Errorf("name = %q, want it to contain 'me'", got[0].Name)
		}
	})

	t.Run("swaps multiple matching entries into one", func(t *testing.T) {
		entries := []RecipientEntry{
			{Key: oldKey, Name: "me-laptop"},
			{Key: otherKey, Name: "teammate"},
			{Key: oldKey, Name: "me-desktop"},
		}
		got, count, err := SwapRecipientKey(entries, oldKey, newKey, "me")
		if err != nil {
			t.Fatalf("SwapRecipientKey error: %v", err)
		}
		if count != 2 {
			t.Errorf("replaced count = %d, want 2", count)
		}
		// Should have 2 entries: new key (replaces first match) + teammate
		if len(got) != 2 {
			t.Fatalf("len = %d, want 2", len(got))
		}
		// Teammate should be preserved
		found := false
		for _, e := range got {
			if e.Key == otherKey {
				found = true
			}
		}
		if !found {
			t.Error("teammate key should be preserved")
		}
	})

	t.Run("errors when old key not found", func(t *testing.T) {
		entries := []RecipientEntry{
			{Key: otherKey, Name: "teammate"},
		}
		_, _, err := SwapRecipientKey(entries, oldKey, newKey, "me")
		if err == nil {
			t.Error("should error when old key not in entries")
		}
	})
}

func TestAddRecipientAndReencrypt(t *testing.T) {
	projectDir := setupHulakProject(t)
	cfgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgDir)
	t.Setenv("HULAK_MASTER_KEY", "")
	t.Setenv("HULAK_SSH_IDENTITY", "")
	t.Setenv("HOME", t.TempDir())

	// Bootstrap: generate key, write recipients + store, register as identity
	key1, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}
	if err := SetIdentity(key1.Identity.String()); err != nil {
		t.Fatalf("SetIdentity: %v", err)
	}
	if err := SaveRecipients([]RecipientEntry{
		{Key: key1.Recipient.String(), Name: "owner"},
	}); err != nil {
		t.Fatalf("SaveRecipients: %v", err)
	}
	store := &Store{Envs: map[string]Env{"global": {"KEY": "val"}}}
	if err := WriteStore(store, key1.Recipient); err != nil {
		t.Fatalf("WriteStore: %v", err)
	}

	t.Run("adds new recipient and re-encrypts", func(t *testing.T) {
		key2, _ := GenerateKeyPair()
		added, err := AddRecipientAndReencrypt(key2.Recipient.String(), "teammate")
		if err != nil {
			t.Fatalf("AddRecipientAndReencrypt: %v", err)
		}
		if !added {
			t.Fatal("expected added=true")
		}

		// New key can decrypt
		got, err := DecryptStore(key2.Identity)
		if err != nil {
			t.Fatalf("ReadStore with new key: %v", err)
		}
		if got.GetEnv("global")["KEY"] != "val" {
			t.Error("store data lost after re-encryption")
		}

		// Old key still works
		if _, err := DecryptStore(key1.Identity); err != nil {
			t.Fatalf("ReadStore with original key: %v", err)
		}

		// Verify recipients.txt has both
		recipPath := filepath.Join(projectDir, utils.HiddenProjectName, utils.RecipientsFile)
		data, err := os.ReadFile(recipPath)
		if err != nil {
			t.Fatalf("read recipients: %v", err)
		}
		if !strings.Contains(string(data), key2.Recipient.String()) {
			t.Error("new key missing from recipients.txt")
		}
	})

	t.Run("returns false for duplicate", func(t *testing.T) {
		added, err := AddRecipientAndReencrypt(key1.Recipient.String(), "owner")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if added {
			t.Error("expected added=false for duplicate")
		}
	})
}

package vault

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"filippo.io/age"

	"github.com/xaaha/hulak/pkg/utils"
)

// Contains recipients.txt file I/O for multi-recipient encryption.

// RecipientEntry pairs an age public key string with an optional human label.
type RecipientEntry struct {
	Key  string // age1... public key
	Name string // from the # comment line (empty if none)
}

// RecipientsFilePath returns the absolute path to .hulak/recipients.txt.
func RecipientsFilePath() (string, error) {
	markerPath, err := utils.GetProjectMarker()
	if err != nil {
		return "", err
	}
	return filepath.Join(markerPath, utils.RecipientsFile), nil
}

// LoadRecipients reads .hulak/recipients.txt and returns age.Recipient values.
// Supports both age (age1...) and SSH (ssh-ed25519, ssh-rsa) key formats.
// Returns error if the file is missing or contains zero recipients.
func LoadRecipients() ([]age.Recipient, error) {
	path, err := RecipientsFilePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read recipients file: %w", err)
	}

	entries, err := ParseRecipientsFileContent(data)
	if err != nil {
		return nil, err
	}

	return RecipientsFromEntries(entries)
}

// ParseRecipientsFileContent reads raw bytes and returns structured entries
// (key + name from preceding # comment). Used by list/remove-recipient.
// age.ParseRecipients discards comments, so this does its own line-by-line parse.
func ParseRecipientsFileContent(data []byte) ([]RecipientEntry, error) {
	var entries []RecipientEntry
	var pendingName string

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			continue
		}

		// if comment, starting with #
		if comment, ok := strings.CutPrefix(line, utils.Comment); ok {
			pendingName = strings.TrimSpace(comment)
			continue
		}

		// Non-comment, non-blank line is a key
		// It's intentional that we are not checking age1
		entries = append(entries, RecipientEntry{
			Key:  line,
			Name: pendingName,
		})
		pendingName = ""
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan recipients file: %w", err)
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("recipients file contains no valid entries")
	}

	return entries, nil
}

// SaveRecipients writes entries to .hulak/recipients.txt.
// Each entry gets an optional "# Name" comment header followed by the key.
func SaveRecipients(entries []RecipientEntry) error {
	path, err := RecipientsFilePath()
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	for i, entry := range entries {
		if entry.Name != "" {
			fmt.Fprintf(&buf, "# %s\n", entry.Name)
		}
		buf.WriteString(entry.Key)
		buf.WriteByte('\n')
		// Blank line between entries for readability, but not after the last one
		if i < len(entries)-1 {
			buf.WriteByte('\n')
		}
	}

	return os.WriteFile(path, buf.Bytes(), utils.FilePer)
}

// AddRecipientEntry validates the new key, checks for duplicates, and returns
// the updated entry list. Does not write to disk — caller does that.
// Set allowRSA to true to accept ssh-rsa keys.
func AddRecipientEntry(entries []RecipientEntry, pubKey, name string, allowRSA bool) ([]RecipientEntry, error) {
	if _, _, err := ParseRecipientKey(pubKey, allowRSA); err != nil {
		return nil, err
	}

	for _, e := range entries {
		if e.Key == pubKey {
			return nil, fmt.Errorf("key already in recipients list")
		}
	}

	return append(entries, RecipientEntry{
		Key:  pubKey,
		Name: FormatRecipientName(name),
	}), nil
}

// AddRecipientAndReencrypt adds a public key to recipients.txt (if not already
// present) and re-encrypts the store to all recipients. Uses the provided
// identity for decryption. Returns true if a new recipient was added.
// This is the shared core for `init` additive flow and `add-recipient`.
func AddRecipientAndReencrypt(pubKey, name string, identity age.Identity) (bool, error) {
	recipPath, err := RecipientsFilePath()
	if err != nil {
		return false, err
	}
	data, err := os.ReadFile(recipPath)
	if err != nil {
		return false, fmt.Errorf("failed to read recipients: %w", err)
	}
	entries, err := ParseRecipientsFileContent(data)
	if err != nil {
		return false, err
	}

	newEntries, err := AddRecipientEntry(entries, pubKey, name, true)
	if err != nil {
		if strings.Contains(err.Error(), "already in recipients") {
			return false, nil
		}
		return false, err
	}

	store, err := ReadStore(identity)
	if err != nil {
		return false, err
	}

	recipients, err := RecipientsFromEntries(newEntries)
	if err != nil {
		return false, err
	}
	if err := WriteStore(store, recipients...); err != nil {
		return false, err
	}
	if err := SaveRecipients(newEntries); err != nil {
		return false, err
	}

	return true, nil
}

// RemoveRecipientEntry removes entries matching query (by full key string or
// by name substring). Returns error if removing would leave zero recipients.
// Returns the original list unchanged if no match found.
func RemoveRecipientEntry(entries []RecipientEntry, query string) ([]RecipientEntry, bool, error) {
	var remaining []RecipientEntry
	var removed bool

	for _, e := range entries {
		// if entry matches the query, add to skip and don't add to remaining
		if e.Key == query || (e.Name != "" && strings.Contains(e.Name, query)) {
			removed = true
			continue
		}
		remaining = append(remaining, e)
	}

	if !removed {
		return entries, false, nil
	}

	if len(remaining) == 0 {
		return entries, false, fmt.Errorf(
			"refusing to remove the last recipient — the store would become unrecoverable. " +
				"Add another recipient first, or delete .hulak/store.age manually",
		)
	}

	return remaining, true, nil
}

// SwapRecipientKey replaces all entries matching oldKey with a single entry for
// newKey. Preserves the name from the first matched entry (with updated date).
// Returns the updated entries, the count of replaced keys, and an error if
// oldKey was not found.
func SwapRecipientKey(
	entries []RecipientEntry,
	oldKey, newKey, name string,
) ([]RecipientEntry, int, error) {
	// Count matches first
	replaced := 0
	for _, e := range entries {
		if e.Key == oldKey {
			replaced++
		}
	}
	if replaced == 0 {
		return nil, 0, fmt.Errorf(
			"current public key not found in recipients.txt — cannot determine which key to replace",
		)
	}

	// Build result: replace first match, skip the rest
	result := make([]RecipientEntry, 0, len(entries)-replaced+1)
	swapped := false
	for _, entry := range entries {
		if entry.Key == oldKey {
			if !swapped {
				result = append(result, RecipientEntry{
					Key:  newKey,
					Name: FormatRecipientName(name),
				})
				swapped = true
			}
			continue
		}
		result = append(result, entry)
	}

	return result, replaced, nil
}

// RecipientsFromEntries converts RecipientEntry slice to age.Recipient slice.
// Accepts all key types already present in recipients.txt (including ssh-rsa),
// since they were validated at add time.
func RecipientsFromEntries(entries []RecipientEntry) ([]age.Recipient, error) {
	recipients := make([]age.Recipient, len(entries))
	for i, e := range entries {
		r, _, err := ParseRecipientKey(e.Key, true)
		if err != nil {
			return nil, fmt.Errorf("invalid key in recipients.txt line %d: %w", i+1, err)
		}
		recipients[i] = r
	}
	return recipients, nil
}

// recipientNameSuffix is the format string appended by FormatRecipientName.
// ParseRecipientName strips it. Keep them in sync by using this constant.
const recipientNameSuffix = " (added "

// FormatRecipientName builds a comment label with today's date.
// Empty name returns empty string.
func FormatRecipientName(name string) string {
	if name == "" {
		return ""
	}
	today := time.Now().Format(time.DateOnly)
	return fmt.Sprintf("%s%s%s)", name, recipientNameSuffix, today)
}

// ParseRecipientName strips the date suffix added by FormatRecipientName.
// Returns the original name if no suffix is found.
func ParseRecipientName(formatted string) string {
	if idx := strings.Index(formatted, recipientNameSuffix); idx >= 0 {
		return formatted[:idx]
	}
	return formatted
}

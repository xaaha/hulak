// Contains command factory and handler for hulak secrets edit.
package userflags

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/xaaha/hulak/pkg/userFlags/cli"
	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/vault"
)

// newEnvEditCmd returns the command struct for `hulak secrets edit`.
func newEnvEditCmd() *cli.Command {
	editFs := flag.NewFlagSet("env edit", flag.ContinueOnError)
	editEnv := registerEnvFlag(editFs, "", "Environment to edit (omit to pick interactively)")

	return &cli.Command{
		Name:  "edit",
		Short: "Edit secrets interactively",
		Long:  "Open the decrypted environment in $EDITOR (falls back to vi).\n\nWhen --env is omitted you'll be prompted to pick an environment from a TUI list,\nthe same flow as `hulak run`. To create a brand-new environment, pass --env\nexplicitly with the new name.\n\nThe decrypted JSON is written to a temp file with 0600 permissions inside .hulak/.\nWhat you save in the editor REPLACES the entire environment — keys you delete\nfrom the file are deleted from the store. Other environments in the store are\nuntouched. The store is re-encrypted atomically.\n\nIf the editor exits non-zero or the file is unchanged, no write occurs.",
		Flags: editFs,
		Examples: []*utils.CommandHelp{
			{
				Command:     "hulak secrets edit",
				Description: "Pick an environment from the TUI, then edit",
			},
			{
				Command:     "hulak secrets edit --env prod",
				Description: "Edit prod directly (skip the picker)",
			},
			{
				Command:     "hulak secrets edit --env new_one",
				Description: "Create a brand-new environment by name",
			},
			{
				Command:     "EDITOR=nvim hulak secrets edit --env staging",
				Description: "Use a specific editor",
			},
			{
				Command:     "EDITOR=\"zed --wait\" hulak secrets edit --env staging",
				Description: "GUI editors need a wait flag so hulak waits until you save (zed --wait, code -w)",
			},
		},
		Run: func(args []string) error { return runEnvEdit(args, *editEnv) },
	}
}

// runEnvEdit handles `hulak secrets edit`. Decrypts the named environment to a
// temporary 0600 JSON file, opens it in $EDITOR (or vi), then validates and
// writes the result back. The saved JSON REPLACES the environment wholesale —
// keys removed in the editor are removed from the store. Other environments in
// the store are untouched. Editor non-zero exit or unchanged content → no write.
//
// When envName is empty, the user is prompted via the env picker TUI — the
// same flow as `hulak run`. Edit deliberately does NOT default to "global":
// editing is destructive enough that we want explicit selection. To create or
// edit a brand-new env, pass it explicitly: `hulak secrets edit --env staging`.
//
// The whole read/edit/validate/write cycle is wrapped in WithStoreLock so an
// edit cannot race with a parallel set/delete.
func runEnvEdit(args []string, envName string) error {
	if len(args) > 0 {
		return fmt.Errorf("too many arguments: got %d, expected none", len(args))
	}
	envName, cancelled, err := resolveAndValidateEnv(envName)
	if err != nil {
		return err
	}
	if cancelled {
		return nil
	}

	return vault.WithStoreLock(func() error {
		store, err := vault.ReadStore()
		if err != nil {
			return err
		}

		// Marshal the env (or {} if the env doesn't exist yet — edit creates it).
		env := store.GetEnv(envName)
		if env == nil {
			env = make(vault.Env)
		}
		original, err := json.MarshalIndent(env, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal env: %w", err)
		}

		// Temp file inside .hulak/ keeps plaintext on the same filesystem
		// (same security boundary as store.age). The name encodes the env so
		// users see "edit-prod.json" in their editor's title bar — much nicer
		// than a random suffix. Safe to use a deterministic name because:
		//   - we're inside WithStoreLock (no concurrent edit)
		//   - ValidateEnvName already restricts to [a-zA-Z0-9_-] (no path tricks)
		//   - O_TRUNC overwrites any leftover from a previous crashed run
		markerPath, err := utils.GetProjectMarker()
		if err != nil {
			return err
		}
		tmpPath := filepath.Join(markerPath, "edit-"+envName+".json")
		tmpFile, err := os.OpenFile(
			tmpPath,
			os.O_RDWR|os.O_CREATE|os.O_TRUNC,
			utils.SecretPer,
		)
		if err != nil {
			return fmt.Errorf("failed to create temp file: %w", err)
		}
		// Always remove the plaintext temp — even on editor crash, invalid
		// JSON, or panic up the stack.
		defer os.Remove(tmpPath)

		if _, err := tmpFile.Write(original); err != nil {
			_ = tmpFile.Close()
			return fmt.Errorf("failed to write temp file: %w", err)
		}
		if err := tmpFile.Close(); err != nil {
			return fmt.Errorf("failed to close temp file: %w", err)
		}

		if err := launchEditor(tmpPath); err != nil {
			return err
		}

		edited, err := os.ReadFile(tmpPath)
		if err != nil {
			return fmt.Errorf("failed to read edited file: %w", err)
		}

		if bytes.Equal(original, edited) {
			utils.PrintSuccessStderr(fmt.Sprintf("No changes to %s", envName))
			return nil
		}

		var newEnv vault.Env
		dec := json.NewDecoder(bytes.NewReader(edited))
		dec.UseNumber()
		if err := dec.Decode(&newEnv); err != nil {
			return fmt.Errorf("invalid JSON in edited file (store unchanged): %w", err)
		}

		store.Envs[envName] = newEnv

		if err := vault.WriteStoreToRecipients(store); err != nil {
			return err
		}

		utils.PrintSuccessStderr(fmt.Sprintf("Updated %s", envName))
		return nil
	})
}

// launchEditor runs $EDITOR (or vi if unset) with path appended as its last
// argument. Stdin/Stdout/Stderr are wired to the parent terminal so the user
// interacts directly with the editor.
//
// $EDITOR is whitespace-split into argv (handles "code -w", "nvim --clean")
// but NOT shell-parsed — quotes and shell metachars in $EDITOR are not
// interpreted. Users with exotic editor invocations should write a wrapper
// script and point $EDITOR at it.
func launchEditor(path string) error {
	editor := strings.TrimSpace(os.Getenv("EDITOR"))
	if editor == "" {
		editor = utils.Editor
	}
	parts := strings.Fields(editor)
	parts = append(parts, path)

	//nolint:gosec // G204 $EDITOR is user-controlled by design — that's the contract.
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("editor failed: %w", err)
	}
	return nil
}

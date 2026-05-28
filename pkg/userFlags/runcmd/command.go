// Package runcmd implements the `hulak run` subcommand: takes a file or
// directory of .hk.yaml requests and dispatches to the runner. New()
// builds the command for registration by the top-level dispatch.
package runcmd

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/xaaha/hulak/pkg/runner"
	"github.com/xaaha/hulak/pkg/userFlags/cli"
	"github.com/xaaha/hulak/pkg/userFlags/cliflags"
	"github.com/xaaha/hulak/pkg/utils"
)

// New builds the `hulak run` command.
func New() *cli.Command {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	envFlagVal := cliflags.RegisterEnv(fs, "", "Environment to use")
	var sequential bool
	var debug bool
	var quiet bool
	var timeout time.Duration
	var sshIdentity string
	fs.BoolVar(&sequential, "sequential", false, "Run directory files sequentially")
	fs.BoolVar(&sequential, "seq", false, "Run directory files sequentially")
	fs.BoolVar(&debug, "debug", false, "Enable debug mode")
	fs.BoolVar(&quiet, "quiet", false, "Suppress the end-of-run summary table")
	fs.BoolVar(&quiet, "q", false, "Suppress the end-of-run summary table")
	dryRun := cliflags.RegisterDryRun(fs)
	show := cliflags.RegisterShow(
		fs,
		"Reveal sensitive headers (Authorization, Cookie, etc.) in --dry-run output",
	)
	fs.DurationVar(
		&timeout,
		"timeout",
		0,
		"Per-request timeout, e.g. 5m or 90s (default 60s)",
	)
	fs.StringVar(&sshIdentity, "ssh-identity", "", "Path to SSH private key for vault decryption")

	runCmd := &cli.Command{
		Name:  "run",
		Short: "Run API request file(s) or directory",
		Long: "Execute one or more API request files.\n\n" +
			"Pass a file path to run a single request, or a directory to run all files in it.\n" +
			"Directories run concurrently by default; use --sequential for ordered execution.",
		Examples: []*utils.CommandHelp{
			{Command: "hulak run path/to/file.yaml", Description: "Run a single request file"},
			{
				Command:     "hulak run path/to/file.yaml --env staging",
				Description: "Run with a specific environment",
			},
			{
				Command:     "hulak run path/to/dir/",
				Description: "Run all files in a directory concurrently",
			},
			{
				Command:     "hulak run path/to/dir/ --sequential",
				Description: "Run directory files sequentially",
			},
			{
				Command:     "hulak run path/to/file.yaml --ssh-identity ~/.ssh/work_ed25519",
				Description: "Use a specific SSH key for vault decryption",
			},
			{
				Command:     "hulak run path/to/file.yaml --dry-run",
				Description: "Print the built request and exit (sensitive headers masked)",
			},
			{
				Command:     "hulak run path/to/file.yaml --dry-run --show",
				Description: "Same as --dry-run but reveal sensitive headers",
			},
		},
		Flags: fs,
		Args: []cli.ArgDef{
			{Name: "path", Required: true, Desc: "File or directory to run"},
		},
	}

	runCmd.Run = func(args []string) error {
		if len(args) == 0 {
			runCmd.PrintHelp()
			return nil
		}

		f, err := parseRunArgs(runCmdArgs{
			Env:         *envFlagVal,
			Sequential:  sequential,
			Debug:       debug,
			Quiet:       quiet,
			DryRun:      *dryRun,
			Show:        *show,
			Timeout:     timeout,
			SSHIdentity: sshIdentity,
			Args:        args,
		})
		if err != nil {
			return err
		}

		// Propagate runner errors so the top-level exit code reflects failures.
		// Per-file detail has already been printed; an empty error message is
		// the runner's signal of "exit non-zero, no extra output needed".
		return runner.Execute(f)
	}

	return runCmd
}

// runCmdArgs bundles the values parsed from the `run` subcommand flagset
// plus the positional args. Passing them as one struct keeps parseRunArgs
// from growing a parameter list every time a flag is added.
type runCmdArgs struct {
	Env         string
	Sequential  bool
	Debug       bool
	Quiet       bool
	DryRun      bool
	Show        bool
	Timeout     time.Duration
	SSHIdentity string
	Args        []string
}

// parseRunArgs builds a runner.Flags from the path and parsed flag values.
// The path routes to FilePath (file), Dir (concurrent), or Dirseq (sequential).
func parseRunArgs(a runCmdArgs) (*runner.Flags, error) {
	path := a.Args[0]

	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot access %q: %w", path, err)
	}

	f := &runner.Flags{
		Debug:       a.Debug,
		Quiet:       a.Quiet,
		DryRun:      a.DryRun,
		Show:        a.Show,
		Timeout:     a.Timeout,
		SSHIdentity: a.SSHIdentity,
	}

	if a.Env != "" {
		f.Env = a.Env
		f.EnvSet = true
	}

	if info.IsDir() {
		if a.Sequential {
			f.Dirseq = path
		} else {
			f.Dir = path
		}
	} else {
		f.FilePath = path
	}

	return f, nil
}

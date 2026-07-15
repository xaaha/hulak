// Package mcpcmd implements the `hulak mcp` subcommand: it serves one or more
// named project directories to AI agents over the Model Context Protocol on
// stdio.
package mcpcmd

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/xaaha/hulak/pkg/mcp"
	"github.com/xaaha/hulak/pkg/userFlags/cli"
	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/vault"
)

// projectMap collects repeated `--project name=path` flags into a map.
// name = keyh
// path = value in the config
type projectMap map[string]string

func (p projectMap) String() string {
	if len(p) == 0 {
		return ""
	}
	return fmt.Sprintf("%v", map[string]string(p))
}

func (p projectMap) Set(v string) error {
	name, path, ok := strings.Cut(v, "=")
	name, path = strings.TrimSpace(name), strings.TrimSpace(path)
	if !ok || name == "" || path == "" {
		return fmt.Errorf("expected name=path, got %q", v)
	}
	if _, dup := p[name]; dup {
		return fmt.Errorf("project %q given more than once", name)
	}
	p[name] = path
	return nil
}

// New builds the `hulak mcp` command. version threads the build version into
// the MCP handshake; schemaJSON is the request schema used to validate
// write_request content.
func New(version string, schemaJSON []byte) *cli.Command {
	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
	projects := projectMap{}
	fs.Var(
		projects,
		"project",
		"Named project as name=path (repeatable, e.g. api=~/work/api-tests)",
	)
	defaultProject := fs.String(
		"default-project",
		"",
		"Project assumed when a request name is unambiguous but no project is given",
	)

	cmd := &cli.Command{
		Name:  "mcp",
		Short: "Serve requests to AI agents over MCP",
		Long: "Start a Model Context Protocol server over stdio.\n\n" +
			"Exposes named project directories to any MCP client (Claude Code,\n" +
			"Cursor, Zed, ...). Pass each project with --project name=path; the\n" +
			"agent picks one per call, or the server reports ambiguity.",
		Flags: fs,
		Examples: []*utils.CommandHelp{
			{
				Command:     "hulak mcp --project api=~/work/api-tests",
				Description: "Serve a single named project",
			},
			{
				Command:     "hulak mcp --project api=~/work/api --project mob=~/work/mob --default-project api",
				Description: "Serve multiple projects with a default",
			},
		},
	}

	cmd.Run = func(_ []string) error {
		if len(projects) == 0 {
			return fmt.Errorf("at least one --project name=path is required")
		}
		srv, err := mcp.NewServer(projects, *defaultProject, version)
		if err != nil {
			return err
		}
		srv.SetRequestSchema(schemaJSON)
		if err := requireIdentity(srv.Projects()); err != nil {
			return err
		}
		return srv.Serve(context.Background())
	}

	return cmd
}

// requireIdentity fails fast when any served project uses the encrypted vault
// but no usable identity is configured. stdin is the JSON-RPC channel, so a
// passphrase prompt would hang the session — the server must not start.
func requireIdentity(projects map[string]string) error {
	for _, path := range projects {
		usesVault := utils.FileExists(filepath.Join(path, utils.HiddenProjectName, utils.StoreFile))
		if usesVault && !vault.HasAnyIdentity() {
			return fmt.Errorf(
				"project %q uses the encrypted vault but no identity is configured — "+
					"cannot decrypt secrets non-interactively.\n"+
					"Set HULAK_MASTER_KEY, or configure a passphrase-less identity "+
					"(run 'hulak init' or point HULAK_SSH_IDENTITY at an SSH key), then retry",
				path,
			)
		}
	}
	return nil
}

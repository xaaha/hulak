# `pkg/userFlags/`

Hulak's CLI dispatch. `main.go` calls `userflags.ParseFlagsSubcmds()` and gets
back either a runner config (interactive mode) or an exit — every other CLI
concern (flag parsing, subcommand routing, help rendering, individual command
logic) lives in this package or one of its subpackages.

## Layout

```
pkg/userFlags/
├── userflags.go          Entry: ParseFlagsSubcmds — routes os.Args
├── subcommands.go        Root command tree builder (one place that knows them all)
├── flags.go              Root flag.CommandLine setup (-fp, -f, -env, …)
├── version.go            `hulak version` (trivial leaf — too small for its own folder)
├── cli_contract_test.go  Surface tests: every expected subcommand/alias is registered
├── snapshot_test.go      Help-output snapshot tests (catches accidental wording drift)
│
├── cli/                  Command struct + Execute dispatch + help rendering
│   ├── command.go        type Command, Execute, FindSub, PrintHelp, flag styling
│   └── project.go        RequireVaultProject (used by every leaf that needs a vault)
│
├── cliflags/             Reusable flag registration helpers
│   ├── env.go            --env / --environment
│   ├── output.go         --out / -o  + ResolveOutputPath (dir-vs-file DWIM)
│   ├── name.go           --name      + ResolveRecipientName fallback rule
│   ├── show.go           --show, --dry-run
│   └── yes.go            --yes / -y  (skip destructive confirm)
│
├── runcmd/               `hulak run` — file/dir → runner.Execute
├── initcmd/              `hulak init`, `init classic`, `gendocs`
├── doctor/               `hulak doctor` (+ all the per-backend health checks)
├── gql/                  `hulak gql` — opens the GraphQL TUI explorer
├── example/              `hulak example` + embedded .hk.yaml templates
└── secrets/              `hulak secrets …` subtree (env CRUD, keys, identity,
                          recipients, sync, backup, migrate, picker)
```

## Import topology (one-way only)

```
main.go
   ↓
userflags  ──→  cli, cliflags, secrets, runcmd, initcmd, doctor, gql, example
                  ↓
                cli, cliflags  (leaves never import each other or userflags)
```

Top imports leaves. Leaves import only `cli` and `cliflags`. No cycles, ever.
If you need a helper in two leaves, lift it to `cli/`, `cliflags/`,
`pkg/utils/`, or `pkg/vault/` — whichever owns the concern.

## How a leaf is wired

Every leaf exposes one constructor, `New() *cli.Command`, registered in
`subcommands.go`. Doctor end-to-end:

```go
// pkg/userFlags/doctor/command.go
func New() *cli.Command {
    fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
    fix := fs.Bool("fix", false, "Auto-repair safe issues")

    return &cli.Command{
        Name:  "doctor",
        Short: "Check project health",
        Flags: fs,
        Run: func(_ []string) error {
            os.Exit(runDoctor(doctorOpts{fix: *fix}))
            return nil
        },
    }
}
```

```go
// pkg/userFlags/subcommands.go
root.SubCommands = []*cli.Command{
    runcmd.New(),
    initcmd.New(),
    doctor.New(),       // ← here
    gql.New(),
    example.New(),
    secrets.New(),
    // … plus the trivial leaves: version, migrate, help
}
```

All the actual logic (`runDoctor`, `doctorOpts`, the checks) stays lowercase
inside the leaf package. Only `New` crosses the boundary.

## Adding a new subcommand

1. `mkdir pkg/userFlags/foo`
2. Create `foo/command.go` with `func New() *cli.Command { … }`
3. Put handlers/types alongside it — keep them unexported
4. Reach for `cliflags.RegisterEnv` / `RegisterOutput` / `RegisterYes` instead
   of writing `fs.StringVar` pairs
5. If it touches the vault: `if err := cli.RequireVaultProject(); err != nil { … }`
6. Register in `subcommands.go`: add `foo.New(),` to `root.SubCommands`
7. Add coverage in `cli_contract_test.go` (existence) and write behavior tests
   in your leaf's `*_test.go`

That's it — dispatch, help, alias resolution, `--help` parsing, and
flag-anywhere ordering are all handled by `cli.Execute`.

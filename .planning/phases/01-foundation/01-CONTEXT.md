# Phase 1: Foundation - Context

**Gathered:** 2026-03-11
**Status:** Ready for planning

<domain>
## Phase Boundary

Project scaffold, CLI subcommands (backup/restore/diff), PostgreSQL connection, and command pattern interfaces (SchemaFetcher, Serializer, DDLGenerator, CommandRegistry). No object fetching, no serialization — pure scaffolding and interfaces.

</domain>

<decisions>
## Implementation Decisions

### CLI Framework
- Use **cobra** for subcommand handling
- One file per subcommand: `cmd/backup.go`, `cmd/restore.go`, `cmd/diff.go` — each defines its own `cobra.Command`
- `main.go` (in `cmd/pgbackup/`) adds subcommands to root and calls `Execute()`
- Running `pgbackup` with no subcommand shows help and exits 0 (cobra default behavior)
- Connection flags (`--host`, `--port`, `--user`, `--password`, `--dbname`) defined once on root as `PersistentFlags()` — inherited by all subcommands

### Package Structure
- Layout: `cmd/pgbackup/main.go` + `internal/` for all logic
- `internal/` subdirectories: `backup/`, `restore/`, `diff/`, `registry/`, `db/` (and others as needed)
- Core interfaces (`SchemaFetcher`, `Serializer`, `DDLGenerator`) and `CommandRegistry` live in **`internal/core`** (shared types package — avoids circular imports)
- Go module name: Claude's discretion (pick a sensible path)
- Basic **Makefile** with `make build`, `make test`, `make lint` targets

### Error & Output Style
- Colored output: red `Error:` prefix when stdout is a terminal (TTY auto-detect), plain `Error:` in CI/non-TTY
- All errors go to stderr
- Exit codes: `0` = success, `1` = all errors (no distinct per-error codes)
- `--verbose` / `-v` flag deferred — add in the first phase that needs progress output

### pgx Connection
- Use **`pgx.Conn`** (single connection, not pool) — simpler, correct for CLI lifecycle, aligns with `--snapshot` REPEATABLE READ transaction
- Pass `*pgx.Conn` as a parameter to constructors: `NewFetcher(conn *pgx.Conn)` style — explicit, testable, no global state
- **Fail fast**: connect and verify immediately after flags are parsed, before any subcommand logic runs. Clear error if DB is unreachable.

### Claude's Discretion
- Go module name (pick `github.com/[reasonable]/pgbackup` or similar)
- Exact Makefile targets beyond build/test/lint
- Internal package naming beyond what's specified above
- TTY detection library choice (isatty or similar)

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets
- None — greenfield project. No existing Go files.

### Established Patterns
- None yet — this phase establishes the patterns all future phases follow.

### Integration Points
- `internal/core` interfaces are the contract point: all future phases implement `SchemaFetcher`, `Serializer`, `DDLGenerator` and register via `CommandRegistry.Register()`
- Connection established in root cobra `PersistentPreRunE` — available to all subcommands before they execute

</code_context>

<specifics>
## Specific Ideas

- "I want it to feel like pg_dump — familiar to database people" (from REQUIREMENTS.md context)
- Should work in CI pipelines: clean exit codes, no interactive prompts
- Detailed design spec at `docs/pre/pgbackup_spec.md` — researcher should read section 4 (directory structure) and section 5.2 (YAML format) for interface design context

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 01-foundation*
*Context gathered: 2026-03-11*

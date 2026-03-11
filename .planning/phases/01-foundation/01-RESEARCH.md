# Phase 1: Foundation - Research

**Researched:** 2026-03-11
**Domain:** Go CLI scaffolding, cobra subcommands, pgx single connection, command pattern interfaces
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### CLI Framework
- Use **cobra** for subcommand handling
- One file per subcommand: `cmd/backup.go`, `cmd/restore.go`, `cmd/diff.go` — each defines its own `cobra.Command`
- `main.go` (in `cmd/pgbackup/`) adds subcommands to root and calls `Execute()`
- Running `pgbackup` with no subcommand shows help and exits 0 (cobra default behavior)
- Connection flags (`--host`, `--port`, `--user`, `--password`, `--dbname`) defined once on root as `PersistentFlags()` — inherited by all subcommands

#### Package Structure
- Layout: `cmd/pgbackup/main.go` + `internal/` for all logic
- `internal/` subdirectories: `backup/`, `restore/`, `diff/`, `registry/`, `db/` (and others as needed)
- Core interfaces (`SchemaFetcher`, `Serializer`, `DDLGenerator`) and `CommandRegistry` live in **`internal/core`** (shared types package — avoids circular imports)
- Go module name: Claude's discretion (pick a sensible path)
- Basic **Makefile** with `make build`, `make test`, `make lint` targets

#### Error & Output Style
- Colored output: red `Error:` prefix when stdout is a terminal (TTY auto-detect), plain `Error:` in CI/non-TTY
- All errors go to stderr
- Exit codes: `0` = success, `1` = all errors (no distinct per-error codes)
- `--verbose` / `-v` flag deferred — add in the first phase that needs progress output

#### pgx Connection
- Use **`pgx.Conn`** (single connection, not pool) — simpler, correct for CLI lifecycle, aligns with `--snapshot` REPEATABLE READ transaction
- Pass `*pgx.Conn` as a parameter to constructors: `NewFetcher(conn *pgx.Conn)` style — explicit, testable, no global state
- **Fail fast**: connect and verify immediately after flags are parsed, before any subcommand logic runs. Clear error if DB is unreachable.

### Claude's Discretion
- Go module name (pick `github.com/[reasonable]/pgbackup` or similar)
- Exact Makefile targets beyond build/test/lint
- Internal package naming beyond what's specified above
- TTY detection library choice (isatty or similar)

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| FOUND-01 | Go project structure (cmd/internal layout) and subcommand CLI (backup/restore/diff) setup | cobra v1.9.1 + standard Go layout; spec section 11 defines exact tree |
| FOUND-02 | PostgreSQL connection (--host, --port, --user, --password, --dbname flags) | pgx/v5 v5.8.0 `pgx.Connect` with keyword/value connstring; `PersistentPreRunE` for fail-fast |
| FOUND-03 | Command pattern foundation (ObjectDef, SchemaFetcher, Serializer, DDLGenerator interfaces) | Spec section 10 provides complete Go interface definitions verbatim |
| FOUND-04 | CommandRegistry for object-kind → command resolution | Spec section 10.5 provides complete `CommandRegistry` struct and `Register()` API verbatim |
</phase_requirements>

---

## Summary

Phase 1 is a pure scaffolding phase — no PostgreSQL queries, no file I/O, no business logic beyond wiring. The project design is fully pre-specified in `docs/pre/pgbackup_spec.md` (sections 10–11), which provides near-complete Go code for the command pattern, interfaces, and registry. The researcher's job here is to verify library versions and patterns, not to design anything new.

The locked decisions cover all structural choices: cobra for CLI, pgx/v5 single connection, `internal/core` for shared interfaces, fail-fast connection in `PersistentPreRunE`. The only discretionary choices remaining are the Go module name and Makefile extras.

A critical note on the spec's project layout (section 11): it shows `cmd/backup/main.go`, `cmd/restore/main.go`, `cmd/diff/main.go` as three separate binaries. The CONTEXT.md overrides this — the locked decision is a **single binary** `cmd/pgbackup/main.go` with cobra subcommands. The planner must follow CONTEXT.md, not the spec's section 11 layout for `cmd/`.

**Primary recommendation:** Use the spec's section 10 code verbatim for interfaces and registry; use cobra's `PersistentPreRunE` on the root command for fail-fast DB connection; pick `github.com/pgbkrs/pgbackup` as the module name (matches the working directory name).

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| github.com/spf13/cobra | v1.9.1 | CLI subcommand framework | Explicitly decided; used by kubectl, Hugo, gh CLI; de facto standard |
| github.com/jackc/pgx/v5 | v5.8.0 | PostgreSQL connection and queries | Explicitly decided; supports Go 1.24; single-conn `pgx.Conn` API is correct for CLI |
| go.yaml.in/yaml/v3 | v3.0.4 | YAML serialization (used in later phases) | The spec lists `gopkg.in/yaml.v3` but `go.yaml.in/yaml/v3` is its maintained successor (April 2025 handoff); API-compatible drop-in |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| github.com/mattn/go-isatty | v0.0.20 | TTY detection for colored output | Use in error formatter to decide color vs plain; Claude's discretion choice |
| encoding/csv | stdlib | CSV read/write | Stdlib; no external dep needed |
| context | stdlib | Context propagation to pgx calls | Always pass `context.Background()` or timeout-derived context |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| go.yaml.in/yaml/v3 | gopkg.in/yaml.v3 | gopkg.in path is archived/unmaintained since April 2025; API identical, prefer maintained fork |
| go.yaml.in/yaml/v3 | goccy/go-yaml | goccy offers struct tags and better perf but different API; not worth migration cost for this use case |
| mattn/go-isatty | golang.org/x/term | `term.IsTerminal(int(os.Stdout.Fd()))` also works; isatty is lighter; either acceptable |
| pgx.Conn | pgxpool.Pool | Pool requires concurrency; CLI is single-user sequential; `pgx.Conn` is correct and simpler |

**Installation:**
```bash
go get github.com/spf13/cobra@latest
go get github.com/jackc/pgx/v5@latest
go get go.yaml.in/yaml/v3@latest
go get github.com/mattn/go-isatty@latest
```

---

## Architecture Patterns

### Recommended Project Structure

This is the CONTEXT.md-locked structure (single binary, not the spec's multi-binary section 11):

```
pgbackup/                        # repo root (go module: github.com/pgbkrs/pgbackup)
├── cmd/
│   └── pgbackup/
│       ├── main.go              # calls cmd.Execute()
│       └── cmd/
│           ├── root.go          # rootCmd, PersistentFlags, PersistentPreRunE
│           ├── backup.go        # backupCmd cobra.Command
│           ├── restore.go       # restoreCmd cobra.Command
│           └── diff.go          # diffCmd cobra.Command
├── internal/
│   ├── core/                    # shared interfaces + registry (no other internal pkg imports this)
│   │   ├── interfaces.go        # SchemaFetcher, Serializer, DDLGenerator, ObjectDef
│   │   ├── types.go             # ObjectKind constants, ObjectHeader, TableDef, ViewDef, ...
│   │   └── registry.go          # CommandRegistry
│   ├── db/                      # DB connection helpers
│   │   └── connect.go           # BuildConnString(), ConnectAndPing()
│   ├── backup/                  # backup orchestration (Phase 2+)
│   ├── restore/                 # restore orchestration (Phase 8+)
│   └── diff/                    # diff orchestration (Phase 10+)
├── go.mod
├── go.sum
└── Makefile
```

**Note on `cmd/` layout:** There are two valid cobra conventions:
1. `cmd/pgbackup/main.go` + `cmd/pgbackup/cmd/*.go` — all cobra files co-located
2. `cmd/pgbackup/main.go` + separate `internal/cli/*.go` for cobra commands

Convention 1 is more common for single-binary tools. Convention 2 makes the cobra layer more testable. Either works; convention 1 is simpler for this phase.

### Pattern 1: PersistentPreRunE for Fail-Fast DB Connection

**What:** Root cobra command's `PersistentPreRunE` reads the connection flags and calls `pgx.Connect`. If it fails, prints to stderr and returns error (cobra exits 1).

**When to use:** Exactly once, on the root command. All three subcommands inherit this behavior automatically.

**Key cobra behavior:** A child command that defines its own `PersistentPreRunE` will override the parent's. Since backup/restore/diff subcommands will NOT define their own `PersistentPreRunE`, the root's runs for all three.

**Example:**
```go
// Source: cobra docs https://cobra.dev + pgx docs https://pkg.go.dev/github.com/jackc/pgx/v5
var conn *pgx.Conn

var rootCmd = &cobra.Command{
    Use:   "pgbackup",
    Short: "PostgreSQL schema backup/restore/diff tool",
    PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
        connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
            host, port, user, password, dbname)
        var err error
        conn, err = pgx.Connect(context.Background(), connStr)
        if err != nil {
            return fmt.Errorf("cannot connect to database: %w", err)
        }
        return nil
    },
}
```

**Note on conn scope:** `conn` is declared as a package-level variable in the `cmd` package so subcommand `Run` functions can access it. Alternatively, pass it via `cobra.Command.SetContext`. Package-level is simpler and correct for a single-binary CLI with no concurrency.

### Pattern 2: CommandRegistry with One-Line Registration

**What:** `internal/core.CommandRegistry` maps `ObjectKind` to the triple (SchemaFetcher, Serializer, DDLGenerator). `Register()` is the single entry point. This is fully specified in the spec (section 10.5).

**When to use:** Registry is constructed once at startup; subcommand handlers receive it as a parameter.

**Example (from spec section 10.5, verified against Go interface conventions):**
```go
// Source: docs/pre/pgbackup_spec.md section 10.5
type CommandRegistry struct {
    fetchers    map[ObjectKind]SchemaFetcher
    serializers map[ObjectKind]Serializer
    generators  map[ObjectKind]DDLGenerator
}

func (r *CommandRegistry) Register(kind ObjectKind, f SchemaFetcher, s Serializer, g DDLGenerator) {
    r.fetchers[kind] = f
    r.serializers[kind] = s
    r.generators[kind] = g
}

func (r *CommandRegistry) Fetcher(kind ObjectKind) SchemaFetcher    { return r.fetchers[kind] }
func (r *CommandRegistry) Serializer(kind ObjectKind) Serializer     { return r.serializers[kind] }
func (r *CommandRegistry) Generator(kind ObjectKind) DDLGenerator    { return r.generators[kind] }
```

In Phase 1, `NewCommandRegistry()` registers stub implementations (empty structs). Real implementations are added in Phases 2–4.

### Pattern 3: pgx.Connect with Keyword/Value Connection String

**What:** pgx accepts both URL-style and keyword/value connstrings. The flag-based approach maps naturally to keyword/value.

**Example:**
```go
// Source: https://pkg.go.dev/github.com/jackc/pgx/v5
connStr := fmt.Sprintf(
    "host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
    host, port, user, password, dbname,
)
conn, err := pgx.Connect(ctx, connStr)
```

**Verification ping:** After connect, run a `conn.Ping(ctx)` or trivial query (`SELECT 1`) to ensure the connection is live. `pgx.Connect` alone may not surface auth errors until the first query on some PostgreSQL configurations.

### Pattern 4: Interface Stubs for Phase 1

All object-type-specific structs (`TableSchemaFetcher`, `TableSerializer`, etc.) are empty struct stubs in Phase 1 — they exist to satisfy the interface and prove the build compiles. Real logic is added in later phases.

```go
// Stub — satisfies SchemaFetcher, build passes, no real work
type TableSchemaFetcher struct{}

func (f *TableSchemaFetcher) Fetch(ctx context.Context, conn *pgx.Conn, schema string) ([]ObjectDef, error) {
    return nil, nil
}
```

### Anti-Patterns to Avoid

- **Global `pgx.Conn` initialized in `init()`:** Use `PersistentPreRunE` instead — runs at the right lifecycle point, handles errors cleanly.
- **Storing conn in `context.Context` via arbitrary key:** Prefer package-level var in `cmd` package; context values require type assertions and hide the dependency.
- **Importing `internal/core` from `main.go`:** `main.go` should only import the `cmd` package and call `Execute()`. All wiring happens inside `cmd/`.
- **One giant `cmd/root.go` with all subcommand logic:** Each subcommand gets its own file (`backup.go`, `restore.go`, `diff.go`) — this is locked in CONTEXT.md.
- **Circular imports between `internal/` packages:** `internal/core` is the shared types package. Other internal packages import core; core imports nothing from other internal packages.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Subcommand routing | Custom arg parser | cobra | Flag inheritance, help generation, shell completion — hundreds of edge cases |
| TTY detection | `os.Getenv("TERM")` heuristics | mattn/go-isatty | Handles pipes, CI, Windows, cygwin; single bool call |
| Connection string building | Complex DSN escaping | pgx keyword/value format | pgx handles quoting and special chars in password |
| YAML marshal/unmarshal | Custom string builder | go.yaml.in/yaml/v3 | Nested structs, null handling, anchors — not trivial |

**Key insight:** For this phase the "don't hand-roll" risk is low (it's pure scaffolding), but establishing the right libraries now prevents rewrites in phases 2–10 when real logic appears.

---

## Common Pitfalls

### Pitfall 1: Spec Section 11 Shows Multi-Binary Layout — CONTEXT.md Overrides It

**What goes wrong:** Developer follows spec section 11 (`cmd/backup/main.go`, `cmd/restore/main.go`, `cmd/diff/main.go`) and builds three separate binaries.

**Why it happens:** Spec was written before the single-binary cobra decision was locked.

**How to avoid:** Use `cmd/pgbackup/main.go` as the single entry point. The three subcommands are cobra `Command`s registered to root, not separate `main` packages.

**Warning signs:** `go build ./cmd/backup/...` producing a separate binary named `backup`.

### Pitfall 2: Child PersistentPreRunE Silently Overrides Parent

**What goes wrong:** A subcommand defines its own `PersistentPreRunE` (e.g., to add extra validation), which replaces the root's DB connection setup — subsequent code panics on nil `conn`.

**Why it happens:** cobra's `PersistentPreRunE` chain: by default only the first (closest) `PersistentPreRunE` in the command hierarchy runs.

**How to avoid:** Do not define `PersistentPreRunE` on any subcommand in Phase 1. If a subcommand needs pre-run logic later, call the parent's function explicitly or use `cobra.EnableTraverseRunHooks = true` (global flag).

**Warning signs:** Tests pass when running backup directly but `conn` is nil inside backup's `Run`.

### Pitfall 3: pgx.Connect Succeeds but Connection Is Unusable

**What goes wrong:** `pgx.Connect` returns nil error but the first real query fails with an auth or permission error.

**Why it happens:** On some PostgreSQL configurations, the TCP handshake succeeds but auth errors surface only on the first protocol exchange — which pgx v5 actually does trigger during `Connect` (it performs the full auth handshake). However, network-level timeouts may cause the connect call to hang rather than error.

**How to avoid:** Set a context deadline on the connect call:
```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
conn, err = pgx.Connect(ctx, connStr)
```

**Warning signs:** `pgbackup backup ...` hangs indefinitely with no output when the DB host is unreachable.

### Pitfall 4: Circular Import Between internal/ Packages

**What goes wrong:** `internal/backup` imports `internal/registry`; `internal/registry` imports `internal/backup` for default registration — Go compiler rejects with circular import error.

**Why it happens:** Registry and concrete implementations naturally reference each other if core types are not separated.

**How to avoid:** `internal/core` owns ALL shared types (ObjectDef, interfaces, CommandRegistry). Concrete implementations (TableSchemaFetcher, etc.) live in separate packages under `internal/` and import `internal/core`. The registry itself lives in `internal/core`. Default registration (the `NewCommandRegistry()` factory that calls `Register(...)`) lives in a thin wiring layer (e.g., `internal/registry/init.go` or directly in `cmd/root.go`).

**Warning signs:** `import cycle not allowed` compiler error.

### Pitfall 5: gopkg.in/yaml.v3 is Archived

**What goes wrong:** Developer uses `gopkg.in/yaml.v3` per the spec's section 12 library list, which is archived as of April 2025.

**Why it happens:** Spec was written before the maintenance handoff.

**How to avoid:** Use `go.yaml.in/yaml/v3` instead. The import path changes but the API is 100% compatible — `yaml.Marshal`, `yaml.Unmarshal`, struct tags are identical.

**Warning signs:** `go get gopkg.in/yaml.v3` succeeds but shows "archived" warning; future security patches won't be backported.

---

## Code Examples

Verified patterns from official sources and spec:

### Root Command Wiring (cmd/pgbackup/cmd/root.go skeleton)
```go
// Source: cobra docs https://cobra.dev
package cmd

import (
    "context"
    "fmt"
    "os"
    "time"

    "github.com/spf13/cobra"
    "github.com/jackc/pgx/v5"
    "github.com/mattn/go-isatty"
)

var (
    host     string
    port     int
    user     string
    password string
    dbname   string
    conn     *pgx.Conn
)

var rootCmd = &cobra.Command{
    Use:               "pgbackup",
    Short:             "PostgreSQL schema backup, restore, and diff tool",
    SilenceUsage:      true,
    SilenceErrors:     true,
    PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
        connStr := fmt.Sprintf(
            "host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
            host, port, user, password, dbname,
        )
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()
        var err error
        conn, err = pgx.Connect(ctx, connStr)
        if err != nil {
            printError(fmt.Errorf("cannot connect to database: %w", err))
            return err
        }
        return nil
    },
}

func Execute() {
    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}

func init() {
    rootCmd.PersistentFlags().StringVar(&host,     "host",     "localhost", "PostgreSQL host")
    rootCmd.PersistentFlags().IntVar(&port,         "port",     5432,       "PostgreSQL port")
    rootCmd.PersistentFlags().StringVar(&user,     "user",     "",         "PostgreSQL user")
    rootCmd.PersistentFlags().StringVar(&password, "password", "",         "PostgreSQL password")
    rootCmd.PersistentFlags().StringVar(&dbname,   "dbname",   "",         "Target database name")

    rootCmd.AddCommand(backupCmd)
    rootCmd.AddCommand(restoreCmd)
    rootCmd.AddCommand(diffCmd)
}

func printError(err error) {
    if isatty.IsTerminal(os.Stderr.Fd()) {
        fmt.Fprintf(os.Stderr, "\033[31mError:\033[0m %v\n", err)
    } else {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
    }
}
```

### Core Interfaces (internal/core/interfaces.go)
```go
// Source: docs/pre/pgbackup_spec.md section 10.3
package core

import (
    "context"
    "github.com/jackc/pgx/v5"
)

// SchemaFetcher fetches object definitions from pg_catalog.
type SchemaFetcher interface {
    Fetch(ctx context.Context, conn *pgx.Conn, schema string) ([]ObjectDef, error)
}

// Serializer converts ObjectDef to/from YAML bytes.
type Serializer interface {
    Serialize(def ObjectDef) ([]byte, error)
    Deserialize(data []byte) (ObjectDef, error)
}

// DDLGenerator produces executable DDL strings from an ObjectDef.
type DDLGenerator interface {
    GenerateDDL(def ObjectDef) ([]string, error)
    GenerateDrop(def ObjectDef) ([]string, error)
}
```

### ObjectDef Type Hierarchy (internal/core/types.go)
```go
// Source: docs/pre/pgbackup_spec.md section 10.2
package core

type ObjectKind string

const (
    KindTable            ObjectKind = "table"
    KindView             ObjectKind = "view"
    KindMaterializedView ObjectKind = "materialized_view"
    KindFunction         ObjectKind = "function"
    KindSequence         ObjectKind = "sequence"
    KindTrigger          ObjectKind = "trigger"
    KindType             ObjectKind = "type"
    KindDomain           ObjectKind = "domain"
    KindEnum             ObjectKind = "enum"
    KindPolicy           ObjectKind = "policy"
)

type ObjectHeader struct {
    Kind   ObjectKind
    Schema string
    Name   string
}

type ObjectDef interface {
    Header() ObjectHeader
}

type TableDef struct {
    ObjectHeader
    // Columns, Indexes, Constraints, etc. — populated in Phase 2+
}

func (d TableDef) Header() ObjectHeader { return d.ObjectHeader }

type ViewDef struct {
    ObjectHeader
    Definition string
}

func (d ViewDef) Header() ObjectHeader { return d.ObjectHeader }

// Additional concrete types: MaterializedViewDef, FunctionDef, SequenceDef,
// TriggerDef, TypeDef, DomainDef, EnumDef, PolicyDef — all follow same pattern.
```

### CommandRegistry (internal/core/registry.go)
```go
// Source: docs/pre/pgbackup_spec.md section 10.5
package core

type CommandRegistry struct {
    fetchers    map[ObjectKind]SchemaFetcher
    serializers map[ObjectKind]Serializer
    generators  map[ObjectKind]DDLGenerator
}

func NewCommandRegistry() *CommandRegistry {
    return &CommandRegistry{
        fetchers:    make(map[ObjectKind]SchemaFetcher),
        serializers: make(map[ObjectKind]Serializer),
        generators:  make(map[ObjectKind]DDLGenerator),
    }
}

func (r *CommandRegistry) Register(kind ObjectKind, f SchemaFetcher, s Serializer, g DDLGenerator) {
    r.fetchers[kind] = f
    r.serializers[kind] = s
    r.generators[kind] = g
}

func (r *CommandRegistry) Fetcher(kind ObjectKind) SchemaFetcher    { return r.fetchers[kind] }
func (r *CommandRegistry) Serializer(kind ObjectKind) Serializer     { return r.serializers[kind] }
func (r *CommandRegistry) Generator(kind ObjectKind) DDLGenerator    { return r.generators[kind] }
```

### Minimal Makefile
```makefile
# Source: CONTEXT.md locked decision — build/test/lint targets required
BINARY := pgbackup
CMD    := ./cmd/pgbackup

.PHONY: build test lint clean

build:
	go build -o bin/$(BINARY) $(CMD)

test:
	go test ./...

lint:
	go vet ./...
	@which golangci-lint > /dev/null && golangci-lint run || echo "golangci-lint not installed, skipping"

clean:
	rm -rf bin/
```

### Cobra Subcommand Stub (cmd/pgbackup/cmd/backup.go)
```go
// Source: cobra docs https://cobra.dev
package cmd

import "github.com/spf13/cobra"

var backupCmd = &cobra.Command{
    Use:   "backup",
    Short: "Back up PostgreSQL schema and data to YAML+CSV files",
    RunE: func(cmd *cobra.Command, args []string) error {
        // conn is available here (set by rootCmd.PersistentPreRunE)
        // Phase 2+ will add real backup logic
        return nil
    },
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `gopkg.in/yaml.v3` | `go.yaml.in/yaml/v3` | April 2025 | Archived; use new import path, same API |
| `pgxpool` for CLIs | `pgx.Conn` for CLIs | Ongoing best practice | CLIs are sequential single-user; pool adds needless complexity |
| Three separate `main` packages for subcommands | Single cobra binary | Spec → CONTEXT.md decision | Simpler distribution, shared connection lifecycle |

**Deprecated/outdated:**
- `gopkg.in/yaml.v3`: archived April 2025; use `go.yaml.in/yaml/v3` instead (API-identical)
- Spec section 11 multi-binary layout: overridden by CONTEXT.md single-binary decision

---

## Open Questions

1. **Go module name**
   - What we know: Must be a valid module path; working directory is `pgbkrs`
   - What's unclear: Whether a GitHub repo under that name exists or will exist
   - Recommendation: Use `github.com/pgbkrs/pgbackup` — matches the workspace dir, clean, conventional. Can be changed in one `go.mod` line if needed.

2. **Password flag vs. PGPASSWORD env var**
   - What we know: CONTEXT.md locks `--password` flag; env var support is v2 (ENH-02)
   - What's unclear: CI pipelines often pass credentials via env; `--password` on CLI risks shell history exposure
   - Recommendation: Accept it for Phase 1 per the lock. Document the shell history risk in help text. Do not implement env var fallback until ENH-02.

3. **`SilenceUsage` and `SilenceErrors` on root command**
   - What we know: Setting these prevents cobra from printing usage on every error (noisy for DB tools)
   - What's unclear: Project preference for error-triggered usage display
   - Recommendation: Set both to `true` on rootCmd. DB connection errors should not dump the full usage block. Manual `--help` is always available.

---

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go standard `testing` package (no external framework required for Phase 1) |
| Config file | none — `go test ./...` uses standard Go test discovery |
| Quick run command | `go test ./internal/core/... -v` |
| Full suite command | `go test ./...` |

**Note:** `testify` (`github.com/stretchr/testify`) is recommended as a supporting assertion library for cleaner test output, but is not required. Phase 1 tests are primarily compile-time proof (interfaces satisfied) and unit tests on `CommandRegistry`.

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| FOUND-01 | Binary builds and all three subcommands are recognized | smoke | `go build ./cmd/pgbackup && ./bin/pgbackup backup --help` | ❌ Wave 0 |
| FOUND-01 | `pgbackup` with no subcommand exits 0 and shows help | unit | `go test ./cmd/pgbackup/cmd/... -run TestRootNoArgs` | ❌ Wave 0 |
| FOUND-02 | `pgbackup backup --host ... --dbname ...` opens DB without error | integration | `go test ./internal/db/... -run TestConnect` (needs live DB) | ❌ Wave 0 |
| FOUND-03 | SchemaFetcher, Serializer, DDLGenerator interfaces exist and build compiles | compile-time | `go build ./internal/core/...` | ❌ Wave 0 |
| FOUND-03 | Stub implementations satisfy the interfaces | unit | `go test ./internal/core/... -run TestInterfaceCompliance` | ❌ Wave 0 |
| FOUND-04 | `CommandRegistry.Register()` stores and `Fetcher()`/`Serializer()`/`Generator()` resolve correctly | unit | `go test ./internal/core/... -run TestCommandRegistry` | ❌ Wave 0 |
| FOUND-04 | A new kind registered with one `Register()` call is immediately resolvable | unit | `go test ./internal/core/... -run TestRegistrySingleLine` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go build ./... && go test ./internal/core/...`
- **Per wave merge:** `go test ./...`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/core/registry_test.go` — covers FOUND-04 (Register, Fetcher, Serializer, Generator)
- [ ] `internal/core/interfaces_test.go` — covers FOUND-03 (compile-time interface compliance via stub implementations)
- [ ] `cmd/pgbackup/cmd/root_test.go` — covers FOUND-01 (no-subcommand exits 0, subcommands recognized)
- [ ] `internal/db/connect_test.go` — covers FOUND-02 (integration test, requires `TEST_DATABASE_URL` env var; skip if not set)
- [ ] No test framework install needed — stdlib `testing` package is sufficient for Phase 1

---

## Sources

### Primary (HIGH confidence)
- `docs/pre/pgbackup_spec.md` — complete interface definitions, registry API, ObjectDef type hierarchy, directory structure (sections 4, 5, 10, 11)
- `.planning/phases/01-foundation/01-CONTEXT.md` — all locked architectural decisions for this phase
- [pkg.go.dev/github.com/jackc/pgx/v5](https://pkg.go.dev/github.com/jackc/pgx/v5) — pgx.Connect signature, ConnConfig, single-connection semantics
- [cobra.dev](https://cobra.dev) — PersistentPreRunE inheritance behavior, PersistentFlags, hook execution order

### Secondary (MEDIUM confidence)
- [GitHub jackc/pgx at v5.8.0](https://github.com/jackc/pgx/tree/v5.8.0) — confirms v5.8.0 as latest, Go 1.24 support
- [GitHub spf13/cobra v1.8.1 go.mod](https://github.com/spf13/cobra/blob/v1.8.1/go.mod) — confirms v1.9.1 as latest published version
- [Codeberg forgejo PR #8956](https://codeberg.org/forgejo/forgejo/pulls/8956) — confirms gopkg.in/yaml.v3 → go.yaml.in/yaml/v3 migration rationale
- [go.dev/doc/modules/layout](https://go.dev/doc/modules/layout) — official Go module layout conventions for cmd/internal structure
- [pkg.go.dev/github.com/mattn/go-isatty](https://pkg.go.dev/github.com/mattn/go-isatty) — IsTerminal API, v0.0.20

### Tertiary (LOW confidence)
- WebSearch: cobra v1.9.1 as "latest published December 2025" — not independently verified against pkg.go.dev release page; treat as approximate, use `go get github.com/spf13/cobra@latest` at task time

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — pgx v5 and cobra are locked decisions, versions cross-verified
- Architecture: HIGH — spec section 10 provides verbatim Go code; CONTEXT.md locks all structural choices
- Pitfalls: MEDIUM — cobra hook inheritance and import cycle pitfalls are well-documented; yaml archival is verified; PitFall 3 (pgx hang) is from experience, not a documented pgx behavior
- yaml.v3 successor: MEDIUM — migration confirmed by multiple sources (forgejo, cobra's own go.mod now uses go.yaml.in); API compatibility stated as guaranteed but not independently tested here

**Research date:** 2026-03-11
**Valid until:** 2026-04-11 (stable ecosystem; pgx and cobra are mature, slow-moving)

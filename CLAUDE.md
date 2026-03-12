# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Repository Is

This is the **GSD (Get Shit Done)** framework — a Claude Code workflow orchestration system. All framework code lives under `.claude/`. The framework itself has no traditional build step, test runner, or package dependencies beyond Node.js built-ins.

## CLI Entry Point

The main CLI is `.claude/get-shit-done/bin/gsd-tools.cjs` — a CommonJS Node.js script with 40+ atomic subcommands. It is invoked by Claude agents via:

```bash
node .claude/get-shit-done/bin/gsd-tools.cjs <command> [args]
```

No npm install required. All dependencies are Node.js built-ins (fs, path, child_process, crypto).

## Architecture

```
.claude/
├── commands/gsd/          # 33 slash command definitions (user-facing, markdown)
├── workflows/             # 32 orchestration workflows (what commands actually do)
├── agents/                # 12 specialized Claude agent prompts
├── hooks/                 # Session lifecycle hooks (update check, context monitor, statusline)
├── get-shit-done/
│   ├── bin/gsd-tools.cjs  # Main CLI
│   ├── bin/lib/           # 11 library modules (state, phase, roadmap, verify, etc.)
│   ├── templates/         # Markdown document templates (PROJECT.md, PLAN.md, etc.)
│   ├── references/        # Technical spec documents for agents to read
│   └── VERSION            # Semver version string
├── settings.json          # Hook registration + statusline config
├── gsd-file-manifest.json # SHA256 integrity hashes for all framework files
└── package.json           # { "type": "commonjs" }
```

### Key Layers

1. **Commands** (`commands/gsd/*.md`) — Markdown specs defining what each `/gsd:*` slash command does; these are read by Claude to know which workflow to invoke.

2. **Workflows** (`workflows/*.md`) — Orchestration logic: step-by-step instructions telling Claude how to run an operation (which agents to spawn, what tools to call, when to confirm with user).

3. **Agents** (`agents/*.md`) — Specialized Claude subagent prompts spawned via the `Agent` tool for focused tasks (planning, execution, verification, research, etc.).

4. **Library modules** (`bin/lib/`) — Node.js modules handling state persistence:
   - `state.cjs` — STATE.md parsing and frontmatter field updates
   - `phase.cjs` — Phase lifecycle (create, renumber, complete)
   - `roadmap.cjs` — ROADMAP.md parsing
   - `verify.cjs` — SUMMARY.md validation
   - `core.cjs` — Shared utilities (file I/O, git, config)
   - `commands.cjs` — Standalone commands (slug generation, todos, timestamps)

### Planning Directory (per-project)

Each project using GSD maintains a `.planning/` directory with:
- `STATE.md` — YAML frontmatter tracking current phase, plan, model profile
- `config.json` — Workflow toggles (parallelization, gates, safety confirmations)
- `PROJECT.md`, `ROADMAP.md`, `REQUIREMENTS.md` — Core project documents
- Per-phase directories: `phases/XX-name/PLAN.md`, `SUMMARY.md`, `CONTEXT.md`

## Session Hooks

- **SessionStart** → `gsd-check-update.js` — checks for framework updates
- **PostToolUse** → `gsd-context-monitor.js` — warns at 65%/80%/>80% context usage
- **statusLine** → `gsd-statusline.js` — shows `model | task | dir | context%`

## File Integrity

`gsd-file-manifest.json` contains SHA256 hashes for all framework files. When modifying framework files, update the manifest entry for any changed file:

```bash
node -e "const c=require('crypto'),f=require('fs');console.log(c.createHash('sha256').update(f.readFileSync('PATH')).digest('hex'))"
```

## Making Changes

- **Adding a new command**: create `commands/gsd/<name>.md` and `workflows/<name>.md`; register the command in `commands/gsd/help.md`
- **Modifying agent behavior**: edit the relevant `agents/gsd-<name>.md`
- **Adding CLI operations**: add a new `case` block in `gsd-tools.cjs` and corresponding library function
- **Templates**: all project-facing document templates are in `get-shit-done/templates/`; agent reference docs are in `get-shit-done/references/`

## Model Profile System

Three tiers defined in `references/model-profiles.md` and resolved at runtime by `gsd-tools.cjs`:
- `quality` — Opus for all decision-making agents
- `balanced` (default) — Opus for planning, Sonnet for execution/research
- `budget` — Sonnet/Haiku minimizing Opus usage

Per-agent overrides are supported in `.planning/config.json`.

## Testing

### Unit tests (no DB required)

```bash
go test -short ./...
```

### Integration tests (requires PostgreSQL)

Start the test DB (Docker):

```bash
source scripts/testdb-start.sh   # sets TEST_DATABASE_URL in current shell
# or
make testdb-start                 # just starts the container
```

Container: `pgbkrs-testdb`, port `15432`, already running is fine — the script is idempotent.

Run all tests:

```bash
make test-integration
# equivalent: go test -p 1 ./...
```

**`-p 1` is required.** Without it, packages run in parallel and `TestBackupNonSnapshotMode` fails intermittently — fetch tests in other packages create/drop objects mid-run, leaving stale OIDs that the non-snapshot backup traversal tries to read.

Stop the DB when done:

```bash
make testdb-stop
```

## Commit Convention

Planning artifact commits use: `docs(XX-YY): description` or `feat(XX-YY): description` where XX=phase, YY=plan. Regular code commits use conventional commits: `feat|fix|test|refactor|perf|chore(scope): message`.

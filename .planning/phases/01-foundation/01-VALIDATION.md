---
phase: 1
slug: foundation
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-11
---

# Phase 1 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go standard `testing` package + `github.com/stretchr/testify` (optional assertions) |
| **Config file** | none — `go test ./...` uses standard Go test discovery |
| **Quick run command** | `go build ./... && go test ./internal/core/...` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~5 seconds (unit only); integration test skipped without `TEST_DATABASE_URL` |

---

## Sampling Rate

- **After every task commit:** Run `go build ./... && go test ./internal/core/...`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** ~5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 1-xx-01 | TBD | 0 | FOUND-04 | unit | `go test ./internal/core/... -run TestCommandRegistry` | ❌ W0 | ⬜ pending |
| 1-xx-02 | TBD | 0 | FOUND-04 | unit | `go test ./internal/core/... -run TestRegistrySingleLine` | ❌ W0 | ⬜ pending |
| 1-xx-03 | TBD | 0 | FOUND-03 | compile | `go build ./internal/core/...` | ❌ W0 | ⬜ pending |
| 1-xx-04 | TBD | 0 | FOUND-03 | unit | `go test ./internal/core/... -run TestInterfaceCompliance` | ❌ W0 | ⬜ pending |
| 1-xx-05 | TBD | 0 | FOUND-01 | unit | `go test ./cmd/pgbackup/cmd/... -run TestRootNoArgs` | ❌ W0 | ⬜ pending |
| 1-xx-06 | TBD | 0 | FOUND-01 | smoke | `go build ./cmd/pgbackup && ./pgbackup backup --help` | ❌ W0 | ⬜ pending |
| 1-xx-07 | TBD | 1 | FOUND-02 | integration | `go test ./internal/db/... -run TestConnect` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/core/registry_test.go` — unit tests for FOUND-04 (Register, Fetcher, Serializer, Generator)
- [ ] `internal/core/interfaces_test.go` — compile-time interface compliance via stub implementations (FOUND-03)
- [ ] `cmd/pgbackup/cmd/root_test.go` — subcommand recognition + no-arg help (FOUND-01)
- [ ] `internal/db/connect_test.go` — integration test for FOUND-02; skips if `TEST_DATABASE_URL` not set

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| `pgbackup backup --host ... --port ... --user ... --dbname ...` connects to live PostgreSQL | FOUND-02 | Requires live DB; CI skips without `TEST_DATABASE_URL` | Set `TEST_DATABASE_URL=postgres://u:p@localhost:5432/d` and run `go test ./internal/db/... -run TestConnect -v` |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 10s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending

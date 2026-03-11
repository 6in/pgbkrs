---
phase: 9
slug: restore-partial-safety
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-12
---

# Phase 9 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing (stdlib) + `go test` |
| **Config file** | none (uses `go test ./...`) |
| **Quick run command** | `go test ./internal/restore/... -short` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/restore/... -short`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 09-01-01 | 01 | 0 | REST-11,REST-12 | unit | `go test ./internal/restore/... -run TestBuildTransitiveClosure\|TestMissingDependency -short` | ❌ W0 | ⬜ pending |
| 09-01-02 | 01 | 0 | REST-03 | unit | `go test ./internal/restore/... -run TestDetectLeaks -short` | ❌ W0 | ⬜ pending |
| 09-02-01 | 02 | 1 | REST-03,REST-10,REST-11,REST-12,REST-13 | unit+integration | `go test ./internal/restore/... -short` | ❌ W0 | ⬜ pending |
| 09-03-01 | 03 | 2 | REST-10,REST-11,REST-13 | build+CLI | `go build ./... && ./bin/pgbackup restore --help` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/restore/filter.go` + `filter_test.go` — BFS closure, filteredRestoreOrder, verifyDepsPresent (unit tests, no DB)
- [ ] `internal/restore/leak.go` + `leak_test.go` — snapshotLiveObjects, detectLeaks
- [ ] `internal/restore/logger.go` + test stubs — Logger struct, NewLogger, LogDrop, LogRestore, LogLeakWarning, WriteSummary, Close
- [ ] `internal/restore/orchestrator.go` updated to accept `Options` struct (breaking change — all callers updated in same wave)
- [ ] `internal/restore/orchestrator_test.go` updated for new signature + stub tests for REST-03, REST-10, REST-11, REST-12, REST-13

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| `--schema` restores only target schema, ignores others | REST-10 | Requires real multi-schema DB | Create 2 schemas, backup, restore --schema s1, verify s2 untouched |
| `--object` with FK: FK constraint included if both tables in closure | REST-11 | Complex graph topology | Requires real FK structure in test DB |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending

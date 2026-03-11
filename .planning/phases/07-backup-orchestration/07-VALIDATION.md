---
phase: 7
slug: backup-orchestration
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-11
---

# Phase 7 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — standard Go testing |
| **Quick run command** | `go test ./internal/backup/... -count=1` |
| **Full suite command** | `go test ./... -count=1` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/backup/... -count=1`
- **After every plan wave:** Run `go test ./... -count=1`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 7-01-01 | 01 | 0 | BKUP-01 | unit stub | `go test ./internal/backup/... -run TestOrchestrator` | ❌ W0 | ⬜ pending |
| 7-01-02 | 01 | 1 | BKUP-01 | unit | `go test ./internal/backup/... -run TestOrchestratorDirectoryLayout` | ✅ | ⬜ pending |
| 7-01-03 | 01 | 1 | BKUP-02 | unit | `go test ./internal/backup/... -run TestSkipUnsupportedColumns` | ✅ | ⬜ pending |
| 7-01-04 | 01 | 1 | BKUP-03 | unit | `go test ./internal/backup/... -run TestSnapshotMode` | ✅ | ⬜ pending |
| 7-01-05 | 01 | 1 | BKUP-04 | unit | `go test ./internal/backup/... -run TestNonSnapshotMode` | ✅ | ⬜ pending |
| 7-02-01 | 02 | 2 | BKUP-01 | integration | `go test ./... -run TestBackupIntegration` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/backup/orchestrator_test.go` — stubs for BKUP-01, BKUP-02, BKUP-03, BKUP-04
- [ ] Test fixtures and helpers for mock database connections

*Wave 0 must create test file stubs before implementation begins.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Correct directory structure on real PostgreSQL | BKUP-01 | Requires live DB | Run `pgbackup backup` against test DB, inspect output dir structure |
| Warnings log contains skipped tables | BKUP-02 | Requires live DB with bytea/xml columns | Verify warnings.json or warnings.log in backup dir |
| Snapshot manifest field | BKUP-03 | Requires live DB | Check manifest.json `snapshot_mode: true` |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending

---
phase: 8
slug: restore-core-pipeline
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-12
---

# Phase 8 — Validation Strategy

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
| 08-01-01 | 01 | 0 | REST-01..09 | unit/integration stubs | `go test ./internal/restore/... -short` | ❌ W0 | ⬜ pending |
| 08-01-02 | 01 | 0 | REST-02,REST-04 | integration | `go test ./internal/restore/... -run TestRestoreDropOrder` | ❌ W0 | ⬜ pending |
| 08-02-01 | 02 | 1 | REST-01,REST-02,REST-04 | integration | `go test ./internal/restore/... -run TestPreRestoreBackup\|TestRestoreDropOrder\|TestRestoreCreateOrder` | ❌ W0 | ⬜ pending |
| 08-02-02 | 02 | 1 | REST-05,REST-06,REST-07,REST-08,REST-09 | integration | `go test ./internal/restore/... -run TestRestoreCopyFrom\|TestRestoreSetval\|TestRestoreIndexes\|TestRestoreFKBatch\|TestRestoreViewsFunctions` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/restore/orchestrator_test.go` — integration test stubs for all 8 REST requirements
- [ ] `internal/restore/loader_test.go` — unit test stubs for manifest reading, defYAMLPath, deserialize round-trip
- [ ] `internal/resolve/manifest.go` — add `ReadManifest` function (currently only `WriteManifest` exists)
- [ ] `internal/serialize/table/serializer.go` — add `definition` field to `yamlIndex` (index DDL gap fix)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Pre-restore backup written to correct path | REST-01 | Filesystem inspection | Run restore, check output dir contains pre-restore backup |
| Partition child tables receive data via COPY | REST-05 | Requires real partitioned schema | Set up partitioned table, backup, restore, verify row counts per child |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending

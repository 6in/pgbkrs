---
phase: 10
slug: schema-diff
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-12
---

# Phase 10 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go standard `testing` package (go 1.24) |
| **Config file** | none — uses `go test` directly |
| **Quick run command** | `go test -short ./internal/diff/...` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test -short ./internal/diff/...`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** ~5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 10-01-01 | 01 | 0 | DIFF-01 | unit | `go test -short ./internal/diff/... -run TestLoadBackup` | ❌ W0 | ⬜ pending |
| 10-01-02 | 01 | 0 | DIFF-02 | unit | `go test -short ./internal/diff/... -run TestDiffAddedRemoved` | ❌ W0 | ⬜ pending |
| 10-01-03 | 01 | 0 | DIFF-03 | unit | `go test -short ./internal/diff/... -run TestDiffTable` | ❌ W0 | ⬜ pending |
| 10-01-04 | 01 | 0 | DIFF-04 | unit | `go test -short ./internal/diff/... -run TestDiffNonTable` | ❌ W0 | ⬜ pending |
| 10-01-05 | 01 | 0 | DIFF-05 | unit | `go test -short ./internal/diff/... -run TestFormatReport` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/diff/diff_test.go` — stubs for DIFF-01 through DIFF-05
- [ ] `internal/diff/diff.go` — package stub (empty `Run` function returning nil)
- [ ] `internal/diff/loader.go` — stub `loadBackup` returning empty map
- [ ] `internal/diff/compare.go` — stub comparison functions
- [ ] `internal/diff/report.go` — stub formatter

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Output format matches spec 9.3 visually | DIFF-05 | Formatting requires human review of Japanese labels and alignment | Run `pgbackup diff <backup-a> <backup-b>` and compare output to spec section 9.3 |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 5s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending

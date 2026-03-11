---
phase: 6
slug: dependency-resolution-manifest
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-11
---

# Phase 6 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing (stdlib) 1.24 |
| **Config file** | None needed (go test convention) |
| **Quick run command** | `go test ./internal/resolve/... -v -count=1` |
| **Full suite command** | `go test ./... -count=1` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/resolve/... -v -count=1`
- **After every plan wave:** Run `go test ./... -count=1`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 10 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 6-01-01 | 01 | 1 | DEPS-01 | unit | `go test ./internal/resolve/ -run TestDAG -v -count=1` | ❌ W0 | ⬜ pending |
| 6-01-02 | 01 | 1 | DEPS-02 | unit | `go test ./internal/resolve/ -run TestTopologicalSort -v -count=1` | ❌ W0 | ⬜ pending |
| 6-01-03 | 01 | 1 | DEPS-02 | unit | `go test ./internal/resolve/ -run TestCycleDetection -v -count=1` | ❌ W0 | ⬜ pending |
| 6-01-04 | 01 | 1 | DEPS-03 | unit | `go test ./internal/resolve/ -run TestFKIsolation -v -count=1` | ❌ W0 | ⬜ pending |
| 6-02-01 | 02 | 1 | DEPS-04 | unit | `go test ./internal/resolve/ -run TestManifest -v -count=1` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/resolve/dag_test.go` — stubs for DEPS-01, DEPS-03
- [ ] `internal/resolve/kahn_test.go` — stubs for DEPS-02
- [ ] `internal/resolve/manifest_test.go` — stubs for DEPS-04

*Existing Go test infrastructure covers all framework needs.*

---

## Manual-Only Verifications

*All phase behaviors have automated verification.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 10s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending

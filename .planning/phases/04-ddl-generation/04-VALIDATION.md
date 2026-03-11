---
phase: 4
slug: ddl-generation
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-11
---

# Phase 4 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (stdlib) |
| **Config file** | none (go test convention) |
| **Quick run command** | `go test ./internal/ddl/... -count=1` |
| **Full suite command** | `go test ./... -count=1` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/ddl/... -count=1`
- **After every plan wave:** Run `go test ./... -count=1`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 10 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 04-01-01 | 01 | 1 | DDLG-01, DDLG-02, DDLG-03, DDLG-04 | unit | `go test ./internal/ddl/... -count=1` | ❌ W0 | ⬜ pending |
| 04-02-01 | 02 | 2 | DDLG-01 | unit | `go test ./internal/ddl/table/ -run TestGenerateDDL -count=1` | ❌ W0 | ⬜ pending |
| 04-02-02 | 02 | 2 | DDLG-01 | unit | `go test ./internal/ddl/table/ -run TestGenerateDDL_NoFK -count=1` | ❌ W0 | ⬜ pending |
| 04-03-01 | 03 | 2 | DDLG-02 | unit+integration | `go test ./internal/ddl/foreignkey/ -count=1` | ❌ W0 | ⬜ pending |
| 04-04-01 | 04 | 2 | DDLG-03 | unit | `go test ./internal/ddl/view/ -run TestGenerateDDL -count=1` | ❌ W0 | ⬜ pending |
| 04-04-02 | 04 | 2 | DDLG-03 | unit | `go test ./internal/ddl/matview/ -run TestGenerateDDL -count=1` | ❌ W0 | ⬜ pending |
| 04-04-03 | 04 | 2 | DDLG-03 | unit | `go test ./internal/ddl/function/ -run TestGenerateDDL -count=1` | ❌ W0 | ⬜ pending |
| 04-04-04 | 04 | 2 | DDLG-03 | unit | `go test ./internal/ddl/trigger/ -run TestGenerateDDL -count=1` | ❌ W0 | ⬜ pending |
| 04-05-01 | 05 | 2 | DDLG-04 | unit | `go test ./internal/ddl/... -run TestGenerateDrop -count=1` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/ddl/table/generator_test.go` — stubs for DDLG-01
- [ ] `internal/ddl/foreignkey/generator_test.go` — stubs for DDLG-02
- [ ] `internal/ddl/view/generator_test.go` — stubs for DDLG-03 (view)
- [ ] `internal/ddl/matview/generator_test.go` — stubs for DDLG-03 (matview)
- [ ] `internal/ddl/function/generator_test.go` — stubs for DDLG-03 (function)
- [ ] `internal/ddl/trigger/generator_test.go` — stubs for DDLG-03 (trigger)
- [ ] `internal/ddl/sequence/generator_test.go` — stubs for DDLG-04 (sequence)
- [ ] `internal/ddl/compositetype/generator_test.go` — stubs for DDLG-04 (type)
- [ ] `internal/ddl/domain/generator_test.go` — stubs for DDLG-04 (domain)
- [ ] `internal/ddl/enum/generator_test.go` — stubs for DDLG-04 (enum)
- [ ] `internal/ddl/policy/generator_test.go` — stubs for DDLG-04 (policy)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Generated DDL executes on real PG | DDLG-01 | Round-trip requires live DB | Execute generated CREATE TABLE against test DB, verify no errors |
| FK ALTER TABLE applies on live DB | DDLG-02 | Round-trip requires live DB | Execute generated ALTER TABLE after CREATE, verify constraint exists |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 10s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending

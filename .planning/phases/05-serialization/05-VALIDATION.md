---
phase: 5
slug: serialization
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-11
---

# Phase 5 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing (stdlib) |
| **Config file** | None (Go convention) |
| **Quick run command** | `go test -short ./internal/serialize/... ./internal/export/...` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test -short ./internal/serialize/... ./internal/export/...`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 05-01-01 | 01 | 0 | SRLZ-01 | unit | `go test ./internal/serialize/table/... -run TestSerialize` | ❌ W0 | ⬜ pending |
| 05-01-02 | 01 | 0 | SRLZ-01 | unit | `go test ./internal/serialize/table/... -run TestDeserialize` | ❌ W0 | ⬜ pending |
| 05-01-03 | 01 | 0 | SRLZ-02 | unit | `go test ./internal/serialize/view/... -run TestSerialize` | ❌ W0 | ⬜ pending |
| 05-01-04 | 01 | 0 | SRLZ-02 | unit | `go test ./internal/serialize/matview/... -run TestSerialize` | ❌ W0 | ⬜ pending |
| 05-01-05 | 01 | 0 | SRLZ-02 | unit | `go test ./internal/serialize/function/... -run TestSerialize` | ❌ W0 | ⬜ pending |
| 05-01-06 | 01 | 0 | SRLZ-02 | unit | `go test ./internal/serialize/trigger/... -run TestSerialize` | ❌ W0 | ⬜ pending |
| 05-01-07 | 01 | 0 | SRLZ-02 | unit | `go test ./internal/serialize/sequence/... -run TestSerialize` | ❌ W0 | ⬜ pending |
| 05-01-08 | 01 | 0 | SRLZ-02 | unit | `go test ./internal/serialize/compositetype/... -run TestSerialize` | ❌ W0 | ⬜ pending |
| 05-01-09 | 01 | 0 | SRLZ-02 | unit | `go test ./internal/serialize/domain/... -run TestSerialize` | ❌ W0 | ⬜ pending |
| 05-01-10 | 01 | 0 | SRLZ-02 | unit | `go test ./internal/serialize/enum/... -run TestSerialize` | ❌ W0 | ⬜ pending |
| 05-01-11 | 01 | 0 | SRLZ-02 | unit | `go test ./internal/serialize/policy/... -run TestSerialize` | ❌ W0 | ⬜ pending |
| 05-02-01 | 02 | 0 | SRLZ-03 | integration | `go test ./internal/export/... -run TestExportTableData` | ❌ W0 | ⬜ pending |
| 05-02-02 | 02 | 0 | SRLZ-04 | unit+integration | `go test ./internal/export/... -run TestPartition` | ❌ W0 | ⬜ pending |
| 05-02-03 | 02 | 0 | SRLZ-05 | unit | `go test ./internal/export/... -run TestChecksum` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/serialize/table/serializer_test.go` — stubs for SRLZ-01
- [ ] `internal/serialize/view/serializer_test.go` — stubs for SRLZ-02 (view)
- [ ] `internal/serialize/matview/serializer_test.go` — stubs for SRLZ-02 (matview)
- [ ] `internal/serialize/function/serializer_test.go` — stubs for SRLZ-02 (function)
- [ ] `internal/serialize/trigger/serializer_test.go` — stubs for SRLZ-02 (trigger)
- [ ] `internal/serialize/sequence/serializer_test.go` — stubs for SRLZ-02 (sequence)
- [ ] `internal/serialize/compositetype/serializer_test.go` — stubs for SRLZ-02 (type)
- [ ] `internal/serialize/domain/serializer_test.go` — stubs for SRLZ-02 (domain)
- [ ] `internal/serialize/enum/serializer_test.go` — stubs for SRLZ-02 (enum)
- [ ] `internal/serialize/policy/serializer_test.go` — stubs for SRLZ-02 (policy)
- [ ] `internal/serialize/foreignkey/serializer_test.go` — stubs for SRLZ-02 (fk)
- [ ] `internal/export/csv_test.go` — stubs for SRLZ-03, SRLZ-04, SRLZ-05

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| COPY FROM round-trip with real PG | SRLZ-03 | Requires live database | Export data.csv, then `COPY table FROM 'data.csv' WITH (FORMAT CSV)` and verify row count matches |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending

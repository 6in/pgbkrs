---
phase: 2
slug: schema-fetch-tables-sequences
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-11
---

# Phase 2 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go standard `testing` package |
| **Config file** | none — `go test ./...` uses standard Go test discovery |
| **Quick run command** | `go build ./... && go test ./internal/... -short` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~30 seconds (with TEST_DATABASE_URL; ~2s without) |

---

## Sampling Rate

- **After every task commit:** Run `go build ./... && go test ./internal/... -short`
- **After every plan wave:** Run `go test ./...` (with TEST_DATABASE_URL set)
- **Before `/gsd:verify-work`:** Full suite must be green with live PostgreSQL
- **Max feedback latency:** ~30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 2-01-01 | 01 | 0 | FETCH-01 | integration | `go test ./internal/fetch/table/... -run TestFetchTable -v` | ❌ W0 | ⬜ pending |
| 2-01-02 | 01 | 0 | FETCH-06 | integration | `go test ./internal/fetch/sequence/... -run TestFetchSequence -v` | ❌ W0 | ⬜ pending |
| 2-01-03 | 01 | 0 | FETCH-07 | integration | `go test ./internal/fetch/compositetype/... -run TestFetchCompositeType -v` | ❌ W0 | ⬜ pending |
| 2-01-04 | 01 | 0 | FETCH-08 | integration | `go test ./internal/fetch/domain/... -run TestFetchDomain -v` | ❌ W0 | ⬜ pending |
| 2-01-05 | 01 | 0 | FETCH-09 | integration | `go test ./internal/fetch/enum/... -run TestFetchEnum -v` | ❌ W0 | ⬜ pending |
| 2-02-01 | 02 | 1 | FETCH-01 | integration | `go test ./internal/fetch/table/... -run TestFetchTable -v` | ❌ W0 | ⬜ pending |
| 2-02-02 | 02 | 1 | FETCH-01 | integration | `go test ./internal/fetch/table/... -run TestFetchPartitionedTable -v` | ❌ W0 | ⬜ pending |
| 2-02-03 | 02 | 1 | FETCH-01 | integration | `go test ./internal/fetch/table/... -run TestFetchTableRLS -v` | ❌ W0 | ⬜ pending |
| 2-02-04 | 02 | 1 | FETCH-01 | integration | `go test ./internal/fetch/table/... -run TestFetchTableConstraints -v` | ❌ W0 | ⬜ pending |
| 2-02-05 | 02 | 1 | FETCH-01 | integration | `go test ./internal/fetch/table/... -run TestFetchTableIndexes -v` | ❌ W0 | ⬜ pending |
| 2-03-01 | 03 | 1 | FETCH-06 | integration | `go test ./internal/fetch/sequence/... -run TestFetchSequence -v` | ❌ W0 | ⬜ pending |
| 2-04-01 | 04 | 1 | FETCH-07 | integration | `go test ./internal/fetch/compositetype/... -run TestFetchCompositeType -v` | ❌ W0 | ⬜ pending |
| 2-04-02 | 04 | 1 | FETCH-07 | unit | `go build ./internal/fetch/...` | ❌ W0 | ⬜ pending |
| 2-05-01 | 05 | 1 | FETCH-08 | integration | `go test ./internal/fetch/domain/... -run TestFetchDomain -v` | ❌ W0 | ⬜ pending |
| 2-06-01 | 06 | 1 | FETCH-09 | integration | `go test ./internal/fetch/enum/... -run TestFetchEnum -v` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/fetch/table/fetcher_test.go` — stubs for FETCH-01 (columns, constraints, indexes, RLS, partitioning)
- [ ] `internal/fetch/sequence/fetcher_test.go` — stubs for FETCH-06
- [ ] `internal/fetch/compositetype/fetcher_test.go` — stubs for FETCH-07
- [ ] `internal/fetch/domain/fetcher_test.go` — stubs for FETCH-08
- [ ] `internal/fetch/enum/fetcher_test.go` — stubs for FETCH-09
- [ ] `internal/fetch/testhelpers/setup.go` — `setupTestSchema` shared fixture helper (creates tables, sequences, types; tears down after test)

All integration tests skip when `TEST_DATABASE_URL` env var is not set.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| RLS policies are visible in fetched table def | FETCH-01 | RLS policy detail is Phase 3+ scope; only the enabled flag is verified automatically | Connect to test DB, `ALTER TABLE t ENABLE ROW LEVEL SECURITY`, run fetcher, confirm `RLSDef.Enabled == true` |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending

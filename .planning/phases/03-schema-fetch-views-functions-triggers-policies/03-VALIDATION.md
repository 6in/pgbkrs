---
phase: 3
slug: schema-fetch-views-functions-triggers-policies
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-11
---

# Phase 3 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go standard `testing` package |
| **Config file** | none — `go test ./...` uses standard Go test discovery |
| **Quick run command** | `go build ./... && go test ./internal/... -short` |
| **Full suite command** | `TEST_DATABASE_URL=<dsn> go test ./...` |
| **Estimated runtime** | ~30 seconds (integration) |

---

## Sampling Rate

- **After every task commit:** Run `go build ./... && go test ./internal/... -short`
- **After every plan wave:** Run `TEST_DATABASE_URL=<dsn> go test ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 3-01-01 | 01 | 1 | FETCH-02 | integration | `go test ./internal/fetch/view/... -run TestFetchView -v` | ❌ Wave 0 | ⬜ pending |
| 3-01-02 | 01 | 1 | FETCH-03 | integration | `go test ./internal/fetch/matview/... -run TestFetchMatView -v` | ❌ Wave 0 | ⬜ pending |
| 3-01-03 | 01 | 1 | FETCH-03 | integration | `go test ./internal/fetch/matview/... -run TestFetchMatViewUnpopulated -v` | ❌ Wave 0 | ⬜ pending |
| 3-02-01 | 02 | 1 | FETCH-04 | integration | `go test ./internal/fetch/function/... -run TestFetchFunction -v` | ❌ Wave 0 | ⬜ pending |
| 3-03-01 | 03 | 1 | FETCH-05 | integration | `go test ./internal/fetch/trigger/... -run TestFetchTrigger -v` | ❌ Wave 0 | ⬜ pending |
| 3-03-02 | 03 | 1 | FETCH-05 | integration | `go test ./internal/fetch/trigger/... -run TestFetchTriggerExcludesInternal -v` | ❌ Wave 0 | ⬜ pending |
| 3-04-01 | 04 | 1 | FETCH-10 | integration | `go test ./internal/fetch/policy/... -run TestFetchPolicy -v` | ❌ Wave 0 | ⬜ pending |
| 3-04-02 | 04 | 1 | FETCH-10 | integration | `go test ./internal/fetch/policy/... -run TestFetchPolicyPublicRoles -v` | ❌ Wave 0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/fetch/view/fetcher_test.go` — stubs for FETCH-02
- [ ] `internal/fetch/matview/fetcher_test.go` — stubs for FETCH-03 (populated and unpopulated cases)
- [ ] `internal/fetch/function/fetcher_test.go` — stubs for FETCH-04
- [ ] `internal/fetch/trigger/fetcher_test.go` — stubs for FETCH-05 (including internal trigger exclusion)
- [ ] `internal/fetch/policy/fetcher_test.go` — stubs for FETCH-10 (including PUBLIC role sentinel)
- [ ] `internal/fetch/testhelpers/setup.go` — extend with view, matview, function, trigger, policy fixtures

*No new test framework install needed. All integration tests skip without `TEST_DATABASE_URL`.*

---

## Manual-Only Verifications

*If none: "All phase behaviors have automated verification."*

All phase behaviors have automated verification.

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending

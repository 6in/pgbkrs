---
phase: 10-schema-diff
verified: 2026-03-12T09:00:00Z
status: human_needed
score: 5/5 must-haves verified
human_verification:
  - test: "Run `pgbackup diff <backup-a> <backup-b>` with two real backup directories that differ"
    expected: "Output shows [追加オブジェクト], [削除オブジェクト], [変更オブジェクト] sections with correct Japanese labels and 4-space-indented detail lines matching spec 9.3"
    why_human: "Spec 9.3 visual format compliance (alignment, indentation, Japanese label accuracy) requires human review against the specification document. The automated tests confirm the section headers exist but do not verify exact column alignment or spacing."
  - test: "Run `pgbackup diff <backup-a> <backup-b>` where a backup contains function objects"
    expected: "Function entries in [追加オブジェクト] or [変更オブジェクト] sections show arg types in parentheses, e.g. `calc_total(integer)` per spec 9.3 example"
    why_human: "The `displayName` function in report.go accepts only core.ObjectHeader (which has no ArgTypes field), so it cannot produce the `function_name(argTypes)` display format shown in spec 9.3. The test suite passes because TestFormatReport does not test function display. Human must confirm whether this matters for the spec or whether plain name is acceptable."
---

# Phase 10: Schema Diff Verification Report

**Phase Goal:** Running `pgbackup diff <backup-a> <backup-b>` compares the two backup directories at the schema level and prints a structured, human-readable diff showing added, removed, and modified objects
**Verified:** 2026-03-12T09:00:00Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | loadBackup reads a manifest and deserializes all schema objects into a typed map keyed schema.kind.name | VERIFIED | loader.go:96-127 calls resolve.ReadManifest, iterates RestoreOrder, deserializes via serializerFor; TestLoadBackup passes |
| 2 | compareObjects detects additions (in B not A), removals (in A not B), and changes (same key, different content) | VERIFIED | compare.go:25-78; TestDiffAddedRemoved passes confirming 1 Added + 1 Removed for asymmetric maps |
| 3 | Table changes enumerate column add/remove/type-change/nullable-change/default-change, index add/remove, constraint add/remove, RLS enable/disable | VERIFIED | compare.go:213-382; diffTable, diffColumns, diffIndexes, diffUniqueConstraints, diffCheckConstraints all implemented; TestDiffTable passes checking "カラム追加" |
| 4 | Non-table kinds (view, function, sequence, etc.) report changed/unchanged via YAML byte comparison | VERIFIED | compare.go:384-416 diffNonTable uses yaml.Marshal for byte comparison; returns kind-appropriate Japanese labels; TestDiffNonTable passes |
| 5 | formatReport writes spec 9.3 output: header line, three conditional sections in Japanese with correct indentation | VERIFIED (with caveat) | report.go:13-72 writes "比較: A → B" header, conditional [追加オブジェクト]/[削除オブジェクト]/[変更オブジェクト] sections, 4-space-indented detail lines, (差分なし) for empty result; TestFormatReport passes; function arg display is incomplete (see human verification) |
| 6 | All five tests in diff_test.go are GREEN | VERIFIED | `go test -short ./internal/diff/... -v` confirms PASS for TestLoadBackup, TestDiffAddedRemoved, TestDiffTable, TestDiffNonTable, TestFormatReport |
| 7 | `pgbackup diff <backup-a> <backup-b>` does NOT require --host/--user/--dbname flags | VERIFIED | root.go:30-32 has early-return guard `if cmd.Name() == "diff" { return nil }` in PersistentPreRunE |
| 8 | cobra.ExactArgs(2) enforced — missing args returns argument error, not DB error | VERIFIED | diff.go:13 `Args: cobra.ExactArgs(2)` |
| 9 | Run() wires loadBackup + compareObjects + formatReport in sequence | VERIFIED | diff.go:9-21 calls all three; returns error if either backup load fails |
| 10 | No regressions in rest of codebase | VERIFIED | `go build ./...` exits 0; `go test ./...` all packages pass |

**Score:** 5/5 must-haves verified (10/10 observable truths verified; 1 item flagged for human confirmation)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/diff/diff.go` | Run() entry point, package declaration | VERIFIED | 21 lines; Run() calls loadBackup x2, compareObjects, formatReport; exports Run |
| `internal/diff/loader.go` | loadBackup reading manifest + deserializing via existing serializers | VERIFIED | 127 lines (min 60 required); resolve.ReadManifest wired; all 11 serializer kinds handled |
| `internal/diff/compare.go` | compareObjects, diffTable, diffNonTable helpers; DiffResult/ObjectChange types | VERIFIED | 416 lines (min 120 required); all helpers present and substantive |
| `internal/diff/report.go` | formatReport writing spec 9.3 format | VERIFIED | 79 lines (min 50 required); three conditional sections in Japanese |
| `internal/diff/diff_test.go` | Five tests covering DIFF-01 through DIFF-05 | VERIFIED | All five test functions present and GREEN |
| `cmd/pgbackup/cmd/diff.go` | diffCmd wired to diff.Run() with ExactArgs(2) | VERIFIED | cobra.ExactArgs(2) confirmed on line 13; diff.Run(args[0], args[1], os.Stdout) on line 15 |
| `cmd/pgbackup/cmd/root.go` | PersistentPreRunE skips DB connection when cmd.Name() == "diff" | VERIFIED | Guard at lines 30-32 confirmed |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `internal/diff/loader.go` | `internal/resolve.ReadManifest` | `resolve.ReadManifest(filepath.Join(backupDir, "_manifest.yaml"))` | WIRED | loader.go:97 |
| `internal/diff/loader.go` | `internal/serialize/*/Serializer.Deserialize` | `s.Deserialize(data)` in loop | WIRED | loader.go:119; all 11 serializer packages imported |
| `internal/diff/compare.go` | `*core.TableDef` type assertion | `a.(core.TableDef)` and `a.(*core.TableDef)` | WIRED | compare.go:198, 204 |
| `internal/diff/diff.go` | loadBackup + compareObjects + formatReport | Run() calls all three in sequence | WIRED | diff.go:10-19 |
| `cmd/pgbackup/cmd/diff.go` | `internal/diff.Run` | `diff.Run(args[0], args[1], os.Stdout)` | WIRED | diff.go:15 |
| `cmd/pgbackup/cmd/root.go` | PersistentPreRunE skip guard | `cmd.Name() == "diff"` early return nil | WIRED | root.go:30 |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| DIFF-01 | 10-01, 10-02 | バックアップ同士のスキーマ比較（データ差分は対象外） | SATISFIED | loadBackup reads only def.yaml schema files via manifest; data CSVs never loaded; TestLoadBackup GREEN |
| DIFF-02 | 10-01, 10-02 | オブジェクトの追加・削除検出 | SATISFIED | compareObjects set-difference logic in compare.go:28-43; TestDiffAddedRemoved GREEN |
| DIFF-03 | 10-01, 10-02 | テーブル変更検出（カラム追加・削除・型変更・NULL制約・デフォルト値、インデックス、制約、トリガー、RLS） | SATISFIED | diffTable covers columns/indexes/unique/check/RLS; trigger+policy post-processing pass moves attached objects into table Changed.Details; TestDiffTable GREEN |
| DIFF-04 | 10-01, 10-02 | ビュー・マテビュー・関数・シーケンス・型・ドメイン・ENUMの変更検出（変更あり/なし） | SATISFIED | diffNonTable via YAML byte comparison; kind-appropriate Japanese labels (本体変更あり/定義変更あり/設定変更あり); TestDiffNonTable GREEN |
| DIFF-05 | 10-01, 10-02, 10-03 | 差分結果の整形出力（仕様書9.3準拠） | SATISFIED (with caveat) | formatReport produces spec 9.3 structure; TestFormatReport GREEN; function arg display stub noted for human verification |

All five DIFF-xx requirements are mapped to plans in REQUIREMENTS.md (Phase 10) — no orphaned requirements.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `internal/diff/report.go` | 74-79 | `displayName` comment claims function arg display but returns `h.Name` only — comment/implementation mismatch | Warning | Function entries in Added/Removed/Changed sections will show plain name without arg types (e.g. `calc_total` instead of `calc_total(integer)`); spec 9.3 example shows arg types for functions |

No TODO/FIXME/PLACEHOLDER comments found. No empty return stubs. No console-log-only implementations.

### Human Verification Required

#### 1. Spec 9.3 Visual Format Compliance

**Test:** Run `pgbackup diff /path/to/backup_a /path/to/backup_b` against two real backups that have differences across multiple object types (tables with column changes, views with definition changes, added/removed functions).
**Expected:** Output matches spec 9.3 format exactly: `比較: <basename-a> → <basename-b>` header; blank line before each populated section; kind left-padded to 16 chars in Added/Removed; 4-space-indented detail lines in Changed; blank line between Changed entries.
**Why human:** The automated tests verify section header strings exist but do not assert exact column alignment, spacing between lines, or comparison against the full spec 9.3 template. Only a human reading the actual output against the spec document can confirm full compliance.

#### 2. Function Argument Display

**Test:** Run `pgbackup diff` on two backups where one has functions. Inspect the [追加オブジェクト] or [変更オブジェクト] output for any function entries.
**Expected (per spec 9.3):** Function entries display with arg types in parentheses: `function         : public.calc_total(integer)`
**Actual (per code):** `displayName(h core.ObjectHeader)` returns `h.Name` — arg types are not shown because `ObjectHeader` has no `ArgTypes` field and `formatReport` does not receive the full `ObjectDef`.
**Why human:** This is a known gap vs. the spec 9.3 example. A human must decide: (a) whether this is an acceptable deviation (ObjectHeader intentionally strips ArgTypes) or (b) whether `DiffResult.Added`/`Removed` should be changed from `[]core.ObjectHeader` to `[]core.ObjectDef` to enable arg display. The automated tests pass because they do not test function name formatting.

### Gaps Summary

No automated-verifiable gaps block goal achievement. All five DIFF requirements are implemented and tested. The `go build ./...` and `go test ./...` commands both pass with zero failures.

One quality concern exists: `displayName` in `report.go` cannot display function argument types because it receives only `core.ObjectHeader`. The spec 9.3 example shows `function: public.calc_total(integer)` with arg types. This is flagged for human review to determine if it represents a spec compliance gap requiring remediation.

---

_Verified: 2026-03-12T09:00:00Z_
_Verifier: Claude (gsd-verifier)_

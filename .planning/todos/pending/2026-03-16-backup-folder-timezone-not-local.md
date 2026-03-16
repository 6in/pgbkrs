---
created: 2026-03-16T07:54:30.858Z
title: バックアップフォルダの時刻がローカルのタイムゾーンになっていない
area: general
files:
  - internal/backup/orchestrator.go:69
---

## Problem

バックアップフォルダ名のタイムスタンプが UTC 固定になっている。

```go
// internal/backup/orchestrator.go:69
backupRoot := filepath.Join(outDir, time.Now().UTC().Format("backup_20060102_150405"))
```

日本時間（JST, UTC+9）で実行すると、フォルダ名が9時間ずれて表示される。例えば午前10時に実行しても `backup_20240101_010000` のようなフォルダ名になってしまう。

## Solution

`time.Now().UTC()` を `time.Now().Local()` に変更してローカルタイムゾーンを使用する。`_manifest.yaml` の `backup_at` フィールドは RFC3339 で UTC を維持してよい（機械可読な記録用途）。

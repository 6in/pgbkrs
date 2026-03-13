---
created: 2026-03-13T00:01:39.984Z
title: README.md を作成する（backup/restore/diff の使い方を含む）
area: docs
files:
  - cmd/pgbackup/cmd/backup.go
  - cmd/pgbackup/cmd/restore.go
  - cmd/pgbackup/cmd/diff.go
  - cmd/pgbackup/cmd/root.go
  - docs/pre/pgbackup_spec.md
---

## Problem

ユーザー向けドキュメントが存在しない。`docs/pre/pgbackup_spec.md` は内部仕様書であり、CLAUDE.md は開発者/AI向け。エンドユーザーがツールをインストールして使うための README.md がない。

## Solution

リポジトリルートに README.md を作成する。含めるべき内容：
- インストール方法（`go build ./cmd/pgbackup` またはバイナリ配布）
- 共通接続フラグ（`--host`, `--port`, `--user`, `--password`, `--dbname`）
- `backup` サブコマンド（`--output`, `--snapshot` フラグ）
- `restore` サブコマンド（`--input` 必須, `--pre-backup-dir`, `--schema`, `--object`, `--log-dir`）
- `diff` サブコマンド（引数2つ: backup-a, backup-b）
- バックアップディレクトリ構造の説明（YAML/CSV）
- 制限事項（`bytea`, `xml` 等を含むテーブルはスキップ）

---
created: 2026-03-13T00:37:50.257Z
title: windows,linux,macのクロスビルドに対応する
area: tooling
files: []
---

## Problem

現在 `go build ./cmd/pgbackup` でローカルプラットフォーム向けのバイナリしか生成できない。Windows / Linux / macOS（amd64 + arm64）向けのバイナリを配布するためのクロスビルド手順・自動化がない。

## Solution

`GOOS` / `GOARCH` を組み合わせたクロスビルドを対応する。Makefile ターゲット（例: `make build-all`）またはスクリプト（`scripts/build-all.sh`）を追加し、以下の組み合わせのバイナリを生成する:

- `linux/amd64`
- `linux/arm64`
- `darwin/amd64`
- `darwin/arm64`
- `windows/amd64`

GitHub Actions での自動ビルド・リリースアーティファクト添付も検討する。

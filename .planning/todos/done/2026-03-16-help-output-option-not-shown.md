---
created: 2026-03-16T07:56:22.563Z
title: --helpで--outputオプションが表示されていない
area: general
files:
  - cmd/pgbackup/cmd/backup.go
---

## Problem

`pgbackup backup --help` を実行すると `--output` は表示される。しかし `pgbackup --help`（ルートヘルプ）ではサブコマンドの Short 説明しか見えないため、ユーザーが `--output` オプションの存在に気づきにくい。

また `backup.go` の `--output` フラグは `Flags()` ではなく `PersistentFlags()` にすべきかどうか、あるいは Short 説明文に `--output` の存在を示す記述を加えるべきかの検討が必要。

実際の出力（`pgbackup --help`）:
```
Available Commands:
  backup      Back up PostgreSQL schema and data to YAML+CSV files
```

`--output` はここには出てこない。

## Solution

`backupCmd` の `Short` または `Long` フィールドに `--output` フラグの存在を示す記述を追加する。もしくは `Example` フィールドを使ってコマンド例（`--output /backups`）を補足する。Cobra の `Example` フィールドは `--help` 時に表示される。

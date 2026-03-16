---
created: 2026-03-16T07:58:57.332Z
title: サブコマンドに対して--helpが使える事を記述されていないかな
area: docs
files:
  - README.md
  - README-JP.md
---

## Problem

README.md / README-JP.md のどちらにも、各サブコマンドに対して `--help` が使えることが記載されていない。

Cobra は `pgbackup backup --help` のようにサブコマンド単位でヘルプを表示できるが、ユーザーがその使い方を知らないと全フラグを把握しづらい。特に `--output` や `--snapshot` などのフラグは `pgbackup --help` のルートヘルプには出てこないため、サブコマンドへの `--help` の案内が重要。

## Solution

README.md / README-JP.md の適切な箇所（「コマンド」セクションの冒頭や「基本的な使い方」など）に以下のような一文を追加する。

```
各サブコマンドのフラグ一覧は `pgbackup <command> --help` で確認できます。
```

英語版:
```
Run `pgbackup <command> --help` to see all available flags for each subcommand.
```

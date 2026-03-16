# pgbackup — PostgreSQL をオブジェクト単位で管理するバックアップツール

## なぜ自作したか

PostgreSQL のバックアップツールといえば `pg_dump` が定番です。しかし `pg_dump` はデータベース全体をひとつのファイルに固めてしまうため、

- 「このテーブルだけ昨日の状態に戻したい」
- 「スキーマがどう変わったか差分を見たい」
- 「バックアップの中身を人間が読める形で確認したい」

といった用途には不向きです。こうした課題を解決するために **pgbackup** を作りました。

---

## pgbackup の特徴

### 1. オブジェクト単位で YAML + CSV に保存

バックアップはスキーマ・オブジェクト種別ごとにファイルへ分割されます。

```
backup_20240101_120000/
└── public/
    ├── tables/
    │   ├── users/
    │   │   ├── def.yaml   ← DDL・制約・インデックス・権限
    │   │   └── data.csv   ← COPY フォーマットのデータ
    │   └── orders/
    │       ├── def.yaml
    │       └── data.csv
    ├── views/
    ├── functions/
    └── sequences/
```

`def.yaml` は人間が読める形式なので、Git で差分管理したり、エディタで中身を確認したりできます。

### 2. 依存関係を自動解決してリストア

テーブル・ビュー・外部キーなどオブジェクト間の依存関係を Kahn 法（トポロジカルソート）で解決し、正しい順序でリストアします。外部キー制約は循環参照を避けるため全テーブル作成後に一括適用されます。

### 3. スキーマ差分の比較

2つのバックアップディレクトリを比較して、スキーマの変化を確認できます。

```bash
pgbackup diff backup_20240101 backup_20240201
```

```
[追加オブジェクト]
  table : public.new_feature

[変更オブジェクト]
  table: public.users
    カラム追加 : phone_number varchar(20) nullable
    カラム変更 : status varchar(20) → varchar(50)
```

### 4. 部分リストア

テーブル1つだけ、またはスキーマ単位でリストアが可能です。依存チェーンも自動で解決されます。

```bash
# スキーマ単位
pgbackup restore --input ./backup_20240101 --schema public

# オブジェクト単位（依存オブジェクトも含めてリストア）
pgbackup restore --input ./backup_20240101 --object public.orders
```

---

## 基本的な使い方

### バックアップ

```bash
pgbackup backup --host localhost --dbname mydb --output /backups
```

一貫したスナップショットが必要な場合は `--snapshot` を付けます（REPEATABLE READ トランザクションで全オブジェクトを取得）。

```bash
pgbackup backup --host localhost --dbname mydb --output /backups --snapshot
```

### リストア

```bash
pgbackup restore --host localhost --dbname mydb \
  --input /backups/backup_20240101_120000
```

リストア前にリストア先 DB の自動バックアップが取られるので、万が一のときも安心です。

### スキーマ差分

```bash
pgbackup diff /backups/backup_20240101_120000 /backups/backup_20240201_120000
```

---

## 制限事項

`bytea`・`xml`・`pg_lsn`・`txid_snapshot` 型のカラムを含むテーブルは、CSV での表現が困難なためスキップされます。スキップされたテーブルは `_manifest.yaml` に記録されます。

---

## まとめ

pgbackup は「バックアップを人間が扱えるファイルとして管理する」という発想で作ったツールです。オブジェクト単位の可視性・部分リストア・スキーマ差分という3つの機能が、日々の運用や開発サイクルでの PostgreSQL 管理をシンプルにしてくれます。

ソースコードは GitHub で公開しています。フィードバックや Issue はお気軽にどうぞ。

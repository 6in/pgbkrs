# pgbackup

PostgreSQL のスキーマバックアップ・リストア・差分比較ツール。スキーマ定義を YAML、データを CSV としてエクスポートし、オブジェクト単位のリストアおよびバックアップ間のスキーマ比較を実現します。

## インストール

```bash
git clone https://github.com/pgbkrs/pgbackup
cd pgbackup
go build -o pgbackup ./cmd/pgbackup
```

## 共通接続フラグ

`diff` 以外のサブコマンドはデータベース接続が必要です。

| フラグ | デフォルト | 説明 |
|--------|-----------|------|
| `--host` | `localhost` | PostgreSQL ホスト |
| `--port` | `5432` | PostgreSQL ポート |
| `--user` | | PostgreSQL ユーザー |
| `--password` | | パスワード |
| `--dbname` | | 対象データベース名 |

## コマンド

### `backup`

PostgreSQL データベースをタイムスタンプ付きのディレクトリへ YAML + CSV 形式でバックアップします。

```bash
pgbackup backup --host localhost --dbname mydb --output /backups
```

**フラグ:**

| フラグ | デフォルト | 説明 |
|--------|-----------|------|
| `--output` | `.` | バックアップフォルダを作成するディレクトリ |
| `--snapshot` | false | `REPEATABLE READ` トランザクションで全オブジェクトを一貫したスナップショットとして取得する |

`--snapshot` を指定するとすべてのオブジェクトが同一トランザクション内で取得されデータ整合性が保証されます。指定しない場合はオブジェクトごとに個別取得するため高速ですが、厳密な整合性は保証されません。

### `restore`

バックアップディレクトリからデータベースをリストアします。

```bash
pgbackup restore --host localhost --dbname mydb --input /backups/backup_20240101_120000
```

リストア前に、リストア先データベースの事前バックアップが `--pre-backup-dir` に自動的に保存されます。

**フラグ:**

| フラグ | デフォルト | 説明 |
|--------|-----------|------|
| `--input` | **（必須）** | リストア元のバックアップディレクトリのパス |
| `--pre-backup-dir` | `.` | 事前バックアップの出力先ディレクトリ |
| `--schema` | | 指定スキーマのオブジェクトのみリストアする（例: `myschema`） |
| `--object` | | 指定オブジェクトとその依存チェーンをリストアする（`schema.name` 形式、例: `public.mytable`） |
| `--log-dir` | `.` | リストアログファイルの出力先ディレクトリ |

**部分リストア:** `--schema` でスキーマ単位、`--object` でオブジェクト単位のリストアが可能です。`--object` 指定時は依存オブジェクトを再帰的に解決してリストアします。依存先がリストア先に存在しない場合はエラーで停止します。

リストアログは `restore_YYYYMMDD_HHMMSS/` ディレクトリに出力されます（`drop.log`、`restore.log`、`summary.log`）。

### `diff`

2つのバックアップディレクトリのスキーマを比較します（データ差分は対象外）。

```bash
pgbackup diff /backups/backup_20240101_120000 /backups/backup_20240201_120000
```

テーブル（カラム・インデックス・制約・トリガー・RLS ポリシー）、ビュー、マテリアライズドビュー、関数、シーケンス、型・ドメイン・enum のオブジェクト追加・削除・変更を検出します。

## バックアップディレクトリ構造

バックアップ実行ごとにタイムスタンプ付きディレクトリが作成されます。

```
backup_YYYYMMDD_HHMMSS/
├── _manifest.yaml               # 依存関係・リストア順序・メタ情報
└── public/                      # スキーマ単位フォルダ
    ├── sequences/
    │   └── user_id_seq.yaml
    ├── tables/
    │   ├── users/
    │   │   ├── def.yaml         # DDL・制約・インデックス・権限
    │   │   └── data.csv         # COPY フォーマットデータ
    │   └── orders/              # パーティションテーブル（親）
    │       ├── def.yaml
    │       └── partitions/
    │           ├── orders_2024/
    │           │   ├── def.yaml
    │           │   └── data.csv
    │           └── orders_2025/
    │               ├── def.yaml
    │               └── data.csv
    ├── views/
    ├── materialized_views/
    ├── functions/
    ├── triggers/
    ├── policies/
    ├── types/
    ├── enums/
    └── domains/
```

## 制限事項

以下の型を含むカラムを持つテーブルはバックアップ対象から除外され、`_manifest.yaml` の `skipped_tables` に記録されます。

- `bytea`
- `xml`
- `pg_lsn`
- `txid_snapshot`

extension はバックアップ対象外です。

# pgbackup

PostgreSQL のスキーマバックアップ・リストア・差分比較ツール。スキーマ定義を YAML、データを CSV としてエクスポートし、オブジェクト単位のリストアおよびバックアップ間のスキーマ比較を実現します。

## インストール

```bash
git clone https://github.com/pgbkrs/pgbackup
cd pgbackup
go build -o pgbackup ./cmd/pgbackup
```

## 共通接続フラグ

`diff` と `data-diff` 以外のサブコマンドはデータベース接続が必要です。

| フラグ | デフォルト | 説明 |
|--------|-----------|------|
| `--host` | `localhost` | PostgreSQL ホスト |
| `--port` | `5432` | PostgreSQL ポート |
| `--user` | | PostgreSQL ユーザー |
| `--password` | | パスワード |
| `--dbname` | | 対象データベース名 |

## コマンド

> 各サブコマンドのフラグ一覧は `pgbackup <command> --help` で確認できます。

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

### `backup-ai`

AI への読み込みに最適化した形式で PostgreSQL データベースをバックアップします。

```bash
pgbackup backup-ai --host localhost --dbname mydb --output /backups
```

`backup` と同じスキーマ YAML ファイルを出力しますが、以下の点が異なります。

- **サンプル CSV データ:** 各テーブルのデータを最大 10 件に絞り、先頭にカラム名のヘッダー行を付けて出力します。AI のコンテキストを大量消費せずに代表的なデータを渡せます。
- **README.md:** バックアップルートにデータベース構造のサマリーを出力します。テーブル一覧・カラム数・主キー・外部キー関係・スキップされたテーブルと、AI への読み込み手順が記載されます。

**フラグ:**

| フラグ | デフォルト | 説明 |
|--------|-----------|------|
| `--output` | `.` | バックアップフォルダを作成するディレクトリ |
| `--snapshot` | false | `REPEATABLE READ` トランザクションで全オブジェクトを一貫したスナップショットとして取得する |

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

### `restore-tui`

ターミナル UI を使って、リストアするスキーマ・テーブルをインタラクティブに選択します。

```bash
pgbackup restore-tui --host localhost --dbname mydb --input /backups/backup_20240101_120000
```

バックアップ内の全スキーマ・テーブルをチェックボックス形式で表示します。選択後に Enter を押すとリストアを開始します。

**キー操作:**

| キー | 操作 |
|------|------|
| `↑` / `↓` または `k` / `j` | カーソル移動 |
| `space` | 選択トグル（スキーマ行では配下の全テーブルを一括切り替え） |
| `a` | 全選択 |
| `d` | 全解除 |
| `Enter` | 確定してリストア開始 |
| `q` / `Esc` | キャンセル |

**フラグ:**

| フラグ | デフォルト | 説明 |
|--------|-----------|------|
| `--input` | **（必須）** | リストア元のバックアップディレクトリのパス |
| `--pre-backup-dir` | `.` | 事前バックアップの出力先ディレクトリ |
| `--log-dir` | `.` | リストアログファイルの出力先ディレクトリ |

`restore` と同様に、処理前にリストア先データベースの事前バックアップが自動的に取得されます。スキーマ内の全テーブルが選択された場合はスキーマフィルターが適用され、個別テーブルが選択された場合はそれぞれ推移的依存オブジェクトとともにリストアされます。

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

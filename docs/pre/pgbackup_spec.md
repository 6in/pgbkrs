# pgbackup 仕様書

## 1. 概要

PostgreSQLデータベースのバックアップ・リストアツール。  
オブジェクト単位でYAML定義ファイルを出力し、データはCSV形式で保存する。  
スキーマ差分の比較機能も提供する。

---

## 2. 基本方針

| 項目 | 内容 |
|------|------|
| 実装言語 | Go（pgx/v5） |
| PostgreSQLバージョン依存 | なし |
| DDL取得方法 | pg_catalogから生成（`pg_get_functiondef` 等） |

---

## 3. バックアップ

### 3.1 ファイル形式

| 対象 | 形式 |
|------|------|
| 構造定義（DDL・制約・インデックス・権限等） | YAML（def.yaml） |
| データ | CSV（PostgreSQL COPY TO形式） |

### 3.2 対象オブジェクト種別

全オブジェクトはスキーマ単位フォルダに統一する。  
PostgreSQLにグローバルスコープのオブジェクトは存在しないため `_global/` は廃止。  
リストア順序はKahn法による依存解決で保証する。

#### スキーマ依存オブジェクト
- table
- view
- materialized view
- function
- trigger
- sequence
- type（複合型）
- domain
- enum
- policy（RLS）

#### 除外
- extension

### 3.3 スキップ対象テーブル

以下の型を含むカラムを持つテーブルはバックアップ対象から除外し、警告ログを出力する。

- `bytea`
- `xml`
- `pg_lsn`
- `txid_snapshot`

※ `bit` / `varbit` はCSVで問題なく扱えるため除外しない。

### 3.4 スナップショット（トランザクション）

`--snapshot` オプションで指定する。

| モード | 挙動 |
|--------|------|
| あり（`--snapshot`） | 全オブジェクトを同一スナップショットで取得。データ整合性が保証される。 |
| なし（デフォルト） | 各オブジェクトを個別に取得。高速だが厳密な整合性は保証されない。 |

### 3.5 シーケンス

バックアップ実行時点の値を取得する。  
スナップショット有無に関わらず、実行時点の値を記録する。

---

## 4. ディレクトリ構造

```
backup_YYYYMMDD_HHMMSS/
├── _manifest.yaml               # 依存関係・リストア順序・メタ情報
├── public/                      # スキーマ単位フォルダ
│   ├── sequences/
│   │   └── user_id_seq.yaml
│   ├── tables/
│   │   ├── users/
│   │   │   ├── def.yaml         # DDL・制約・インデックス・権限
│   │   │   └── data.csv         # COPYフォーマットデータ
│   │   └── orders/              # パーティションテーブル（親）
│   │       ├── def.yaml         # パーティション定義含む・データなし
│   │       └── partitions/      # 子テーブル群
│   │           ├── orders_2024/
│   │           │   ├── def.yaml
│   │           │   └── data.csv
│   │           └── orders_2025/
│   │               ├── def.yaml
│   │               └── data.csv
│   ├── views/
│   │   └── active_users.yaml
│   ├── materialized_views/
│   │   └── mv_sales.yaml
│   ├── functions/
│   │   └── calc_total.yaml
│   ├── triggers/
│   │   └── audit_trigger.yaml
│   ├── policies/                # RLSポリシー
│   ├── types/                   # 複合型
│   ├── enums/
│   └── domains/
└── myschema/
    └── ...                      # スキーマ単位で同様の構成
```

---

## 5. ファイル仕様

### 5.1 _manifest.yaml

```yaml
backup_at: "2024-01-01T12:00:00Z"
pg_version: "16.1"
tool_version: "1.0.0"
snapshot: true

skipped_tables:
  - schema: public
    name: binary_data
    reason: "contains bytea column: payload"

objects:
  - id: public.users
    kind: table
    depends_on: []
  - id: public.orders
    kind: table
    depends_on: [public.users]
  - id: public.active_users
    kind: view
    depends_on: [public.users, public.orders]

restore_order:
  - { schema: public, kind: enum,     name: status_type }
  - { schema: public, kind: sequence, name: user_id_seq }
  - { schema: public, kind: table,    name: users }
  - { schema: public, kind: table,    name: orders }
  - { schema: public, kind: fk,       name: orders_user_id_fkey, from_table: orders }
  - { schema: public, kind: view,     name: active_users }
```

### 5.2 テーブル def.yaml

```yaml
kind: table
schema: public
name: users

columns:
  - name: id
    type: bigint
    nullable: false
    default: "nextval('users_id_seq')"
  - name: email
    type: varchar(255)
    nullable: false
  - name: created_at
    type: timestamptz
    nullable: false
    default: now()

constraints:
  primary_key:
    name: users_pkey
    columns: [id]
  unique:
    - name: users_email_key
      columns: [email]

indexes:
  - name: idx_users_created_at
    columns: [created_at]
    method: btree

# FKはfrom_table側（orders）のdef.yamlにのみ記述する
# users/def.yaml にはFKエントリを持たない

triggers:
  - name: audit_users
    ref: public/triggers/audit_trigger.yaml

partitioning: null

rls:
  enabled: false
  policies: []

grants:
  - grantee: app_user
    privileges: [SELECT, INSERT, UPDATE]

data:
  file: data.csv
  columns: [id, email, created_at]
  row_count: 150000
  checksum: "sha256:abc123..."
```

### 5.3 data.csv

PostgreSQLの `COPY TO CSV` フォーマットをそのまま使用する。

```
1,alice@example.com,2024-01-01 00:00:00+09
2,bob@example.com,2024-01-02 00:00:00+09
```

---

## 6. 依存解決

### 6.1 アルゴリズム

インメモリDAG（有向非巡回グラフ）+ トポロジカルソート（Kahn法）を採用する。  
バックアップ時にリストア順序を計算し、`_manifest.yaml` に永続化する。

### 6.2 外部キーの特別処理

循環FK問題を回避するため、外部キー制約は独立したオブジェクトとして扱い、  
全テーブル作成・データ投入完了後に一括適用する。

---

## 7. リストア

### 7.1 基本動作

1. リストア先DBをバックアップ（事前に必須）
2. リストア先DBの現在のオブジェクト一覧を取得（漏れ検知用）
3. `restore_order` の逆順でDROP・drop.logに記録
4. DROP後のオブジェクト一覧と比較し、残存オブジェクトを警告ログ出力
5. `_manifest.yaml` の `restore_order` 順にCREATE
6. シーケンス値をSETVALで復元
7. データをCOPY FROMで投入
   - 通常テーブル：テーブルのdata.csvをCOPY FROM
   - パーティションテーブル：子テーブル単位でdata.csvをCOPY FROM
8. インデックス作成
9. 外部キー制約を一括適用
10. ビュー・関数・トリガー・ポリシー作成

### 7.2 リストア粒度

| 粒度 | 内容 |
|------|------|
| DB全体 | 全スキーマ・全オブジェクト |
| スキーマ指定 | 指定スキーマのオブジェクト全て |
| オブジェクト指定 | 指定オブジェクトとその依存チェーン |

### 7.3 トリガーの扱い

トリガーの無効化はツールの対象外とする。  
リストア前のオンライン接続遮断は運用側で実施する。

### 7.4 部分リストア時の依存解決

指定オブジェクトの `depends_on` を再帰的に辿り、依存チェーンを取得してリストアする。  
依存先オブジェクトがリストア先DBに存在しない場合は **エラーで停止** する。

---

## 8. ログ管理

### 8.1 ログファイル構成

リストア実行時に以下のログファイルを出力する。

```
restore_YYYYMMDD_HHMMSS/
├── drop.log       # DROP操作の詳細ログ
├── restore.log    # CREATE・COPY操作の詳細ログ
└── summary.log    # 全体サマリー
```

### 8.2 drop.log フォーマット

```
[2024-01-01 12:00:00] DROP START  schema=public kind=view  name=active_users → OK
[2024-01-01 12:00:01] DROP START  schema=public kind=table name=orders       → OK
[2024-01-01 12:00:01] DROP START  schema=public kind=table name=users        → OK
[2024-01-01 12:00:02] DROP START  schema=public kind=seq   name=user_id_seq  → OK
[2024-01-01 12:00:02] DROP START  schema=public kind=enum  name=status_type  → OK
[2024-01-01 12:00:02] DROP VERIFY 期待DROP数=5 実績DROP数=5 → 一致
```

### 8.3 漏れ検知

```
1. DROP前にリストア先DBの現在オブジェクト一覧を取得
2. restore_orderの逆順でDROPを実行・記録
3. DROP後に再度オブジェクト一覧を取得
4. 差分を検出 → バックアップ対象外のオブジェクトが残存している場合は警告

[2024-01-01 12:00:03] DROP VERIFY 警告: 以下のオブジェクトがDROP対象外として残存
  view : public.extra_view  ← バックアップに含まれていないオブジェクト
```

---

## 9. スキーマ差分機能

### 9.1 比較対象

バックアップ同士のスキーマ比較。  
データ差分は対象外。

### 9.2 検出項目

| オブジェクト種別 | 検出内容 |
|-----------------|---------|
| table | カラム（追加・削除・型変更・NULL制約・デフォルト値）、インデックス、制約（PK・UK・FK）、トリガー、RLSポリシー |
| view | 定義の変更あり / なし のみ |
| materialized view | 定義の変更あり / なし のみ |
| function | シグネチャ、本体の変更あり / なし のみ |
| sequence | 定義の変更あり / なし |
| type / domain / enum | 変更あり / なし |

### 9.3 出力例

```
比較: backup_20240101 → backup_20240201

[追加オブジェクト]
  table    : public.new_feature
  function : public.new_func(integer)

[削除オブジェクト]
  index    : public.idx_orders_old
  table    : public.obsolete_log

[変更オブジェクト]
  table: public.users
    カラム追加  : phone_number varchar(20) nullable
    カラム変更  : status type varchar(20) → varchar(50)

  table: public.orders
    カラム削除  : legacy_flag
    FK追加     : orders_shop_id_fkey → public.shops(id)

  function: public.calc_total(integer)
    本体変更あり

  view: public.active_users
    定義変更あり
```

---

## 10. コマンドパターン設計

### 10.1 概要

以下の3処理をコマンドパターンで実装する。  
オブジェクト種別ごとに3インターフェースを実装し、`CommandRegistry` で種別→コマンドを解決する。  
新オブジェクト種別の追加は `Registry.Register()` への1行追加のみで完結する。

### 10.2 ObjectDef 型設計

共通ヘッダ + インターフェース方式を採用する。

```go
// 共通ヘッダ
type ObjectHeader struct {
    Kind   ObjectKind
    Schema string
    Name   string
}

// 共通インターフェース
type ObjectDef interface {
    Header() ObjectHeader
}

// 各種別が ObjectHeader を embed して実装
type TableDef struct {
    ObjectHeader
    Columns     []ColumnDef
    Indexes     []IndexDef
    Constraints ConstraintsDef
    ForeignKeys []ForeignKeyDef  // from_table側のみ記述
    Triggers    []TriggerRef
    Partitioning *PartitionDef
    RLS         RLSDef
    Grants      []GrantDef
    Data        DataDef
}

type ViewDef struct {
    ObjectHeader
    Definition string
    Grants     []GrantDef
}

// 以下同様：
// MaterializedViewDef, FunctionDef, SequenceDef,
// TriggerDef, TypeDef, DomainDef, EnumDef, PolicyDef
```

| 処理軸 | 役割 |
|--------|------|
| `SchemaFetcher` | pg_catalogからオブジェクト定義を取得 |
| `Serializer` | ObjectDef ↔ YAML の変換 |
| `DDLGenerator` | ObjectDef → 実行可能なDDL文に変換 |

### 10.3 インターフェース定義

```go
// ① pg_catalogからスキーマ情報を取得
type SchemaFetcher interface {
    Fetch(ctx context.Context, conn *pgx.Conn, schema string) ([]ObjectDef, error)
}

// ② ObjectDef ↔ YAML
type Serializer interface {
    Serialize(def ObjectDef) ([]byte, error)
    Deserialize(data []byte) (ObjectDef, error)
}

// ③ ObjectDef → DDL文字列（複数文を返す: CREATE + ALTER等）
type DDLGenerator interface {
    GenerateDDL(def ObjectDef) ([]string, error)
    GenerateDrop(def ObjectDef) ([]string, error)
}

// TableDDLGenerator は2フェーズのDDLを返す
// Phase1: CREATE TABLE（FK制約なし）
// Phase2: ALTER TABLE ADD CONSTRAINT（FK制約、全テーブル作成後に適用）
// manifest.yaml の restore_order に fk エントリとして記録され
// リストアエンジンが最終フェーズで一括実行する
```

### 10.4 コマンド実体（オブジェクト種別ごと）

```go
type TableSchemaFetcher        struct{}  // implements SchemaFetcher
type TableSerializer           struct{}  // implements Serializer
type TableDDLGenerator         struct{}  // implements DDLGenerator

type ViewSchemaFetcher         struct{}
type ViewSerializer            struct{}
type ViewDDLGenerator          struct{}

type FunctionSchemaFetcher     struct{}
type FunctionSerializer        struct{}
type FunctionDDLGenerator      struct{}

// 以下同様：
// sequence, trigger, type, domain, enum, policy, materialized_view
```

### 10.5 CommandRegistry

```go
type ObjectKind string

const (
    KindTable            ObjectKind = "table"
    KindView             ObjectKind = "view"
    KindMaterializedView ObjectKind = "materialized_view"
    KindFunction         ObjectKind = "function"
    KindSequence         ObjectKind = "sequence"
    KindTrigger          ObjectKind = "trigger"
    KindType             ObjectKind = "type"
    KindDomain           ObjectKind = "domain"
    KindEnum             ObjectKind = "enum"
    KindPolicy           ObjectKind = "policy"
)

type CommandRegistry struct {
    fetchers    map[ObjectKind]SchemaFetcher
    serializers map[ObjectKind]Serializer
    generators  map[ObjectKind]DDLGenerator
}

func NewCommandRegistry() *CommandRegistry {
    r := &CommandRegistry{...}
    r.Register(KindTable,
        &TableSchemaFetcher{},
        &TableSerializer{},
        &TableDDLGenerator{},
    )
    r.Register(KindView,
        &ViewSchemaFetcher{},
        &ViewSerializer{},
        &ViewDDLGenerator{},
    )
    // 以下、種別ごとに同様に登録
    return r
}
```

### 10.6 呼び出し側

バックアップ・リストア処理はオブジェクト種別を意識せず統一的に扱える。

```go
// バックアップ
func Backup(ctx context.Context, conn *pgx.Conn, registry *CommandRegistry) error {
    for _, kind := range AllKinds {
        fetcher    := registry.Fetcher(kind)
        serializer := registry.Serializer(kind)

        defs, _ := fetcher.Fetch(ctx, conn, schema)
        for _, def := range defs {
            data, _ := serializer.Serialize(def)
            // ファイル書き出し
        }
    }
}

// リストア
func Restore(ctx context.Context, conn *pgx.Conn, registry *CommandRegistry, def ObjectDef) error {
    generator := registry.Generator(def.Kind)
    ddls, _   := generator.GenerateDDL(def)
    for _, ddl := range ddls {
        conn.Exec(ctx, ddl)
    }
}
```

---

## 11. Goプロジェクト構成

```
pgbackup/
├── cmd/
│   ├── backup/
│   │   └── main.go
│   ├── restore/
│   │   └── main.go
│   └── diff/
│       └── main.go
└── internal/
    ├── command/        # コマンドパターン実装
    │   ├── registry.go         # CommandRegistry
    │   ├── interfaces.go       # SchemaFetcher / Serializer / DDLGenerator
    │   ├── table/
    │   │   ├── fetcher.go
    │   │   ├── serializer.go
    │   │   └── generator.go
    │   ├── view/
    │   │   ├── fetcher.go
    │   │   ├── serializer.go
    │   │   └── generator.go
    │   ├── function/
    │   ├── sequence/
    │   ├── trigger/
    │   ├── type/
    │   ├── domain/
    │   ├── enum/
    │   ├── policy/
    │   └── materialized_view/
    ├── graph/          # DAG・トポロジカルソート（Kahn法）
    ├── diff/           # スキーマ差分比較
    └── model/          # 共通struct（TableDef, ColumnDef...）
```

---

## 12. 主要ライブラリ

| ライブラリ | 用途 |
|-----------|------|
| `github.com/jackc/pgx/v5` | PostgreSQL接続・COPY操作 |
| `gopkg.in/yaml.v3` | YAML読み書き |
| `encoding/csv` | CSV読み書き（標準ライブラリ） |


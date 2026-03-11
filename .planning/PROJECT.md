# pgbackup

## What This Is

PostgreSQLデータベースのバックアップ・リストアCLIツール。オブジェクト単位でYAML定義ファイルを出力し、データはCSV（COPY TO形式）で保存する。スキーマ差分の比較機能も提供する。Go言語（pgx/v5）で実装し、サブコマンド方式（`pgbackup backup` / `pgbackup restore` / `pgbackup diff`）で操作する。

## Core Value

PostgreSQLのスキーマ構造とデータを、人間が読めるファイル形式（YAML + CSV）でオブジェクト単位にバックアップし、依存関係を自動解決して正しい順序でリストアできること。

## Requirements

### Validated

(None yet — ship to validate)

### Active

- [ ] pg_catalogからオブジェクト定義を取得しYAMLで出力（table, view, materialized view, function, trigger, sequence, type, domain, enum, policy）
- [ ] テーブルデータをCOPY TO CSV形式で出力
- [ ] DAG + Kahn法による依存解決とリストア順序の計算
- [ ] _manifest.yamlに依存関係・リストア順序・メタ情報を記録
- [ ] FK制約を独立オブジェクトとして扱い循環FK問題を回避
- [ ] --snapshotオプションによるトランザクション一貫性モード
- [ ] スキップ対象テーブル（bytea, xml, pg_lsn, txid_snapshot列を含む）の検出と警告
- [ ] パーティションテーブルの親子構造対応（親はデータなし、子テーブル単位でデータ出力）
- [ ] シーケンス値のバックアップとSETVALによる復元
- [ ] リストア：restore_order逆順でDROP → 順序通りCREATE → データ投入 → インデックス → FK一括適用
- [ ] リストア粒度：DB全体 / スキーマ指定 / オブジェクト指定（依存チェーン自動解決）
- [ ] リストア前の自動バックアップ（事前必須）
- [ ] DROP漏れ検知（DROP前後のオブジェクト一覧比較）
- [ ] リストアログ出力（drop.log, restore.log, summary.log）
- [ ] スキーマ差分比較（バックアップ同士、データ差分は対象外）
- [ ] PostgreSQL接続は個別フラグ（--host, --port, --user, --password, --dbname）
- [ ] コマンドパターンによるオブジェクト種別ごとの処理分離（SchemaFetcher / Serializer / DDLGenerator）
- [ ] CommandRegistryで種別→コマンド解決、新種別追加はRegister()1行

### Out of Scope

- extension のバックアップ — PostgreSQLのextensionは運用管理の範疇
- データ差分比較 — スキーマ差分のみ対象
- トリガー無効化 — リストア前のオンライン接続遮断は運用側で実施
- モバイルアプリ・Web UI — CLIツールとして完結

## Context

- 実装言語: Go（pgx/v5）
- DDL取得方法: pg_catalogから生成（pg_get_functiondef等）、pg_dump非依存
- PostgreSQLバージョン依存なし
- テスト戦略: ロジック部分のユニットテスト + 実DB接続の結合テスト
- 主要ライブラリ: pgx/v5（PostgreSQL接続・COPY操作）、yaml.v3（YAML読み書き）、encoding/csv（標準ライブラリ）
- 仕様書: `docs/pre/pgbackup_spec.md` に詳細な設計仕様あり

## Constraints

- **言語**: Go — pgx/v5を使用
- **DDL取得**: pg_catalogからのみ取得（pg_dump不使用）
- **ファイル形式**: 構造定義はYAML、データはCSV（COPY TO形式）固定
- **コマンド形式**: サブコマンド方式（pgbackup backup / restore / diff）

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| サブコマンド方式（1バイナリ） | 配布・管理が容易 | — Pending |
| 個別フラグ接続（--host等） | PostgreSQL標準の接続パラメータに準拠 | — Pending |
| FK制約を独立オブジェクトとして分離 | 循環FK問題を根本的に回避 | — Pending |
| Kahn法トポロジカルソート | 安定した依存解決、循環検知が容易 | — Pending |
| bytea等のカラムを含むテーブルをスキップ | CSVで安全に扱えない型を排除 | — Pending |

---
*Last updated: 2026-03-11 after initialization*

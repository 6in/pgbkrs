# Requirements: pgbackup

**Defined:** 2026-03-11
**Core Value:** PostgreSQLのスキーマ構造とデータを人間が読めるYAML+CSV形式でオブジェクト単位にバックアップし、依存関係を自動解決して正しい順序でリストアできること

## v1 Requirements

### Foundation

- [x] **FOUND-01**: Goプロジェクト構成（cmd/internal構造）とサブコマンドCLI（backup/restore/diff）のセットアップ
- [x] **FOUND-02**: PostgreSQL接続（--host, --port, --user, --password, --dbname フラグ）
- [x] **FOUND-03**: コマンドパターン基盤（ObjectDef, SchemaFetcher, Serializer, DDLGenerator インターフェース）
- [x] **FOUND-04**: CommandRegistryによるオブジェクト種別→コマンド解決

### Backup - Schema Fetch

- [x] **FETCH-01**: テーブル定義のpg_catalog取得（カラム、制約、インデックス、権限、RLS、パーティション定義）
- [x] **FETCH-02**: ビュー定義のpg_catalog取得
- [x] **FETCH-03**: マテリアライズドビュー定義のpg_catalog取得
- [x] **FETCH-04**: 関数定義のpg_catalog取得（pg_get_functiondef）
- [x] **FETCH-05**: トリガー定義のpg_catalog取得
- [x] **FETCH-06**: シーケンス定義・現在値のpg_catalog取得
- [x] **FETCH-07**: 複合型定義のpg_catalog取得
- [x] **FETCH-08**: ドメイン定義のpg_catalog取得
- [x] **FETCH-09**: ENUM定義のpg_catalog取得
- [x] **FETCH-10**: RLSポリシー定義のpg_catalog取得

### Backup - Serialization

- [x] **SRLZ-01**: テーブルdef.yamlの出力（仕様書5.2準拠）
- [x] **SRLZ-02**: ビュー・マテビュー・関数・トリガー等のYAML出力
- [x] **SRLZ-03**: テーブルデータのCOPY TO CSV出力
- [x] **SRLZ-04**: パーティションテーブルの親子構造出力（親はデータなし、子テーブル単位でCSV）
- [x] **SRLZ-05**: data.csvのsha256チェックサム計算と行数記録

### Backup - Orchestration

- [x] **BKUP-01**: スキーマ単位のディレクトリ構造生成（仕様書セクション4準拠）
- [x] **BKUP-02**: スキップ対象テーブル検出（bytea, xml, pg_lsn, txid_snapshot列）と警告ログ
- [ ] **BKUP-03**: --snapshotオプションによるトランザクション一貫性モード
- [ ] **BKUP-04**: スナップショットなしモード（デフォルト、各オブジェクト個別取得）

### Dependency Resolution

- [x] **DEPS-01**: インメモリDAG構築（オブジェクト間の依存関係）
- [x] **DEPS-02**: Kahn法トポロジカルソートによるリストア順序計算
- [x] **DEPS-03**: FK制約の独立オブジェクト化（循環FK回避）
- [x] **DEPS-04**: _manifest.yaml生成（依存関係、リストア順序、メタ情報、スキップ情報）

### Restore

- [ ] **REST-01**: リストア前の自動バックアップ実行
- [ ] **REST-02**: restore_order逆順でのDROP実行とdrop.log記録
- [ ] **REST-03**: DROP漏れ検知（DROP前後のオブジェクト一覧比較、残存オブジェクト警告）
- [ ] **REST-04**: restore_order順でのCREATE実行
- [ ] **REST-05**: COPY FROMによるデータ投入（通常テーブル + パーティション子テーブル）
- [ ] **REST-06**: シーケンス値のSETVAL復元
- [ ] **REST-07**: インデックス作成
- [ ] **REST-08**: FK制約の一括適用（全テーブル作成・データ投入後）
- [ ] **REST-09**: ビュー・関数・トリガー・ポリシーの作成
- [ ] **REST-10**: リストア粒度対応（DB全体 / スキーマ指定 / オブジェクト指定）
- [ ] **REST-11**: 部分リストア時の依存チェーン自動解決（depends_on再帰走査）
- [ ] **REST-12**: 依存先オブジェクト未存在時のエラー停止
- [ ] **REST-13**: リストアログ出力（drop.log, restore.log, summary.log）

### Schema Diff

- [ ] **DIFF-01**: バックアップ同士のスキーマ比較（データ差分は対象外）
- [ ] **DIFF-02**: オブジェクトの追加・削除検出
- [ ] **DIFF-03**: テーブル変更検出（カラム追加・削除・型変更・NULL制約・デフォルト値、インデックス、制約、トリガー、RLS）
- [ ] **DIFF-04**: ビュー・マテビュー・関数・シーケンス・型・ドメイン・ENUMの変更検出（変更あり/なし）
- [ ] **DIFF-05**: 差分結果の整形出力（仕様書9.3準拠）

### DDL Generation

- [x] **DDLG-01**: テーブルCREATE DDL生成（FK制約なし = Phase1）
- [x] **DDLG-02**: テーブルFK制約ALTER TABLE DDL生成（Phase2）
- [x] **DDLG-03**: ビュー・マテビュー・関数・トリガー等のCREATE DDL生成
- [x] **DDLG-04**: 全オブジェクト種別のDROP DDL生成

## v2 Requirements

### Enhanced Features

- **ENH-01**: DSN文字列接続対応（--dsn "postgres://..."）
- **ENH-02**: 環境変数対応（PGHOST, PGPORT等）
- **ENH-03**: 進捗表示（プログレスバー）
- **ENH-04**: 並列バックアップ（複数テーブルの同時COPY TO）

## Out of Scope

| Feature | Reason |
|---------|--------|
| extensionのバックアップ | PostgreSQLのextensionは運用管理の範疇 |
| データ差分比較 | スキーマ差分のみを対象とする設計判断 |
| トリガー無効化 | リストア前のオンライン接続遮断は運用側で実施 |
| Web UI / GUI | CLIツールとして完結 |
| pg_dump連携 | pg_catalog直接参照の方針 |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| FOUND-01 | Phase 1 | Complete |
| FOUND-02 | Phase 1 | Complete |
| FOUND-03 | Phase 1 | Complete |
| FOUND-04 | Phase 1 | Complete |
| FETCH-01 | Phase 2 | Complete |
| FETCH-06 | Phase 2 | Complete |
| FETCH-07 | Phase 2 | Complete |
| FETCH-08 | Phase 2 | Complete |
| FETCH-09 | Phase 2 | Complete |
| FETCH-02 | Phase 3 | Complete |
| FETCH-03 | Phase 3 | Complete |
| FETCH-04 | Phase 3 | Complete |
| FETCH-05 | Phase 3 | Complete |
| FETCH-10 | Phase 3 | Complete |
| DDLG-01 | Phase 4 | Complete |
| DDLG-02 | Phase 4 | Complete |
| DDLG-03 | Phase 4 | Complete |
| DDLG-04 | Phase 4 | Complete |
| SRLZ-01 | Phase 5 | Complete |
| SRLZ-02 | Phase 5 | Complete |
| SRLZ-03 | Phase 5 | Complete |
| SRLZ-04 | Phase 5 | Complete |
| SRLZ-05 | Phase 5 | Complete |
| DEPS-01 | Phase 6 | Complete |
| DEPS-02 | Phase 6 | Complete |
| DEPS-03 | Phase 6 | Complete |
| DEPS-04 | Phase 6 | Complete |
| BKUP-01 | Phase 7 | Complete |
| BKUP-02 | Phase 7 | Complete |
| BKUP-03 | Phase 7 | Pending |
| BKUP-04 | Phase 7 | Pending |
| REST-01 | Phase 8 | Pending |
| REST-02 | Phase 8 | Pending |
| REST-04 | Phase 8 | Pending |
| REST-05 | Phase 8 | Pending |
| REST-06 | Phase 8 | Pending |
| REST-07 | Phase 8 | Pending |
| REST-08 | Phase 8 | Pending |
| REST-09 | Phase 8 | Pending |
| REST-03 | Phase 9 | Pending |
| REST-10 | Phase 9 | Pending |
| REST-11 | Phase 9 | Pending |
| REST-12 | Phase 9 | Pending |
| REST-13 | Phase 9 | Pending |
| DIFF-01 | Phase 10 | Pending |
| DIFF-02 | Phase 10 | Pending |
| DIFF-03 | Phase 10 | Pending |
| DIFF-04 | Phase 10 | Pending |
| DIFF-05 | Phase 10 | Pending |

**Coverage:**
- v1 requirements: 47 total
- Mapped to phases: 47
- Unmapped: 0 ✓

---
*Requirements defined: 2026-03-11*
*Last updated: 2026-03-11 after roadmap creation*

# 🐘 pgbackup — PostgreSQL をオブジェクト単位で管理するバックアップツールを作りました！

こんにちは！今回は自作の PostgreSQL バックアップツール **pgbackup** を紹介させてください 🎉

---

## 🤔 なぜ自作したの？

PostgreSQL のバックアップといえば `pg_dump` が鉄板ですよね。でも使っていてこんな不満、ありませんか？

- 😩「このテーブルだけ昨日の状態に戻したい…でも全体リストアしかできない」
- 😩「スキーマがいつの間にか変わってる。どこが変わったか見たい」
- 😩「バックアップファイル、バイナリで中身が全然読めない」

`pg_dump` はデータベース全体をひとつのファイルに固めてしまうので、こういった「ピンポイントな操作」がとにかく苦手です。

そこで作ったのが **pgbackup** です！✨

---

## 🌟 pgbackup の特徴

### 1️⃣ オブジェクト単位で YAML + CSV に保存

バックアップがスキーマ・オブジェクト種別ごとにファイルへ分割されます。人間が読める形式なので、Git で差分管理したりエディタで確認したりできます 👀

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

### 2️⃣ 依存関係を自動解決してリストア 🧩

テーブル・ビュー・外部キーなどオブジェクト間の依存関係を Kahn 法（トポロジカルソート）で自動解決して、正しい順序でリストアします。外部キー制約の循環参照問題も、全テーブル作成後に一括適用することでスマートに回避しています💡

### 3️⃣ スキーマ差分の比較 🔍

2つのバックアップを比較して、スキーマの変化をひと目で確認できます！

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

「あれ、このカラムいつ増えたっけ？」がすぐわかります 🙌

### 4️⃣ 部分リストア 🎯

テーブル1つだけ、またはスキーマ単位でリストアできます。依存チェーンも自動で解決してくれるので安心です！

```bash
# スキーマ単位
pgbackup restore --input ./backup_20240101 --schema public

# オブジェクト単位（依存オブジェクトも含めてリストア）
pgbackup restore --input ./backup_20240101 --object public.orders
```

---

## 🚀 基本的な使い方

### バックアップ

```bash
pgbackup backup --host localhost --dbname mydb --output /backups
```

データ整合性を完全に保証したいときは `--snapshot` を付けましょう。REPEATABLE READ トランザクションで全オブジェクトを一貫して取得します 📸

```bash
pgbackup backup --host localhost --dbname mydb --output /backups --snapshot
```

### リストア

```bash
pgbackup restore --host localhost --dbname mydb \
  --input /backups/backup_20240101_120000
```

リストア前にリストア先 DB の自動バックアップが取られるので、万が一のときも安心です 🛡️

### スキーマ差分

```bash
pgbackup diff /backups/backup_20240101_120000 /backups/backup_20240201_120000
```

---

## ⚠️ 制限事項

`bytea`・`xml`・`pg_lsn`・`txid_snapshot` 型のカラムを含むテーブルは、CSV での表現が困難なためスキップされます。スキップされたテーブルは `_manifest.yaml` に記録されます。

---

## 🛠️ GSD フレームワークで開発しました

pgbackup の開発には **GSD（Get Shit Done）** という Claude Code 向けのワークフローフレームワークを活用しました！

GSD は AI との協業を「フェーズ管理 → 計画 → 実行 → 検証」という流れで構造化してくれるツールです。ロードマップの作成から各フェーズの計画・実行まで、Claude がワークフローに沿って自律的に動いてくれます。

「アイデアを思いついたら `/gsd:add-todo` でキャプチャ → あとで `/gsd:check-todos` で確認して作業開始」という流れがとても気持ちよくて、このブログ記事自体も GSD の todo 管理から生まれました 😄

GSD を使ったことで、設計の迷いや手戻りが減り、スムーズに開発を進めることができました ✨

---

## 🎉 まとめ

pgbackup は「バックアップを人間が扱えるファイルとして管理する」という発想から生まれたツールです。

- 📂 オブジェクト単位の可視性
- 🎯 部分リストア
- 🔍 スキーマ差分比較

この3つの機能が、日々の運用や開発サイクルでの PostgreSQL 管理をぐっとシンプルにしてくれると思います！

ソースコードは GitHub で公開しています。ぜひ使ってみてください 🙏 フィードバックや Issue もお気軽にどうぞ！

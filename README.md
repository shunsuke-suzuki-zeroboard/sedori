# sedori

日本のECサイト間の価格差（せどりチャンス）を自動検出し、LINEで通知するツール。

## 機能

- Amazon, 楽天市場, メルカリの横断価格比較
- 設定可能な利益率・価格差の閾値フィルタリング
- LINE Messaging APIによるリアルタイム通知
- 定期監視モード（デフォルト30分間隔）

## 必要要件

- Go 1.24+
- LINE Messaging API のチャネルアクセストークンとユーザーID
- Amazon PA-API 5.0 のアクセスキー（Amazon検索を使う場合）
- 楽天ウェブサービス アプリケーションID（楽天検索を使う場合）

## セットアップ

```bash
cp config.json.example config.json
# config.json を編集してAPIキーを設定

go build -o sedori ./cmd/sedori
```

## 使い方

```bash
# LINE通知の接続テスト
./sedori -config config.json -test-line

# 1回だけ実行
./sedori -config config.json -once

# 定期監視（デフォルト30分間隔）
./sedori -config config.json
```

## CLIフラグ

| フラグ | デフォルト | 説明 |
|-------|-----------|------|
| `-config` | `config.json` | 設定ファイルのパス |
| `-once` | `false` | 1回実行して終了 |
| `-test-line` | `false` | LINE通知テストを送信して終了 |

## 設定

`config.json.example` を参照。主な設定項目:

- `keywords`: 検索キーワード一覧
- `min_profit_rate`: 通知する最低利益率（%、デフォルト10%）
- `min_price_diff`: 通知する最低価格差（円、デフォルト500円）
- `interval_minutes`: 監視間隔（分、デフォルト30分）
- 各サイトの `enabled` フラグで個別に有効/無効を切り替え

## アーキテクチャ

クリーンアーキテクチャに基づいたレイヤー構成。詳細は [CLAUDE.md](./CLAUDE.md) を参照。

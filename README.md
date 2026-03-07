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

## LINE 通知の設定手順

### 1. LINE Developers でチャネルを作成

1. [LINE Developers Console](https://developers.line.biz/console/) にログイン
2. プロバイダーを作成（初回のみ）
3. 「Messaging API」チャネルを新規作成
4. チャネル設定画面で以下を取得:
   - **チャネルアクセストークン**: 「Messaging API設定」タブ → 「チャネルアクセストークン（長期）」を発行
   - **あなたのユーザーID**: 「チャネル基本設定」タブ → 「あなたのユーザーID」

### 2. LINE 公式アカウントを友だち追加

チャネル作成時に生成される LINE 公式アカウントを、通知を受け取りたい LINE アカウントで友だち追加する。「Messaging API設定」タブの QR コードからスキャンできる。

### 3. config.json に設定

```json
{
  "line": {
    "channel_access_token": "取得したチャネルアクセストークン",
    "user_id": "取得したユーザーID"
  }
}
```

### 4. 通知テスト

```bash
./sedori -config config.json -test-line
```

成功すると LINE に2通のテストメッセージが届く:
1. テキストメッセージ（接続確認）
2. サンプルのせどりチャンス通知（フォーマット確認）

### トラブルシューティング

| エラー | 原因 | 対処 |
|-------|------|------|
| `401 Invalid channel access token` | トークンが無効 | LINE Developers でトークンを再発行 |
| `400 Bad Request` | ユーザーIDが不正 | 「あなたのユーザーID」（U から始まる文字列）を確認 |
| `request failed` | ネットワークエラー | インターネット接続・プロキシ設定を確認 |
| メッセージが届かない | 友だち追加していない | LINE 公式アカウントを友だち追加する |

## アーキテクチャ

クリーンアーキテクチャに基づいたレイヤー構成。詳細は [CLAUDE.md](./CLAUDE.md) を参照。

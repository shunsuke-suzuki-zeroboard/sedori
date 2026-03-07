# sedori 開発ガイド

日本のECサイト（Amazon, 楽天, メルカリ）間の価格差を監視し、LINE通知するせどり支援ツール。
Go 1.24.7、外部依存なし（stdlib only）。モジュール: `github.com/shunsuke-suzuki-zeroboard/sedori`

## コマンド

```bash
go build -o sedori ./cmd/sedori     # ビルド
go test ./...                        # テスト実行
go vet ./...                         # 静的解析
./sedori -config config.json         # 定期監視モード
./sedori -config config.json -once   # 1回実行して終了
./sedori -config config.json -test-line  # LINE通知テスト
```

## 開発ワークフロー

GitHub Flow に基づき、機能ブランチ上で設計者・実装者・コードレビュアーの3役割を分担して進める。

### ブランチ運用（GitHub Flow）

```
main（常にデプロイ可能） ← feature branch（PR経由でマージ）
```

#### ルール

- **`main` ブランチは常にデプロイ可能な状態を維持する。** 直接コミットは禁止
- 作業は必ず `main` から機能ブランチを切って行う
- 機能ブランチ名: `feature/<機能名>`、`fix/<修正内容>`、`docs/<ドキュメント内容>`
- 1つの機能ブランチ = 1つの目的。複数の無関係な変更を混ぜない
- こまめにコミットし、早めにプッシュして作業を可視化する
- 実装完了後、Pull Request を作成してレビューを依頼する
- PR マージは **Squash and merge** を使用し、マージ後に機能ブランチを削除する

#### ブランチ操作の流れ

```bash
# 1. main を最新化
git checkout main && git pull origin main

# 2. 機能ブランチを作成
git checkout -b feature/add-yahoo-scraper

# 3. 実装・コミット（こまめに）
git add <files> && git commit -m "feat: Yahoo scraper の基本実装"

# 4. プッシュ
git push -u origin feature/add-yahoo-scraper

# 5. GitHub で Pull Request を作成 → レビュー → Squash and merge

# 6. ローカルを整理
git checkout main && git pull origin main
git branch -d feature/add-yahoo-scraper
```

#### 並行作業

複数の機能を並行して進める場合、それぞれ独立した機能ブランチで作業する。ブランチ間の依存が生じる場合は、先にマージすべきブランチを明確にする。

```
main ─────────────────────────────────────
  ├─ feature/add-yahoo-scraper    （並行作業 A）
  └─ feature/add-slack-notifier   （並行作業 B）
```

### 進行フロー（各ブランチ内）

```
設計 → 実装 → レビュー → (指摘があれば修正 → 再レビュー) → マージ
```

### 1. 設計者（Designer）

要件を分析し、実装方針を決定する。

- **要件分析**: 何を実現するか、既存機能への影響範囲を特定
- **変更対象ファイル一覧**: どのレイヤーのどのファイルを追加・変更するか列挙
- **インターフェース設計**: 新規・変更する型・関数のシグネチャを定義
- **依存方向の確認**: クリーンアーキテクチャの依存ルール（`domain` ← `usecase` ← `adapter`）に違反しないか検証
- **既存パターンとの整合**: スクレイパー追加手順・通知先追加手順など、既存の拡張パターンに沿っているか確認

**成果物**: 実装方針（変更ファイル一覧、型/関数シグネチャ、データフロー）

### 2. 実装者（Implementer）

設計に従ってコードを書き、テストを作成する。

- **コーディング規約への準拠**: 「コーディング規約」セクションのルールに従う
- **テスト作成**: 「テスト規約」セクションに従い、実装と同時にテストを書く
- **ビルド検証**: 実装完了後に以下を通過させる
  ```bash
  go build ./...
  go vet ./...
  go test ./...
  ```
- **設計からの逸脱禁止**: 設計で決めた方針を変更する場合は、設計者に差し戻す
- **PR 作成**: 実装完了後、Pull Request を作成しレビュアーを指定する

### 3. コードレビュアー（Reviewer）

「コードレビューガイド」の全チェック項目に基づき PR をレビューする。

- **設計との整合性**: 実装が設計方針に沿っているか確認
- **チェックリスト確認**: アーキテクチャ、エラー処理、並行処理、セキュリティ、パフォーマンス、可読性の6カテゴリを検証
- **指摘事項**: 問題があれば具体的な修正案とともに実装者に差し戻す
- **承認**: 全項目をクリアしたら PR を Approve し、Squash and merge を実行

## アーキテクチャ（クリーンアーキテクチャ）

### 依存の方向

```
cmd/main (DI) → usecase → domain ← adapter ← infrastructure
```

**内側のレイヤーは外側を知らない。** domain は他パッケージに一切依存しない。

### パッケージ責務

| レイヤー | パッケージ | 責務 |
|---------|-----------|------|
| domain | `internal/domain` | エンティティ（Product, PriceDiff, Site）、インターフェース（ProductSearcher, Notifier） |
| usecase | `internal/usecase` | 検索→比較→通知のオーケストレーション（ArbitrageUseCase） |
| adapter | `internal/adapter/scraper` | 各ECサイトのAPI呼び出し（Amazon PA-API, 楽天, メルカリ） |
| adapter | `internal/adapter/notifier` | LINE Messaging API 通知 |
| infrastructure | `internal/infrastructure/config` | JSON設定ファイルの読み込み |
| - | `cmd/sedori` | エントリポイント、DI、定期実行ループ、シグナルハンドリング |

### ディレクトリ構成

```
cmd/sedori/main.go
internal/
  domain/
    product.go          # Site, Product, PriceDiff
    repository.go       # ProductSearcher, Notifier インターフェース
  usecase/
    arbitrage.go        # ArbitrageUseCase
  adapter/
    scraper/
      amazon.go         # AmazonScraper (PA-API 5.0, AWS Sig V4)
      rakuten.go        # RakutenScraper (楽天市場API)
      mercari.go        # MercariScraper (公開検索API)
    notifier/
      line.go           # LINENotifier (Messaging API v2)
  infrastructure/
    config/
      config.go         # Config, Load()
```

## コーディング規約

- **エラーラップ**: `fmt.Errorf("package: context: %w", err)` — プレフィックスは小文字のパッケージ/コンポーネント名（`amazon:`, `line:`, `usecase:`）
- **ファクトリ関数**: `NewXxxType(params) *XxxType` — 具体的なポインタを返す
- **HTTPクライアント**: 各adapter構造体が `*http.Client` を保持、`Timeout: 10 * time.Second`
- **Context伝播**: HTTP呼び出しには `http.NewRequestWithContext(ctx, ...)` を使う
- **並行処理**: `sync.WaitGroup` + `sync.Mutex`、チャネルは使わない
- **JSON structタグ**: 設定ファイルは `snake_case`、外部API応答はAPI仕様に従う
- **API応答型**: unexported（パッケージプライベート）、同一ファイル内に定義
- **価格**: `int`（円単位、小数なし）
- **依存関係**: **stdlib のみ**。サードパーティモジュールを追加しない
- **ログ**: `log.Printf` で運用ログ、`fmt.Printf` はmainのコンソール出力のみ
- **インターフェース定義**: `domain` パッケージに置く。adapter パッケージには置かない

## スクレイパー追加手順

1. `internal/domain/product.go` に `Site` 定数を追加（例: `SiteYahoo Site = "yahoo"`）
2. `internal/adapter/scraper/yahoo.go` を作成:
   - unexported な API応答構造体を定義
   - `YahooScraper` 構造体（`client *http.Client` フィールド必須）
   - `NewYahooScraper(...)` ファクトリ関数
   - `Site() domain.Site` — 新しい定数を返す
   - `Search(ctx context.Context, keyword string) ([]domain.Product, error)`
   - エラーは `fmt.Errorf("yahoo: ...: %w", err)` でラップ
3. `internal/infrastructure/config/config.go` に `YahooConfig` 構造体と `Config.Yahoo` フィールドを追加
4. `cmd/sedori/main.go` の `buildSearchers()` に条件分岐を追加
5. `internal/adapter/notifier/line.go` の `siteLabel()` にケースを追加
6. `config.json.example` に新セクションを追加

## 通知先追加手順

1. `internal/adapter/notifier/slack.go` などを作成し `domain.Notifier` インターフェースを実装
2. `internal/infrastructure/config/config.go` に設定構造体を追加
3. `cmd/sedori/main.go` で DI（生成して usecase に渡す）

## テスト規約

- stdlib `testing` パッケージのみ使用（testify, gomock 不可）
- テストファイルはソースと同じディレクトリに配置: `xxx_test.go`
- scraper テスト: `net/http/httptest.NewServer` でHTTPレスポンスをモック
- comparator/usecase テスト: 既知の商品セットで期待結果を検証
- config テスト: `os.CreateTemp` で一時ファイルを使用

## コードレビューガイド

コード変更時に以下の観点で確認する。

### アーキテクチャ

- [ ] `domain` パッケージが外部パッケージに依存していないか（import に `adapter`, `usecase`, `infrastructure` が含まれていないか）
- [ ] `adapter` パッケージが `usecase` に依存していないか（依存方向: adapter → domain のみ）
- [ ] `usecase` パッケージが `adapter` や `infrastructure` に依存していないか（依存方向: usecase → domain のみ）
- [ ] インターフェースは `domain` パッケージに定義されているか（adapter 側に定義していないか）
- [ ] 新しい外部サービス連携は adapter レイヤーに配置されているか

### エラー処理

- [ ] エラーは `fmt.Errorf("component: context: %w", err)` でラップされているか
- [ ] エラーが握り潰されていないか（`_ = doSomething()` のような無視がないか）
- [ ] `resp.Body.Close()` が `defer` で確実に呼ばれているか
- [ ] HTTP レスポンスのステータスコードを検証してからボディを読んでいるか
- [ ] エラーメッセージに十分なコンテキスト（操作内容、対象リソース）が含まれているか

### 並行処理

- [ ] goroutine 内で `ctx` を伝播しているか（`http.NewRequestWithContext` の使用）
- [ ] 共有データへの読み書きが `sync.Mutex` で保護されているか
- [ ] `wg.Add(1)` が goroutine 起動前に呼ばれ、`defer wg.Done()` が goroutine 冒頭にあるか
- [ ] goroutine がリークしないか（ctx キャンセル時に終了するか）

### セキュリティ

- [ ] APIキー・トークンがソースコードにハードコードされていないか（設定ファイル経由で注入）
- [ ] ログ出力に機密情報（アクセスキー、トークン、パスワード）が含まれていないか
- [ ] URL パラメータに含めるユーザー入力が `url.QueryEscape` でエスケープされているか
- [ ] 外部APIのレスポンスを信頼しすぎていないか（nil チェック、長さチェック）

### パフォーマンス

- [ ] サイズが既知のスライスに `make([]T, 0, n)` で capacity を事前確保しているか
- [ ] 複数の文字列結合に `strings.Builder` を使っているか（`+` 演算子のループ使用を避ける）
- [ ] `*http.Client` を構造体フィールドとして再利用しているか（リクエストごとに生成していないか）
- [ ] 不要な `json.Marshal` / `json.Unmarshal` の往復がないか

### 可読性

- [ ] 関数が単一責務を守っているか（1つの関数が検索・比較・通知を全部やっていないか）
- [ ] 変数名・関数名が処理内容を的確に表しているか
- [ ] 自明なコードに冗長なコメントを付けていないか（コメントは「なぜ」を説明する）
- [ ] exported な型・関数に GoDoc コメントがあるか（`// TypeName ...` 形式）
- [ ] マジックナンバーに意味のある名前（定数または変数）が付けられているか

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

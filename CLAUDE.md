# terraform-provider-line プロジェクト引継書 兼 起動プロンプト

新しいセッション・新しいリポジトリでこのプロジェクトを始める際に、そのままClaude Codeへの最初のメッセージとして貼り付けて使うことを想定した文書。

---

## このプロジェクトで実現したいこと

`terraform-provider-line` という、**LINE Messaging APIをTerraformで宣言的に管理できるOSSのTerraformプロバイダー**を新規開発する。

LINE公式アカウントの各種設定（Webhook URL、リッチメニュー、LIFFアプリ等）は、通常LINE Developersコンソールで手動クリックするか、個別にAPIを叩くスクリプトを書くしかない。Cloudflareには`cloudflare/cloudflare`という充実した公式Terraformプロバイダーが存在し、`terraform apply`一発でインフラ全体を宣言的に管理できるのに対し、LINEにはそれに相当するものが存在しない。この空白を埋める。

## 背景・経緯（なぜこれをやるか）

- 別プロジェクト（SNS自動化の書籍執筆）の調査過程で、Cloudflare向けの公式Terraformプロバイダーが極めて充実している（200種類以上のリソース、APIトークン発行・請求アラートまでカバー）ことが分かった一方、**LINE・Meta（Instagram）・X（Twitter）向けの専用Terraform/Pulumiプロバイダーは、公式・非公式とも実質存在しない**ことが判明した
- X向けに"twitter"を名乗るプロバイダーはいくつか存在するが、リスト/ブロック管理程度の限定機能で2020〜2022年から更新が止まっており、実用に耐えない
- 3プラットフォームを比較検討した結果、**LINEが最も着手しやすい**と判断した。理由：
  1. LINE社が公式にOpenAPI仕様（`https://github.com/line/line-openapi`）を公開しており、スキーマ定義の土台になる（Metaは非公開でこの道が閉ざされている）
  2. Webhook・リッチメニュー・LIFFアプリなど「永続する設定リソース」が多く、Terraformのリソースモデル（作成・読み取り・更新・削除）と相性が良い
  3. 市場に実質空白地帯であり、需要があれば普及する余地が大きい

## スコープ方針（重要・迷ったらここに立ち返る）

- **対象はLINEのみ**。Meta・Xは対象外（別プロジェクトとして切り離す。同時に手を広げない）
- **「設定リソース」に限定し、「一過性のアクション」は対象外にする**。Terraformのリソースモデルは「永続し、作成・読み取り・更新・削除ができるもの」に向いている。メッセージ送信・プッシュ配信のような「送ったら終わりで、更新も削除も概念として存在しない」操作は無理にTerraform化しない（Terraformプロバイダーの役割ではなく、アプリケーションコード側でMessaging API SDKを直接叩く方が適切）
- **命名は`terraform-provider-line`**。Terraform Registryの命名規約に従う。リソース名のプレフィックスは`line_`（例：`line_rich_menu`、`line_liff_app`）。「LINE Harness」という別プロジェクト（LINE公式アカウント運用の無料OSS CRMツール）とは無関係なので、"Harness"という語は使わない
- 「LINE Corporation / LINEヤフー株式会社とは非公式・無関係である」旨をREADMEに明記する（他の非公式プロバイダーの慣例に倣う）

## MVPで実装すべきリソース候補（優先順）

以下は「設定リソースであり、CRUDモデルに素直に乗る」という基準で選定した候補。実装前に各APIの実際の挙動（特に更新・削除のセマンティクス）を`line/line-openapi`で必ず確認すること。

1. **`line_webhook_endpoint`**：Webhook URLの設定（`PUT/GET /v2/bot/channel/webhook/endpoint`）。最もシンプルで最初の一歩に適する
2. **`line_liff_app`**：LIFFアプリの作成・更新・削除（`POST/GET/PUT/DELETE /liff/v1/apps`）。CRUDがきれいに揃っている
3. **`line_rich_menu`**：リッチメニューの作成・取得・削除（`POST/GET/DELETE /v2/bot/richmenu`）。**注意：画像アップロード（`POST /v2/bot/richmenu/{id}/content`）はリソース作成とは別の"content"サブリソースとして設計する必要がある。素直な単一CRUDでは表現できない点が最大の技術的難所**
4. **`line_rich_menu_default`**：全ユーザーへのデフォルトリッチメニュー設定（`POST/DELETE /v2/bot/user/all/richmenu`）。実質的にトグル的なリソース
5. **`line_rich_menu_alias`**：タブ切替式リッチメニュー用エイリアス（`POST/PUT/DELETE /v2/bot/richmenu/alias`）

MVPフェーズでは1〜3を実装できれば公開価値がある。4〜5は次フェーズでよい。

**あえて後回し・対象外にするもの**：
- チャネルアクセストークンの発行・管理（トークンのライフサイクルは「シークレット」の性質が強く、Terraform stateに載せること自体がセキュリティ上望ましくない可能性がある。要検討）
- あいさつメッセージ・応答設定（LINE公式アカウントマネージャー専用機能で、対応するAPIエンドポイントが存在しない）
- メッセージ送信・配信（前述の通りCRUDに乗らないため対象外）

## 技術的な進め方

- **HashiCorp公式のTerraform Plugin Framework**（Go）を使う。現行のTerraformプロバイダー開発の標準的な作法
- スキーマ定義の出発点として、`line/line-openapi`のOpenAPI仕様を参照する。HashiCorp公式の`terraform-plugin-codegen-openapi`（現状Tech Preview、本番非推奨と明記されている）を試してみる価値はあるが、そのまま使えることは期待せず、生成結果を叩き台として手動で整形する前提で進める
- 認証は`api_key`（LINE Channel Access Token）をプロバイダー設定（`provider "line" { channel_access_token = ... }`）で受け取る設計にする。環境変数`LINE_CHANNEL_ACCESS_TOKEN`からも読み込めるようにする（Terraformプロバイダーの一般的な作法）
- 参考にすべき既存プロバイダーの設計：`cloudflare/cloudflare`（充実したリソース設計の手本）、`Mastercard/terraform-provider-restapi`（汎用REST API連携の考え方、ただし今回は専用プロバイダーなのでこの制約からは解放される）

## 運用・保守について（最初から意識しておくこと）

このプロジェクト自体がOSSとして公開・保守されることになるため、以下は初期設計の段階から考慮する：

- **バスファクター対策**：README・CONTRIBUTING.mdを最初から整備し、単独メンテナへの依存を減らす設計を意識する（別プロジェクトの書籍執筆で「OSSメンテナのバーンアウト・バスファクター」がテーマの一つになっており、その教訓を自分ごととして適用する）
- **APIの変更への追従**：LINE Messaging APIのバージョニング・変更履歴を追う仕組み（`line/line-openapi`のリポジトリをwatchする等）を最初から用意する
- **CI/CDでのacceptance test**：実際のLINE公式アカウント（テスト用チャネル）に対して統合テストを回す仕組みを早期に用意する
- **セマンティックバージョニングとTerraform Registryへの公開**：`registry.terraform.io`への公開手順（GPG署名等）を早めに確認しておく

## 最初にやってほしいこと

1. リポジトリの初期化（Go modules、Terraform Plugin Frameworkのボイラープレート）
2. `line/line-openapi`から関連エンドポイント（Webhook、LIFF、リッチメニュー）の仕様を読み込み、Terraformスキーマ（`schema.Schema`）の設計案を提示する
3. `line_webhook_endpoint`リソースをまず1つ、Create/Read/Update/Deleteまで一通り実装し、動作確認する（最小の垂直スライスとして）
4. その後、`line_liff_app`、`line_rich_menu`（content sub-resource設計を含む）と拡張する

---

## 参考情報源

- [line/line-openapi](https://github.com/line/line-openapi) — LINE公式が公開するOpenAPI仕様
- [Messaging API reference](https://developers.line.biz/en/reference/messaging-api/)
- [LIFF Server API reference](https://developers.line.biz/en/reference/liff-server/)
- [Using rich menus](https://developers.line.biz/en/docs/messaging-api/using-rich-menus/)
- [cloudflare/cloudflare（Terraformプロバイダーの設計参考）](https://registry.terraform.io/providers/cloudflare/cloudflare/latest)
- [HashiCorp Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework)
- [hashicorp/terraform-plugin-codegen-openapi（Tech Preview）](https://github.com/hashicorp/terraform-plugin-codegen-openapi)

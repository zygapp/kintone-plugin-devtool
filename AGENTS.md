# AGENTS.md

このファイルは、Claude Code がこのリポジトリで作業する際のガイダンスと詳細な指示を提供します。

## 仕様書

詳細な仕様は [SPECIFIC.md](./SPECIFIC.md) を参照してください。

## プロジェクト概要

kpdev（kintone plugin developer）は、Tailwind v4のようなコマンド中心のDXでkintoneプラグイン開発を行うためのCLIツールです。kintoneのclassic script環境と、モダンなVite dev server + HMRワークフローを橋渡しします。

## アーキテクチャ

kintoneプラグインはclassic scriptのみ対応。開発時はESM + HMRを使いたい。この課題を以下の構造で解決：

```
kintone
  ↓ classic script
kintone-dev-loader.js（開発用プラグインとしてインストール）
  ↓ dynamic import
Vite dev server (ESM + HMR)
  ↓
src/main/main.*      # desktop/mobile 共通
src/config/main.*    # プラグイン設定画面
```

開発用ローダープラグイン（`.kpdev/managed/dev-plugin/`）はkintoneとViteを繋ぐ唯一の橋であり、手動で変更してはいけません。

## CLIコマンド

- `kpdev init` - プロジェクト初期化（対話形式で設定を入力）
- `kpdev dev` - 開発用ローダープラグインをデプロイし、Vite dev serverをHTTPS（localhost:3000）で起動
  - `--skip-deploy`: ローダープラグインのデプロイをスキップ
  - `--no-browser`: ブラウザを自動で開かない
- `kpdev build` - 本番用プラグインZIPを生成（console.error以外のconsole.*とdebuggerは自動削除）
  - ビルド前にバージョン更新を対話形式で選択可能（現状維持/パッチ/マイナー/メジャー/カスタム）
  - `--no-minify`: minify無効
  - `--remove-console`: console削除（デフォルト）
- `kpdev deploy` - 本番用プラグインZIPをkintoneにAPI経由でデプロイ
  - デプロイ先選択時に新規環境を追加可能
  - `--file`: 指定ZIPをデプロイ
  - `--all`: 全環境にデプロイ（対話スキップ）
- `kpdev config` - プロジェクト設定を対話形式で変更
  - 現在の設定を表示
  - プラグイン情報（manifest）の編集（既存値が初期値として設定される）
  - 必須パラメータの編集（`config.required_params`）
  - 開発環境の設定
  - 本番環境の管理（追加/編集/削除）
  - ターゲット（desktop/mobile）の設定

## 技術スタック

- Go CLI（cobra）
- Vite（dev server、ビルド）
- 自己署名証明書（`.kpdev/certs/`）
- プラグイン署名（RSA 1024bit、PKCS#1 SHA1）
- 対応フレームワーク: React, Vue, Svelte, Vanilla（JS/TS）

## 配布方式

Go本体 + npmラッパー（全プラットフォームのバイナリを1パッケージに同梱）:
- パッケージ名: `@zygapp/kintone-plugin-devtool`
- 対応プラットフォーム: darwin-x64, darwin-arm64, linux-x64, linux-arm64, win32-x64, win32-arm64

## プロジェクト構成

```
kintone-plugin-dev/
├── cmd/
│   └── kpdev/
│       └── main.go              # エントリーポイント
├── internal/
│   ├── cmd/                     # CLIコマンド実装
│   │   ├── root.go              # ルートコマンド
│   │   ├── init.go              # kpdev init
│   │   ├── dev.go               # kpdev dev
│   │   ├── build.go             # kpdev build
│   │   ├── deploy.go            # kpdev deploy
│   │   └── config.go            # kpdev config
│   ├── config/                  # 設定管理
│   │   ├── config.go            # config.json 読み書き
│   │   ├── env.go               # .env 読み込み
│   │   └── manifest.go          # manifest.json 順序付きJSON出力
│   ├── generator/               # 生成処理
│   │   ├── project.go           # プロジェクト雛形生成
│   │   ├── vite.go              # Vite設定生成
│   │   ├── loader.go            # ローダープラグイン生成
│   │   ├── cert.go              # 証明書生成
│   │   └── key.go               # RSA秘密鍵生成
│   ├── kintone/                 # kintone API
│   │   ├── api.go               # API共通処理
│   │   ├── plugin.go            # プラグインAPI
│   │   └── file.go              # ファイルアップロード
│   ├── plugin/                  # プラグインパッケージング
│   │   ├── packager.go          # ZIP生成
│   │   └── signer.go            # 署名処理
│   └── prompt/                  # 対話処理
│       └── prompt.go            # ユーザー入力
├── npm/                         # npmパッケージ
│   └── kpdev/
│       ├── package.json
│       ├── index.js             # npmラッパー
│       └── postinstall.js       # インストール後処理
├── build/                       # ビルド成果物
├── Makefile
├── go.mod
├── go.sum
├── AGENTS.md
└── SPECIFIC.md
```

## 生成されるプラグインプロジェクト構成

```
my-kintone-plugin/
├── src/
│   ├── main/                    # desktop/mobile 共通コード
│   │   ├── main.tsx
│   │   ├── App.tsx
│   │   └── style.css
│   └── config/                  # プラグイン設定画面
│       ├── main.tsx
│       ├── App.tsx
│       └── style.css
├── .kpdev/
│   ├── config.json              # プロジェクト設定
│   ├── manifest.json            # プラグインマニフェスト
│   ├── vite.config.ts
│   ├── certs/
│   ├── keys/
│   └── managed/
│       ├── dev-plugin/
│       ├── dev-plugin.zip
│       └── loader.meta.json
├── icon.png
├── package.json
└── .gitignore
```

## 主要ファイル

- `.kpdev/config.json` - プロジェクト設定
- `.kpdev/manifest.json` - プラグインマニフェスト（`kpdev config`で編集可能）
- `.kpdev/vite.config.ts` - Vite設定（kpdevが管理、変更禁止）
- `.kpdev/keys/private.dev.ppk` - 開発用秘密鍵（Git追跡推奨）
- `.kpdev/keys/private.prod.ppk` - 本番用秘密鍵（Git追跡推奨）
- `.kpdev/managed/dev-plugin/` - 開発用ローダープラグイン（変更禁止）
- `.kpdev/managed/loader.meta.json` - ローダーメタデータ
- `.kpdev/certs/` - HTTPS用自己署名証明書

※ 秘密鍵はプラグインIDの維持のためGitで追跡する。変更するとプラグインIDが変わる。

※ Vite設定をカスタマイズする場合は、プロジェクトルートに `vite.config.ts` を作成（そちらが優先される）

## 認証

優先順位: `.env` > `.kpdev/config.json`

認証対話スキップに必要な環境変数:
- `KPDEV_USERNAME` / `KPDEV_PASSWORD` - 開発環境
- `KPDEV_DEV_USERNAME` / `KPDEV_DEV_PASSWORD` - 開発環境（明示的）
- `KPDEV_PROD_A_USERNAME` / `KPDEV_PROD_A_PASSWORD` - 本番環境A（複数環境対応）

## 設計原則

1. ローダーは安定資産 - 自動再生成しない
2. 開発者は`src/`以下だけを意識する
3. devモードではローダープラグインのみkintoneにデプロイ（ソースコードはVite dev serverから配信）
4. deployはビルド成果物（ZIP）のみアップロード
5. 開発用と本番用で秘密鍵を分離（プラグインIDが異なる）
6. 非公式API経由で高速デプロイ

## 重要な設計判断

### 1. 秘密鍵の分離

開発用（`private.dev.ppk`）と本番用（`private.prod.ppk`）を分離する理由：
- プラグインIDが秘密鍵から導出される
- 開発用プラグインが本番環境に混入することを防ぐ
- 開発用は `[DEV]` プレフィックスで視覚的に区別

### 2. ローダープラグインの設計

- kintone（classic script）とVite（ESM）の橋渡し
- 同期XHRで Vite client をロード
- dynamic import でエントリーモジュールをロード
- 自動再生成しない（安定資産として扱う）

### 3. 複数本番環境対応

- `config.json` の `prod` を配列化
- 環境変数で認証情報を環境別に管理
- `--env` フラグで特定環境のみデプロイ可能

## 開発ガイドライン

### Go コーディング規約

1. **パッケージ構成**
   - `cmd/` - エントリーポイントのみ
   - `internal/` - 外部公開しないパッケージ
   - 機能ごとにサブパッケージを分割

2. **エラーハンドリング**
   - エラーは適切にラップして返す
   - ユーザー向けメッセージは日本語
   - デバッグ情報は英語でもよい

3. **依存パッケージ**
   - CLI: `github.com/spf13/cobra`
   - 対話: `github.com/AlecAivazis/survey/v2`
   - 色付き出力: `github.com/fatih/color`
   - .env: `github.com/joho/godotenv`

### コマンド実装パターン

各コマンドは以下のパターンで実装：

```go
var xxxCmd = &cobra.Command{
    Use:   "xxx",
    Short: "短い説明",
    Long:  "長い説明",
    RunE: func(cmd *cobra.Command, args []string) error {
        // 1. 設定読み込み
        // 2. 前提条件チェック
        // 3. メイン処理
        // 4. 結果出力
        return nil
    },
}

func init() {
    rootCmd.AddCommand(xxxCmd)
    // フラグ定義
}
```

### 出力フォーマット

```
→ 処理の開始...
○ 進行中の処理...
✓ 完了した処理
✗ 失敗した処理

Plugin ID:
  abcdefghijklmnopqrstuvwxyz

Dev server:
  https://localhost:3000
```

### プラグイン署名の実装

```go
// RSA 1024bit 鍵生成
privateKey, _ := rsa.GenerateKey(rand.Reader, 1024)

// PKCS#1 SHA1 署名
hash := sha1.Sum(contentsZip)
signature, _ := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA1, hash[:])

// プラグインID生成（公開鍵SHA256の先頭32文字を変換）
pubKeyDer, _ := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
hash256 := sha256.Sum256(pubKeyDer)
hexStr := hex.EncodeToString(hash256[:])[:32]
// 0-9a-f → a-p に変換
```

### テスト方針

1. **ユニットテスト**
   - 各パッケージに `*_test.go` を配置
   - モックを使用してAPI呼び出しをテスト

2. **統合テスト**
   - 実際のkintone環境を使用
   - `.env.test` で認証情報を管理

## kintone API 関連

### 非公式API（プラグインデプロイ）

```
POST /k/api/dev/plugin/import.json
Body: { "item": <fileKey> }
Header: X-Cybozu-Authorization: base64(username:password)
```

### ファイルアップロード

```
POST /k/v1/file.json
Content-Type: multipart/form-data
Header: X-Cybozu-Authorization: base64(username:password)
Response: { "fileKey": "xxx" }
```

## バージョン管理

バージョンは **Makefile の `VERSION` 変数で一元管理**される。

```makefile
VERSION := 0.1.0
```

この値が以下に自動反映される:
- **Go CLI** - ビルド時に `-ldflags` で注入
- **npm パッケージ** - `make build-all` 時に `package.json` を自動更新

### リリース手順

1. `Makefile` の `VERSION` を更新
2. ビルド: `make build-all`（package.json も自動更新される）
3. コミット: `chore: バージョンを v0.x.x に更新`
4. タグ追加: `git tag v0.x.x && git push origin v0.x.x`
5. npm公開: `make npm-publish-token TOKEN=xxx`
6. GitHubリリース作成: 下記「GitHubリリース作成」セクション参照

### GitHubリリース作成

`gh release create` コマンドでリリースを作成する。リリースノートには各変更のコミットIDを含める（GitHubが自動でリンクに変換する）。

```bash
gh release create v0.x.x --title "v0.x.x" --notes "$(cat <<'EOF'
## 変更内容

### 新機能
- 機能の説明 (a1b2c3d)

### 改善
- 改善の説明 (e4f5g6h)

### バグ修正
- 修正の説明 (i7j8k9l)
EOF
)"
```

#### リリースノートの書き方

1. **変更をカテゴリ分け**: 新機能、改善、バグ修正、その他
2. **各変更にコミットIDを追記**: `(abc1234)` 形式
3. **コミット一覧の取得方法**:
   ```bash
   # 前回リリースからの変更一覧を取得
   git log v0.x.x..HEAD --oneline
   ```

#### リリースノート例

```markdown
## 変更内容

### 新機能
- init コマンドに --scope オプションを追加 (a1b2c3d)
- ドメイン自動補完機能を追加 (e4f5g6h)

### 改善
- CLI出力を日本語に統一 (i7j8k9l)

### バグ修正
- package.json のキー順序を修正 (m0n1o2p)
```

### ビルドコマンド

```bash
# ローカルビルド
make build

# 全プラットフォームビルド
make build-all

# npm公開（OTP認証）
make npm-publish OTP=123456

# npm公開（トークン認証）
make npm-publish-token TOKEN=npm_xxx
```

## コミットポリシー

### コミットメッセージ形式

```
<type>: <summary>
```

### type一覧

- `feat` - 新機能
- `fix` - バグ修正
- `docs` - ドキュメントのみの変更
- `refactor` - リファクタリング（機能変更なし）
- `test` - テストの追加・修正
- `chore` - ビルド、CI、依存関係などの雑務

### ルール

- メッセージは日本語で記述
- 1行目は50文字以内を目安
- 動詞で始める（「追加」「修正」「変更」など）
- 1コミット1目的（複数の変更を混ぜない）
- 絵文字を使わない
- クレジット（Co-Authored-Byなど）を記載しない

## トラブルシューティング

### 証明書エラー

```
openssl でSAN（Subject Alternative Name）を含めること
- DNS: localhost
- IP: 127.0.0.1
- IP: ::1
```

### プラグインID不一致

秘密鍵が変わるとプラグインIDも変わる。既存インストールが無効になるため、秘密鍵は安全に管理すること。

### Vite HMR が動作しない

1. 証明書がブラウザに信頼されているか確認
2. `https://localhost:3000` にアクセスして確認
3. ローダープラグインが正しくインストールされているか確認

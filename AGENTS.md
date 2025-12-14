# AGENTS.md

このファイルは、Claude Code がこのリポジトリで作業する際の詳細な指示を提供します。

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
│   │   └── env.go               # .env 読み込み
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
├── CLAUDE.md
├── AGENTS.md
└── SPECIFIC.md
```

## 生成されるプラグインプロジェクト構成

```
my-plugin/
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

### ビルドとリリース

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

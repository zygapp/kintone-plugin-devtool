# kpdev（kintone plugin developer）仕様書 v0.3

## 1. 目的

kintoneプラグイン開発を Tailwind v4 のようなコマンド中心のDX で行うためのCLIツール。

- `kpdev init` で雛形と内部設定を生成
- `kpdev dev` で Vite dev server + HMR を使い、kintone画面にリアルタイム反映
- `kpdev build` で本番用プラグインZIPを生成
- `kpdev deploy` で kintone に API 経由デプロイ
- `kpdev config` で設定を対話形式で変更
- `kpdev migrate` で既存プロジェクトを最新仕様に更新
- `kpdev update` で依存パッケージを一括更新

## 2. 基本思想（重要）

- kintoneプラグインは classic script のみ対応
- Vite dev は ESM
- プラグインはZIPパッケージとして配布
- **開発用と本番用で秘密鍵を分離**（プラグインIDが異なる）
- **複数の本番環境へのデプロイをサポート**

よって dev 時は以下の構造を取る：

```
kintone
  ↓ classic script
kintone-dev-loader.js（開発用プラグインとしてインストール）
  ↓ dynamic import
Vite dev server (ESM + HMR)
  ↓
src/main/main.*      # desktop/mobile 用
src/config/main.*    # プラグイン設定画面用
```

- エンジニアは `src/` 以下だけを意識すればよい
- loader は触らない・安定資産
- `src/main/` は desktop と mobile で共通のコードを使用

## 3. 非目的（v0.1ではやらない）

- チーム共有用トンネル（ngrok / Cloudflare Tunnel）
- OSの証明書ストアへの信頼登録自動化
- APIトークン / Basic認証対応
- プラグイン設定画面のプレビュー機能

## 4. 対応環境

- OS：macOS / Linux / Windows
- Node.js：18 以上（Vite実行用）
- CLI配布：npm公開（Go本体 + Nodeラッパー）

## 5. CLI 配布方式

### 採用方式

Go本体 + npmラッパー（esbuildの配布モデルを採用）

### npmパッケージ構成

- `@zygapp/kintone-plugin-devtool`（全プラットフォーム対応）

対応プラットフォーム：
- darwin-x64（macOS Intel）
- darwin-arm64（macOS Apple Silicon）
- linux-x64
- linux-arm64
- win32-x64
- win32-arm64

`npm install` 時に全バイナリがインストールされ、postinstall でOS/CPUに合ったバイナリが選択される。

## 6. コマンド仕様

### 6.1 kpdev init

#### 目的

- プラグインプロジェクト初期化
- Vite設定
- 自己署名証明書生成
- 開発用ローダープラグイン生成
- 開発用・本番用秘密鍵生成

#### 対話フロー

1. ディレクトリ作成の確認（カレントに展開 or 新規ディレクトリ作成）
2. プロジェクト名（プラグイン名）
3. kintone開発環境ドメイン（例：`example.cybozu.com`）
4. フレームワーク選択：`React` | `Vue` | `Svelte` | `Vanilla`
5. 言語選択：`TypeScript` | `JavaScript`
6. 対象画面（複数選択可）：`デスクトップ` | `モバイル`
7. 認証情報（ユーザー名、パスワード）
8. パッケージマネージャー選択：`npm` | `pnpm` | `yarn` | `bun`

※ プラグインはシステム全体にインストールされるため、アプリIDは不要

#### コマンドオプション

```bash
kpdev init [flags]

Flags:
  --name-ja string          プラグイン名（日本語）
  --name-en string          プラグイン名（英語）
  --description-ja string   説明（日本語）
  --description-en string   説明（英語）
```

これらのフラグを指定すると、対話をスキップして直接値を設定できる。

#### 対話スキップ条件

既存ファイルに応じて対話をスキップし、値を自動取得する：

| 既存ファイル | スキップする対話 | 取得方法 |
|-------------|----------------|---------|
| `package.json` | プロジェクト名、パッケージマネージャー | 既存プロジェクトとして扱う |
| `.kpdev/managed/loader.meta.json` | プロジェクト名、ドメイン、フレームワーク、言語 | メタデータから取得 |
| `.kpdev/config.json` | kintoneドメイン | 設定値から取得 |
| `src/main/main.*` | フレームワーク、言語選択 | 拡張子から推測 |
| `.env`（認証情報あり） | ユーザー名、パスワード | 環境変数から取得 |

#### 認証情報の取得優先順位

1. `.env` の `KPDEV_USERNAME` / `KPDEV_PASSWORD`
2. `.kpdev/config.json` の `auth`
3. 対話で入力（password はマスク入力）

#### 生成物構成

```
./{projectName}/
├ src/
│  ├ main/
│  │  ├ main.(js|ts|tsx|vue|svelte)   # desktop/mobile 共通エントリ
│  │  ├ App.*
│  │  └ style.css
│  └ config/
│     ├ main.(js|ts|tsx|vue|svelte)   # プラグイン設定画面エントリ
│     ├ App.*
│     └ style.css
├ .kpdev/
│  ├ config.json
│  ├ manifest.json             # プラグインマニフェスト
│  ├ vite.config.ts
│  ├ certs/
│  │  ├ localhost-key.pem
│  │  └ localhost.pem
│  ├ keys/
│  │  ├ private.dev.ppk      # 開発用秘密鍵
│  │  └ private.prod.ppk     # 本番用秘密鍵
│  └ managed/
│     ├ dev-plugin/
│     │  ├ manifest.json
│     │  ├ icon.png
│     │  ├ desktop.js        # ローダー（src/main/ を読み込む）
│     │  ├ mobile.js         # ローダー（src/main/ を読み込む）
│     │  └ config.html       # ローダー（src/config/ を読み込む）
│     ├ dev-plugin.zip
│     └ loader.meta.json
├ icon.png（56x56）
├ package.json
├ .gitignore
├ eslint.config.js      # フレームワーク別ESLint設定
└ README.md
```

#### ESLint設定

init時にフレームワークに応じたESLint設定が自動生成される：
- **React**: eslint-plugin-react-hooks, eslint-plugin-react-refresh
- **Vue**: eslint-plugin-vue
- **Svelte**: eslint-plugin-svelte
- **Vanilla**: 基本設定のみ

TypeScriptプロジェクトの場合は、typescript-eslint も自動設定される。

#### Vite設定の扱い

- `.kpdev/vite.config.ts` はkpdevが管理
- ユーザーはこのファイルを編集しない
- カスタマイズが必要な場合：プロジェクトルートに `vite.config.ts` を作成すると、そちらが優先される

### 6.2 証明書生成仕様

- `openssl` を使用
- SAN を必ず含める：
  - `DNS: localhost`
  - `IP: 127.0.0.1`
  - `IP: ::1`
- 生成先：`.kpdev/certs/`
- OSの信頼登録はユーザー手動

### 6.3 プラグイン秘密鍵生成仕様

- `crypto/rsa` を使用（Goで実装）
- RSA 1024bit（kintoneプラグイン仕様）
- **開発用と本番用で分離**：
  - `.kpdev/keys/private.dev.ppk` - 開発用
  - `.kpdev/keys/private.prod.ppk` - 本番用
- 一度生成したら変更しない（プラグインIDが変わるため）
- 存在しない場合は自動生成

#### 秘密鍵分離の理由

- 開発環境と本番環境で**異なるプラグインID**になる
- 開発用プラグインが本番環境に混入することを防ぐ
- プラグイン名に `[DEV]` プレフィックスを付けて区別

### 6.4 kpdev dev

#### 目的

- 開発用ローダープラグインをkintoneにインストール
- Vite dev server 起動（https）
- HMR 有効
- ファイル監視・自動再デプロイ

#### 動作

1. 開発用ローダープラグイン（`.kpdev/managed/dev-plugin.zip`）を生成
2. **非公式API経由**でプラグインをkintoneにアップロード・インストール
3. Vite dev server を起動（`https://localhost:3000`）
4. ファイル変更を監視し、変更時にフルリロード

#### オプション

- `--skip-deploy`: ローダープラグインのデプロイをスキップ（2回目以降の起動時など）
- `--no-browser`: ブラウザを自動で開かない
- `--force`, `-f`: 確認ダイアログをスキップ（CI/CD向け）

#### 起動時の表示

```
→ 開発用プラグインをkintoneにデプロイ中...

○ プラグインをパッケージング中...
✓ プラグインをパッケージング完了
○ プラグインをアップロード中...
✓ プラグインをアップロード完了

Plugin ID:
  abcdefghijklmnopqrstuvwxyz

Dev server:
  https://localhost:3000

Entries:
  main:   /src/main/main.tsx
  config: /src/config/main.tsx

Loader:
  OK（再登録不要）
```

## 7. 非公式API経由デプロイ仕様

### エンドポイント

```
POST /k/api/dev/plugin/import.json
```

### デプロイ手順

```
1. POST /k/v1/file.json
   - プラグインZIPをアップロード
   - fileKey を取得

2. POST /k/api/dev/plugin/import.json
   - Body: { "item": fileKey }
   - Header: X-Cybozu-Authorization: base64(username:password)
```

### 認証ヘッダー

```
X-Cybozu-Authorization: Base64(username:password)
```

### 利点

- システム管理画面を開かずにデプロイ可能
- 開発サイクルの高速化
- CI/CDパイプラインでの自動デプロイ

## 8. 開発用ローダープラグイン仕様

### 役割

kintone（classic）と Vite（ESM）をつなぐ唯一の橋

### 構成

```
dev-plugin/
├ manifest.json
├ icon.png
├ desktop.js      # desktopローダー
├ mobile.js       # mobileローダー
└ config.html     # config用HTML（ローダー埋め込み）
```

### manifest.json（例）

```json
{
  "manifest_version": 1,
  "version": 1,
  "type": "APP",
  "name": {
    "ja": "[DEV] sample",
    "en": "[DEV] sample"
  },
  "description": {
    "ja": "kpdev開発用ローダープラグイン",
    "en": "kpdev development loader plugin"
  },
  "icon": "icon.png",
  "desktop": {
    "js": ["desktop.js"]
  },
  "mobile": {
    "js": ["mobile.js"]
  },
  "config": {
    "html": "config.html"
  }
}
```

### ローダーJS（例：desktop.js）

```javascript
// kpdev-loader
// schemaVersion: 1
// generatedAt: 2025-12-14T09:00:00+09:00
// origin: https://localhost:3000
// target: main

(() => {
  const origin = "https://localhost:3000";
  const t = Date.now();

  const xhr = new XMLHttpRequest();
  xhr.open("GET", origin + "/main.js?t=" + t, false);
  xhr.send();
  if (xhr.status === 200) {
    eval(xhr.responseText);
  }

  import(origin + "/@vite/client").catch(() => {});
})();
```

※ `desktop.js` と `mobile.js` は同じ内容（`/main.js` を読み込む）

### config.html（例）

```html
<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
</head>
<body>
  <div id="kpdev-config-root"></div>
  <script>
    (() => {
      const origin = "https://localhost:3000";
      const t = Date.now();

      const xhr = new XMLHttpRequest();
      xhr.open("GET", origin + "/config.js?t=" + t, false);
      xhr.send();
      if (xhr.status === 200) {
        eval(xhr.responseText);
      }

      import(origin + "/@vite/client").catch(() => {});
    })();
  </script>
</body>
</html>
```

### ルール

- `.kpdev/managed/` に配置
- kpdev は勝手に上書きしない
- 再生成は将来の明示コマンドでのみ行う

## 9. loader.meta.json 仕様

### 目的

- loader の生成条件と状態を記録
- 再登録が必要かどうかを判定

### スキーマ

```json
{
  "schemaVersion": 1,
  "kpdevVersion": "0.1.0",
  "generatedAt": "2025-12-14T09:00:00+09:00",
  "dev": {
    "origin": "https://localhost:3000"
  },
  "project": {
    "name": "sample",
    "framework": "react",
    "language": "typescript"
  },
  "targets": {
    "desktop": true,
    "mobile": false
  },
  "entries": {
    "main": "/src/main/main.tsx",
    "config": "/src/config/main.tsx"
  },
  "kintone": {
    "domain": "example.cybozu.com"
  },
  "pluginIds": {
    "dev": "abcdefghijklmnopqrstuvwxyz",
    "prod": "zyxwvutsrqponmlkjihgfedcba"
  },
  "files": {
    "loaderZipPath": ".kpdev/managed/dev-plugin.zip",
    "loaderZipSha256": "hexstring...",
    "devKeyPath": ".kpdev/keys/private.dev.ppk",
    "prodKeyPath": ".kpdev/keys/private.prod.ppk",
    "certKeyPath": ".kpdev/certs/localhost-key.pem",
    "certCertPath": ".kpdev/certs/localhost.pem"
  }
}
```

### 判定ルール

- loader ZIP の sha256 が不一致 → 再登録警告
- entries / origin が meta と不一致 → 再登録警告
- 自動再生成はしない

## 10. kpdev build

### 目的

本番用プラグインZIP生成

### 内容

1. **バージョン更新の選択**（対話形式）
   - 現在のまま
   - パッチ更新（1.0.0 → 1.0.1）
   - マイナー更新（1.0.0 → 1.1.0）
   - メジャー更新（1.0.0 → 2.0.0）
   - カスタム入力
2. 各エントリ（main, config）を Vite build（IIFE）
   - `src/main/` → `desktop.js` と `mobile.js`（同一内容）
   - `src/config/` → `config.js`
3. `.kpdev/manifest.json` を更新・コピー（`[DEV]` プレフィックスなし）
4. icon.png をコピー
5. **本番用秘密鍵（private.prod.ppk）で署名**
6. ZIP生成

### コマンド

```bash
# 対話形式でモードを選択
kpdev build

# 本番ビルド（minify + console削除）
kpdev build --mode prod

# プレビルド（minifyなし + console残す + 名前に[開発]付与）
kpdev build --mode pre
```

### ビルドオプション

- `--mode`: ビルドモード（prod|pre）。未指定時は対話で選択
  - `prod`: 本番ビルド（minify + console削除）
  - `pre`: プレビルド（minifyなし + console残す + プラグイン名に[開発]付与）
- `--skip-version`: バージョン確認をスキップ
- `--no-minify`: minify 無効（デフォルトは有効）
- `--remove-console`: console.log/info を削除（デフォルト有効）

### バージョン同期

ビルド時にバージョンを更新すると、以下のファイルが自動的に同期される：
- `.kpdev/manifest.json` の `version`
- `package.json` の `version`

### 出力先

```
dist/
├ plugin/
│  ├ contents.zip
│  ├ PUBKEY
│  └ SIGNATURE
└ plugin-prod-v{version}.zip
```

### config.html の生成

```html
<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <link rel="stylesheet" href="config.css">
</head>
<body>
  <div id="kpdev-config-root"></div>
  <script src="config.js"></script>
</body>
</html>
```

### プラグインZIP構造

```
plugin-prod-v1.0.0.zip
├ contents.zip          # プラグインコンテンツ
├ PUBKEY               # 公開鍵（PKCS8 DER形式）
└ SIGNATURE            # RSA署名
```

### contents.zip 内部構造

```
contents.zip
├ manifest.json
├ icon.png
├ js/
│  ├ desktop.js
│  ├ mobile.js
│  └ config.js
├ css/
│  ├ desktop.css
│  ├ mobile.css
│  └ config.css
└ html/
   └ config.html
```

## 11. kpdev deploy

### 目的

kintone に API 経由でプラグインをインストール

### コマンド

```bash
# 対話形式でモードとデプロイ先を選択
kpdev deploy

# 本番ビルドしてデプロイ
kpdev deploy --mode prod

# プレビルドしてデプロイ
kpdev deploy --mode pre

# 既存のZIPをデプロイ
kpdev deploy --file dist/plugin-prod-v1.0.0.zip

# 全環境にデプロイ（対話スキップ）
kpdev deploy --all

# CI/CD向け（対話スキップ、最新ZIPを全環境にデプロイ）
kpdev deploy --force
```

### オプション

- `--mode`: ビルドモード（prod|pre）。未指定時は対話で選択
- `--file`: デプロイするZIPファイルのパス（指定時はビルドをスキップ）
- `--all`: 全環境にデプロイ（対話スキップ）
- `--force`, `-f`: 確認ダイアログをスキップ（CI/CD向け）

### 対話フロー

```
→ デプロイ先を選択してください（複数選択可）:
  [ ] + 新規環境を追加
  [x] production-a (company-a.cybozu.com)
  [ ] production-b (company-b.cybozu.com)
  [ ] production-c (company-c.cybozu.com)
```

「新規環境を追加」を選択すると、環境名・ドメイン・認証情報を対話形式で入力し、設定に追加できる。

### デプロイ手順

1. `POST /k/v1/file.json`（ZIPアップロード）→ fileKey取得
2. `POST /k/api/dev/plugin/import.json`（非公式API）
   - Body: `{ "item": fileKey }`
   - Header: `X-Cybozu-Authorization: base64(username:password)`
   - レスポンス: `{ "success": true, "result": {} }`（Plugin ID/バージョンは含まれない）

※ デプロイ成功時に表示するPlugin IDとバージョンは、APIレスポンスではなくローカルファイルから取得：
- Plugin ID: `loader.meta.json` の `pluginIds.prod`
- バージョン: `manifest.json` の `version`

### 認証

- `X-Cybozu-Authorization: base64(username:password)`
- `.env` → `.kpdev/config.json` の順で取得

## 11.5 kpdev config

### 目的

プロジェクト設定を対話形式で変更

### メニュー

```
⚙ 設定メニュー

操作を選択してください:
  現在の設定を表示
  プラグイン情報 (manifest) の編集
  開発環境の設定
  本番環境の管理
  ターゲット (desktop/mobile) の設定
  フレームワークの変更
  エントリーポイントの設定
  終了
```

### 機能

1. **現在の設定を表示**
   - プラグイン情報（名前、説明、バージョン）
   - 開発環境（ドメイン、認証状態）
   - 本番環境一覧
   - ターゲット設定
   - フレームワーク/言語

2. **プラグイン情報（manifest）の編集**
   - プラグイン名（日本語/英語）
   - 説明（日本語/英語）
   - バージョン
   - ホームページURL

3. **開発環境の設定**
   - kintoneドメイン
   - 認証情報（ユーザー名/パスワード）

4. **本番環境の管理**
   - 環境を追加
   - 環境を編集
   - 環境を削除

5. **ターゲット（desktop/mobile）の設定**
   - デスクトップ有効/無効
   - モバイル有効/無効
   - 変更時は manifest.json も自動更新

6. **フレームワークの変更**
   - React / Vue / Svelte / Vanilla から選択
   - 選択するとエントリーポイントが自動的に更新される
   - 現在のフレームワークは選択肢から除外

7. **エントリーポイントの設定**
   - メインエントリ（src/main/main.*）のパスを変更
   - コンフィグエントリ（src/config/main.*）のパスを変更

## 11.6 kpdev migrate

### 目的

既存プロジェクトを最新のkpdev仕様に更新する。

### コマンド

```bash
# 対話形式でマイグレーション
kpdev migrate

# 確認なしでマイグレーション（CI/CD向け）
kpdev migrate --force
```

### 処理内容

1. **Vite設定の更新**
   - `.kpdev/vite.config.ts` を最新版に置き換え
   - Vite 7対応（handleHotUpdate → watcher.on('change')）

2. **package.json の依存関係更新**
   - vite, @vitejs/plugin-react 等を最新バージョンに更新
   - 不要な依存関係を削除

3. **manifest.json の標準化**
   - プロパティ順序を標準順序に並び替え
   - 欠落している必須プロパティを追加

4. **loader.meta.json の更新**
   - スキーマバージョンを最新に更新
   - 新しいフィールドを追加

### オプション

- `--force`, `-f`: 確認ダイアログをスキップ

## 11.7 kpdev update

### 目的

プロジェクトの依存パッケージを一括更新する。

### コマンド

```bash
kpdev update
```

### 処理内容

1. package.json の依存関係を最新バージョンに更新
2. パッケージマネージャーを使用してインストール（npm/pnpm/yarn/bun）
3. 更新されたパッケージの一覧を表示

### 更新対象

- `vite`
- `@vitejs/plugin-react` / `@vitejs/plugin-vue` / `@sveltejs/vite-plugin-svelte`
- `typescript`（TypeScriptプロジェクトの場合）
- その他フレームワーク固有の依存関係

## 12. 複数本番環境デプロイ

### 設定方法

`.kpdev/config.json` の `prod` を配列にする：

```json
{
  "kintone": {
    "dev": {
      "domain": "dev.cybozu.com",
      "auth": {
        "username": "dev-user",
        "password": "dev-pass"
      }
    },
    "prod": [
      {
        "name": "production-a",
        "domain": "company-a.cybozu.com",
        "auth": {
          "username": "admin-a",
          "password": "pass-a"
        }
      },
      {
        "name": "production-b",
        "domain": "company-b.cybozu.com",
        "auth": {
          "username": "admin-b",
          "password": "pass-b"
        }
      }
    ]
  },
  "dev": {
    "origin": "https://localhost:3000"
  }
}
```

### デプロイ実行例

```bash
kpdev deploy

# 出力例:
→ デプロイ先を選択してください（複数選択可）:
  [x] production-a (company-a.cybozu.com)
  [x] production-b (company-b.cybozu.com)

→ プラグインをデプロイ中...

○ production-a にデプロイ中...
✓ production-a にデプロイ完了
○ production-b にデプロイ中...
✓ production-b にデプロイ完了

✓ 2環境へのデプロイが完了しました
```

### オプション

- `--all`: 対話をスキップして全環境にデプロイ
- `--file <path>`: 指定したZIPファイルをデプロイ

### 環境変数での認証情報管理

`.env` で環境ごとに認証情報を設定可能：

```env
# 開発環境
KPDEV_DEV_USERNAME=dev-user
KPDEV_DEV_PASSWORD=dev-pass

# 本番環境A
KPDEV_PROD_A_USERNAME=admin-a
KPDEV_PROD_A_PASSWORD=pass-a

# 本番環境B
KPDEV_PROD_B_USERNAME=admin-b
KPDEV_PROD_B_PASSWORD=pass-b
```

## 13. manifest.json 仕様

`.kpdev/manifest.json` はプラグインの基本情報を定義する。`kpdev config` コマンドで対話形式で編集可能：

```json
{
  "version": "1.0.0",
  "manifest_version": 1,
  "type": "APP",
  "icon": "icon.png",
  "name": {
    "ja": "サンプルプラグイン",
    "en": "Sample Plugin"
  },
  "description": {
    "ja": "サンプルプラグインです",
    "en": "This is a sample plugin"
  },
  "homepage_url": {
    "ja": "https://example.com/ja",
    "en": "https://example.com/en"
  },
  "desktop": {
    "js": ["js/desktop.js"],
    "css": ["css/desktop.css"]
  },
  "mobile": {
    "js": ["js/mobile.js"],
    "css": ["css/mobile.css"]
  },
  "config": {
    "html": "html/config.html",
    "js": ["js/config.js"],
    "css": ["css/config.css"],
    "required_params": ["message"]
  }
}
```

### プロパティ順序

manifest.json のプロパティは以下の標準順序で保存される：
1. version
2. manifest_version
3. type
4. icon
5. name
6. description
7. homepage_url
8. config
9. desktop
10. mobile

### オプションフィールド

- `homepage_url`: プラグインのヘルプページURL（日本語/英語）
- `config.required_params`: 必須設定パラメータの配列

### kpdev が自動更新するフィールド

ビルド時に以下のパスを自動設定：
- `desktop.js` / `desktop.css`
- `mobile.js` / `mobile.css`
- `config.html` / `config.js` / `config.css`

ユーザーはこれらのパスを気にする必要がない。

## 14. プラグイン署名仕様

### 署名アルゴリズム

- **PKCS#1 SHA1** 署名スキーム
- RSA 1024bit 鍵

### プラグインID生成

```
1. 秘密鍵から公開鍵を導出
2. 公開鍵のSHA256ハッシュを計算
3. 先頭32文字を取得
4. 16進数(0-9a-f)をkintone形式(a-p)に変換
```

### 変換規則

```
0→a, 1→b, 2→c, 3→d, 4→e, 5→f, 6→g, 7→h,
8→i, 9→j, a→k, b→l, c→m, d→n, e→o, f→p
```

### 署名プロセス

```go
// 1. contents.zip を作成
contentsZip := createContentsZip(files)

// 2. 秘密鍵で署名
signature := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA1, sha1Hash(contentsZip))

// 3. 公開鍵をPKCS8 DER形式でエクスポート
pubKeyDer := x509.MarshalPKCS8PublicKey(publicKey)

// 4. 最終ZIPを作成
finalZip := createZip(contentsZip, pubKeyDer, signature)
```

## 15. .kpdev/config.json 仕様

```json
{
  "kintone": {
    "dev": {
      "domain": "example.cybozu.com",
      "auth": {
        "username": "xxx",
        "password": "yyy"
      }
    },
    "prod": [
      {
        "name": "production",
        "domain": "prod.cybozu.com",
        "auth": {
          "username": "admin",
          "password": "pass"
        }
      }
    ]
  },
  "dev": {
    "origin": "https://localhost:3000",
    "entry": {
      "main": "/src/main/main.tsx",
      "config": "/src/config/main.tsx"
    }
  },
  "targets": {
    "desktop": true,
    "mobile": false
  }
}
```

※ プラグインはシステム全体にインストールされるため、アプリIDや適用範囲は不要

### 優先順位

1. `.env`
2. `.kpdev/config.json`

## 16. Git管理ルール

`.gitignore` に必須：

```
.env
.kpdev/config.json
.kpdev/certs/
node_modules/
dist/
```

### 追跡対象とする理由

チーム開発を考慮し、以下は追跡対象とする：

- `.kpdev/vite.config.ts` - フレームワーク設定を共有
- `.kpdev/manifest.json` - プラグイン定義を共有
- `.kpdev/managed/` - ローダーとメタデータを共有
- `.kpdev/keys/` - **秘密鍵を共有（プラグインID維持のため必須）**

### 秘密鍵の扱い

- `.kpdev/keys/` は **Gitで追跡する**（プラグインIDを維持するため）
- 秘密鍵を変更するとプラグインIDが変わり、既存インストールが無効になる
- チームメンバー全員が同じ秘密鍵を使用することで、一貫したプラグインIDを維持

## 17. 実装技術（Go本体）

| 用途 | パッケージ |
|------|-----------|
| CLI | cobra |
| HTTP | net/http |
| JSON | encoding/json |
| プロセス | os/exec |
| hash | crypto/sha256, crypto/sha1 |
| ZIP | archive/zip |
| RSA署名 | crypto/rsa, crypto/x509 |

## 18. Vite設定仕様

### エンドポイント

- `/main.js` - メインエントリのIIFEバンドル（desktop/mobile共通）
- `/config.js` - configエントリのIIFEバンドル

### ビルド時の挙動

各エントリを個別にビルドし、CSSはインライン化：

```javascript
// main.js の先頭に自動挿入
(function(){
  var s = document.createElement('style');
  s.textContent = "...css content...";
  document.head.appendChild(s);
})();
```

ビルド成果物:
- `src/main/` → `desktop.js` と `mobile.js` に複製
- `src/config/` → `config.js`

## 19. 開発モードのエラー耐性

- ビルドエラーが発生してもウォッチャーは停止しない
- エラー内容をコンソールに表示し、監視を継続
- 次のファイル変更で再ビルドを試行

## 20. 最重要設計原則（再掲）

1. loader は触らせない
2. src 以下だけ考えさせる
3. dev はローダープラグインのみ自動デプロイ（ソースコードはdev serverから配信）
4. deploy は build 成果物（ZIP）だけ
5. **開発用と本番用で秘密鍵を分離**
6. **非公式API経由で高速デプロイ**
7. **複数の本番環境への一括デプロイをサポート**

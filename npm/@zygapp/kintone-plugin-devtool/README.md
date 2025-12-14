<div align="center">

# @zygapp/kintone-plugin-devtool

**kintone プラグイン開発を超簡単に**

[![npm version](https://img.shields.io/npm/v/@zygapp/kintone-plugin-devtool.svg)](https://www.npmjs.com/package/@zygapp/kintone-plugin-devtool)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![Go](https://img.shields.io/badge/Built_with-Go-00ADD8.svg)](https://go.dev/)
[![Node.js](https://img.shields.io/badge/Node.js-18+-green.svg)](https://nodejs.org/)

<br />

Vite + HMR で kintone プラグイン開発を快適に。<br />
コードを保存すれば、即座に kintone 画面に反映されます。

</div>

---

## Features

| | |
|---|---|
| **Hot Module Replacement** | コード変更が即座に kintone 画面に反映。ページリロード不要。 |
| **モダンフレームワーク対応** | React / Vue / Svelte / Vanilla に対応 |
| **TypeScript サポート** | 型安全な開発環境を標準提供 |
| **プラグイン署名** | RSA 署名付きプラグイン ZIP を自動生成 |
| **複数環境デプロイ** | 開発・本番環境を分離、複数の本番環境への一括デプロイ |
| **シンプルなワークフロー** | `init` → `dev` → `build` → `deploy` の 4 ステップ |
| **クロスプラットフォーム** | macOS / Linux / Windows（Intel & ARM） |

---

## Quick Start

### 1. インストール

```bash
npm install -g @zygapp/kintone-plugin-devtool
```

### 2. プロジェクト作成

```bash
kpdev init my-plugin
cd my-plugin
```

対話形式で以下を設定します：
- プラグイン名
- kintone ドメイン（例：`example.cybozu.com`）
- フレームワーク（React / Vue / Svelte / Vanilla）
- 言語（TypeScript / JavaScript）
- カスタマイズ対象（デスクトップ / モバイル）
- 認証情報

### 3. 開発開始

```bash
kpdev dev
```

ブラウザが自動で開きます。`https://localhost:3000` の SSL 証明書を許可すると、kintone にリダイレクトされます。

### 4. 本番デプロイ

```bash
kpdev build
kpdev deploy
```

---

## Commands

### `kpdev init [project-name]`

新しいプラグインプロジェクトを初期化します。

```bash
kpdev init my-plugin
```

**生成されるもの:**
- ソースコード（`src/main/`, `src/config/`）
- 開発用ローダープラグイン
- 開発用・本番用 RSA 秘密鍵
- SSL 証明書
- Vite 設定

### `kpdev dev`

開発用ローダープラグインを kintone にデプロイし、Vite dev server を起動します。

```bash
kpdev dev
```

**オプション:**

| オプション | 説明 |
|-----------|------|
| `--skip-deploy` | ローダープラグインのデプロイをスキップ（2回目以降の起動時に便利） |
| `--no-browser` | ブラウザを自動で開かない |

### `kpdev build`

本番用プラグイン ZIP を生成します。

ビルド前にバージョン更新を対話形式で選択できます（現状維持 / パッチ / マイナー / メジャー / カスタム）。

`console.error` 以外の `console.*` と `debugger` は自動的に削除されます。

```bash
kpdev build
```

**オプション:**

| オプション | 説明 |
|-----------|------|
| `--no-minify` | minify を無効化 |
| `--remove-console` | console.* を削除（デフォルト有効） |

**出力ファイル:**
- `dist/plugin-prod-v{version}.zip`

### `kpdev deploy`

本番用プラグイン ZIP を kintone にデプロイします。

複数の本番環境への一括デプロイに対応。デプロイ先選択時に新規環境を追加することもできます。

```bash
kpdev deploy
```

**オプション:**

| オプション | 説明 |
|-----------|------|
| `--file` | 指定した ZIP ファイルをデプロイ |
| `--all` | 全環境にデプロイ（対話スキップ） |

### `kpdev config`

プロジェクト設定を対話形式で変更します。

```bash
kpdev config
```

**設定可能な項目:**
- プラグイン情報（名前、説明、バージョン）
- 開発環境（ドメイン、認証情報）
- 本番環境の管理（追加 / 編集 / 削除）
- ターゲット（デスクトップ / モバイル）

---

## Project Structure

```
my-plugin/
├── src/
│   ├── main/             # desktop/mobile 共通コード
│   │   ├── main.tsx
│   │   ├── App.tsx
│   │   └── style.css
│   └── config/           # プラグイン設定画面
│       ├── main.tsx
│       ├── App.tsx
│       └── style.css
├── .kpdev/
│   ├── config.json       # プロジェクト設定
│   ├── manifest.json     # プラグインマニフェスト
│   ├── vite.config.ts    # Vite 設定（自動生成）
│   ├── certs/            # SSL 証明書
│   ├── keys/             # RSA 秘密鍵
│   │   ├── private.dev.ppk   # 開発用
│   │   └── private.prod.ppk  # 本番用
│   └── managed/          # ローダープラグイン（自動生成）
├── dist/                 # ビルド出力
├── icon.png              # プラグインアイコン（56x56）
├── package.json
├── .env                  # 認証情報
└── .gitignore
```

---

## Authentication

認証情報は以下の優先順位で取得されます：

### 1. `.env` ファイル（推奨）

```env
# 開発環境
KPDEV_USERNAME=your-username
KPDEV_PASSWORD=your-password

# 本番環境（複数対応）
KPDEV_PROD_A_USERNAME=admin-a
KPDEV_PROD_A_PASSWORD=pass-a
```

### 2. `.kpdev/config.json`

```json
{
  "kintone": {
    "dev": {
      "domain": "example.cybozu.com",
      "auth": {
        "username": "your-username",
        "password": "your-password"
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
  }
}
```

> **Note:** `.env` と `.kpdev/config.json` は `.gitignore` に追加されます。認証情報をリポジトリにコミットしないでください。

---

## SSL Certificate

開発サーバーは HTTPS で起動します。初回アクセス時に自己署名証明書の警告が表示されます。

### 証明書を信頼する方法

1. `https://localhost:3000` にアクセス
2. ブラウザの警告画面で「詳細設定」→「安全でないサイトへ進む」を選択
3. または、OS の証明書ストアに `.kpdev/certs/localhost.pem` を登録

---

## How It Works

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

kintone プラグインは classic script のみ対応ですが、kpdev は開発時に Vite の ESM + HMR を活用できるようローダープラグインを自動生成・デプロイします。

**開発者が意識するのは `src/` 以下のコードだけ。** ローダーや設定ファイルは kpdev が管理します。

### 開発用と本番用の分離

- **開発用**（`private.dev.ppk`）: ローダープラグイン用。プラグイン名に `[DEV]` プレフィックスが付きます。
- **本番用**（`private.prod.ppk`）: 本番ビルド用。

秘密鍵ごとにプラグイン ID が異なるため、開発用プラグインが本番環境に混入することを防ぎます。

### CLI について

kpdev は Go で実装されたネイティブバイナリです。npm 経由でインストールすると、OS・アーキテクチャに応じた実行ファイルが自動選択されます。高速な起動と安定した動作を実現しています。

---

## Requirements

- **Node.js** 18 以上
- **kintone** 環境（cybozu.com）

---

## Supported Platforms

| OS | Architecture |
|----|--------------|
| macOS | Intel (x64) / Apple Silicon (arm64) |
| Linux | x64 / arm64 |
| Windows | x64 / arm64 |

---

## Troubleshooting

### HMR が動作しない

- `https://localhost:3000` の SSL 証明書を許可しているか確認してください
- ブラウザの開発者ツールでコンソールエラーを確認してください

### ローダープラグインのデプロイに失敗する

- `.env` または `.kpdev/config.json` の認証情報を確認してください
- kintone のシステム管理権限があるか確認してください

### プラグイン ID が変わった

- 秘密鍵（`.kpdev/keys/`）が変更または削除された可能性があります
- 秘密鍵は一度生成したら変更しないでください
- チーム開発では `.kpdev/keys/` を Git で追跡し、全員が同じ秘密鍵を使用してください

### Windows で証明書エラーが出る

- PowerShell を管理者権限で実行し、証明書をインポートしてください
- または、ブラウザで `https://localhost:3000` にアクセスして手動で許可してください

## License

MIT
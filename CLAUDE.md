# CLAUDE.md

このファイルは、Claude Code (claude.ai/code) がこのリポジトリで作業する際のガイダンスを提供します。

## 指示書

リポジトリで作業する際の詳細指示は [AGENTS.md](./AGENTS.md) を参照してください

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
  - プラグイン情報（manifest）の編集
  - 開発環境の設定
  - 本番環境の管理（追加/編集/削除）
  - ターゲット（desktop/mobile）の設定

## 配布方式

Go本体 + npmラッパー（全プラットフォームのバイナリを1パッケージに同梱）:
- パッケージ名: `@zygapp/kintone-plugin-devtool`
- 対応プラットフォーム: darwin-x64, darwin-arm64, linux-x64, linux-arm64, win32-x64, win32-arm64

## 技術スタック

- Go CLI（cobra）
- Vite（dev server、ビルド）
- 自己署名証明書（`.kpdev/certs/`）
- プラグイン署名（RSA 1024bit、PKCS#1 SHA1）
- 対応フレームワーク: React, Vue, Svelte, Vanilla（JS/TS）

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
4. タグ追加: `git tag v0.x.x`
5. npm公開: `make npm-publish OTP=xxxxxx`

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

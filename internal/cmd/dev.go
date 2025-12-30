package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/fsnotify/fsnotify"
	"github.com/kintone/kpdev/internal/config"
	"github.com/kintone/kpdev/internal/generator"
	"github.com/kintone/kpdev/internal/kintone"
	"github.com/kintone/kpdev/internal/plugin"
	"github.com/spf13/cobra"
)

var (
	flagSkipDeploy bool
	flagNoBrowser  bool
	flagDevForce   bool
)

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "開発サーバーを起動",
	Long:  `開発用ローダープラグインをkintoneにデプロイし、Vite dev server を起動します。`,
	RunE:  runDev,
}

func init() {
	rootCmd.AddCommand(devCmd)

	devCmd.Flags().BoolVar(&flagSkipDeploy, "skip-deploy", false, "ローダープラグインのデプロイをスキップ")
	devCmd.Flags().BoolVar(&flagNoBrowser, "no-browser", false, "ブラウザを自動で開かない")
	devCmd.Flags().BoolVarP(&flagDevForce, "force", "f", false, "確認ダイアログをスキップ（CI/CD向け）")
}

func runDev(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	// 設定を読み込み
	cfg, err := config.Load(cwd)
	if err != nil {
		return fmt.Errorf("設定ファイルが見つかりません。先に kpdev init を実行してください: %w", err)
	}

	// メタデータを読み込み
	meta, err := generator.LoadLoaderMeta(cwd)
	if err != nil {
		return fmt.Errorf("loader.meta.json が見つかりません。先に kpdev init を実行してください: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	// プラグインをデプロイ
	if !flagSkipDeploy {
		fmt.Printf("%s 開発用プラグインをkintoneにデプロイ中...\n\n", cyan("→"))

		// プラグインZIPを作成
		fmt.Printf("○ プラグインをパッケージング中...")
		zipPath, err := plugin.PackageDevPlugin(cwd)
		if err != nil {
			fmt.Println()
			return fmt.Errorf("パッケージングエラー: %w", err)
		}
		fmt.Printf(" %s\n", green("✓"))

		// kintoneにアップロード
		fmt.Printf("○ プラグインをアップロード中...")

		// 認証情報を取得
		username := cfg.Kintone.Dev.Auth.Username
		password := cfg.Kintone.Dev.Auth.Password

		// .envから取得を試みる
		if envCfg, err := config.LoadEnv(cwd); err == nil && envCfg.HasAuth() {
			username = envCfg.Username
			password = envCfg.Password
		}

		if username == "" || password == "" {
			fmt.Println()
			return fmt.Errorf("認証情報が設定されていません")
		}

		client := kintone.NewClient(cfg.Kintone.Dev.Domain, username, password)

		fileKey, err := client.UploadFile(zipPath)
		if err != nil {
			fmt.Println()
			return fmt.Errorf("アップロードエラー: %w", err)
		}
		fmt.Printf(" %s\n", green("✓"))

		// プラグインをインポート
		fmt.Printf("○ プラグインをインポート中...")
		_, err = client.ImportPlugin(fileKey)
		if err != nil {
			fmt.Println()
			return fmt.Errorf("インポートエラー: %w", err)
		}
		fmt.Printf(" %s\n", green("✓"))

		fmt.Println()
	}

	// プラグイン情報を表示
	fmt.Printf("Plugin ID:\n")
	fmt.Printf("  %s\n", cyan(meta.PluginIDs.Dev))
	fmt.Println()

	fmt.Printf("Dev server:\n")
	fmt.Printf("  %s\n", cyan(meta.Dev.Origin))
	fmt.Println()

	fmt.Printf("Entries:\n")
	fmt.Printf("  main:   %s\n", meta.Entries.Main)
	fmt.Printf("  config: %s\n", meta.Entries.Config)
	fmt.Println()

	if flagSkipDeploy {
		fmt.Printf("Loader:\n")
		fmt.Printf("  %s（デプロイをスキップ）\n", yellow("SKIP"))
		fmt.Println()
	} else {
		fmt.Printf("Loader:\n")
		fmt.Printf("  %s（再登録不要）\n", green("OK"))
		fmt.Println()
	}

	// config.html監視を開始
	configHTMLPath := filepath.Join(cwd, "src", "config", "index.html")
	go watchConfigHTML(cwd, configHTMLPath, cfg)

	// Vite dev server を起動
	fmt.Printf("%s Dev server を起動中...\n", cyan("→"))

	viteConfigPath := filepath.Join(config.GetConfigDir(cwd), "vite.config.ts")

	viteCmd := exec.Command("npx", "vite", "--config", viteConfigPath)
	viteCmd.Dir = cwd
	viteCmd.Stdout = os.Stdout
	viteCmd.Stderr = os.Stderr
	viteCmd.Stdin = os.Stdin

	// シグナルハンドリング
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	if err := viteCmd.Start(); err != nil {
		return fmt.Errorf("Vite起動エラー: %w", err)
	}

	// ブラウザを開く
	if !flagNoBrowser {
		go func() {
			// Viteが起動するまで少し待つ
			// 実際にはViteの起動完了を検出するのが理想
			openBrowser(meta.Dev.Origin)
		}()
	}

	// シグナルまたはプロセス終了を待つ
	go func() {
		<-sigChan
		if viteCmd.Process != nil {
			viteCmd.Process.Signal(syscall.SIGTERM)
		}
	}()

	return viteCmd.Wait()
}

func watchConfigHTML(projectDir, configHTMLPath string, cfg *config.Config) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Printf("  ファイル監視の初期化に失敗: %v\n", err)
		return
	}
	defer watcher.Close()

	// 親ディレクトリを監視（ファイルが存在しない場合も考慮）
	configDir := filepath.Dir(configHTMLPath)
	if err := watcher.Add(configDir); err != nil {
		fmt.Printf("  %s の監視に失敗: %v\n", configDir, err)
		return
	}

	gray := color.New(color.FgHiBlack).SprintFunc()
	fmt.Printf("  %s\n", gray("config.html を監視中: "+configHTMLPath))

	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	// 処理中フラグ
	processing := false

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			// index.html の変更のみ処理
			if filepath.Base(event.Name) != "index.html" {
				continue
			}

			if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
				continue
			}

			// 処理中は無視
			if processing {
				continue
			}
			processing = true

			// 連続イベントをまとめるための短い待機
			time.Sleep(100 * time.Millisecond)

			fmt.Printf("\n%s config.html が変更されました。プラグインを再デプロイ中...\n", yellow("→"))

			// dev-plugin/config.html を更新
			devPluginDir := filepath.Join(config.GetConfigDir(projectDir), "managed", "dev-plugin")
			srcHTML, err := os.ReadFile(configHTMLPath)
			if err != nil {
				fmt.Printf("  %s HTMLの読み込みに失敗: %v\n", yellow("⚠"), err)
				processing = false
				continue
			}

			// config.html をそのままコピー
			if err := os.WriteFile(filepath.Join(devPluginDir, "config.html"), srcHTML, 0644); err != nil {
				fmt.Printf("  %s HTMLの書き込みに失敗: %v\n", yellow("⚠"), err)
				processing = false
				continue
			}

			// プラグインをパッケージング
			fmt.Printf("  ○ プラグインをパッケージング中...")
			zipPath, err := plugin.PackageDevPlugin(projectDir)
			if err != nil {
				fmt.Printf(" %s\n", yellow("✗"))
				fmt.Printf("    %v\n", err)
				processing = false
				continue
			}
			fmt.Printf(" %s\n", green("✓"))

			// kintoneにアップロード
			fmt.Printf("  ○ プラグインをアップロード中...")

			username := cfg.Kintone.Dev.Auth.Username
			password := cfg.Kintone.Dev.Auth.Password
			if envCfg, err := config.LoadEnv(projectDir); err == nil && envCfg.HasAuth() {
				username = envCfg.Username
				password = envCfg.Password
			}

			client := kintone.NewClient(cfg.Kintone.Dev.Domain, username, password)
			fileKey, err := client.UploadFile(zipPath)
			if err != nil {
				fmt.Printf(" %s\n", yellow("✗"))
				fmt.Printf("    %v\n", err)
				processing = false
				continue
			}
			fmt.Printf(" %s\n", green("✓"))

			// プラグインをインポート
			fmt.Printf("  ○ プラグインをインポート中...")
			_, err = client.ImportPlugin(fileKey)
			if err != nil {
				fmt.Printf(" %s\n", yellow("✗"))
				fmt.Printf("    %v\n", err)
				processing = false
				continue
			}
			fmt.Printf(" %s\n", green("✓"))

			// kintone 側の反映を待つ
			time.Sleep(500 * time.Millisecond)

			// HMR を発火させるためにエントリファイルを touch
			entryFile := filepath.Join(projectDir, "src", "config", "main.ts")
			if _, err := os.Stat(entryFile); os.IsNotExist(err) {
				entryFile = filepath.Join(projectDir, "src", "config", "main.tsx")
			}
			if _, err := os.Stat(entryFile); os.IsNotExist(err) {
				entryFile = filepath.Join(projectDir, "src", "config", "main.js")
			}
			if _, err := os.Stat(entryFile); os.IsNotExist(err) {
				entryFile = filepath.Join(projectDir, "src", "config", "main.jsx")
			}
			if _, err := os.Stat(entryFile); err == nil {
				now := time.Now()
				os.Chtimes(entryFile, now, now)
			}

			fmt.Printf("%s 再デプロイ完了。自動リロードします。\n", cyan("→"))

			processing = false

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			fmt.Printf("ファイル監視エラー: %v\n", err)
		}
	}
}

func openBrowser(url string) {
	var cmd *exec.Cmd

	switch {
	case fileExists("/usr/bin/xdg-open"):
		cmd = exec.Command("xdg-open", url)
	case fileExists("/usr/bin/open"):
		cmd = exec.Command("open", url)
	default:
		// Windows or other
		cmd = exec.Command("cmd", "/c", "start", url)
	}

	cmd.Start()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

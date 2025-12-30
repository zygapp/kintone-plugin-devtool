package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/fatih/color"
	"github.com/kintone/kpdev/internal/config"
	"github.com/kintone/kpdev/internal/generator"
	"github.com/kintone/kpdev/internal/kintone"
	"github.com/kintone/kpdev/internal/plugin"
	"github.com/kintone/kpdev/internal/prompt"
	"github.com/spf13/cobra"
)

var (
	flagDeployFile  string
	flagDeployAll   bool
	flagDeployForce bool
	flagDeployMode  string
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "プラグインをkintoneにデプロイ",
	Long: `本番用プラグインをkintoneにAPI経由でデプロイします。

モード:
  prod (デフォルト) - 本番用ビルド (minify + console削除)
  pre              - プレビルド (minifyなし + console残す + 名前に[開発]付与)`,
	RunE: runDeploy,
}

func init() {
	rootCmd.AddCommand(deployCmd)

	deployCmd.Flags().StringVar(&flagDeployFile, "file", "", "デプロイするZIPファイルのパス")
	deployCmd.Flags().BoolVar(&flagDeployAll, "all", false, "全環境にデプロイ（対話スキップ）")
	deployCmd.Flags().BoolVarP(&flagDeployForce, "force", "f", false, "確認ダイアログをスキップ（CI/CD向け）")
	deployCmd.Flags().StringVar(&flagDeployMode, "mode", "prod", "ビルドモード (prod|pre)")
}

func runDeploy(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	// 設定を読み込み
	cfg, err := config.Load(cwd)
	if err != nil {
		return fmt.Errorf("設定ファイルが見つかりません。先に kpdev init を実行してください: %w", err)
	}

	// デプロイするZIPファイルを決定
	var zipPath string
	if flagDeployFile != "" {
		// 指定されたファイルを使用
		zipPath = flagDeployFile
		if !filepath.IsAbs(zipPath) {
			zipPath = filepath.Join(cwd, zipPath)
		}
		if _, err := os.Stat(zipPath); err != nil {
			return fmt.Errorf("指定されたファイルが見つかりません: %s", zipPath)
		}
	} else {
		// dist内のZIPファイルを検索
		distDir := filepath.Join(cwd, "dist")
		zipFiles := findZipFiles(distDir)

		needBuild := true
		if len(zipFiles) > 0 {
			if flagDeployForce {
				// --force: 最新のZIPファイルを自動選択
				latestZip := findLatestZipFile(distDir, zipFiles)
				if latestZip != "" {
					zipPath = filepath.Join(distDir, latestZip)
					needBuild = false
				}
			} else {
				// 対話形式: ZIPファイルが存在する場合は選択を促す
				choice, err := prompt.AskZipFile(zipFiles)
				if err != nil {
					return err
				}

				if !choice.BuildNew {
					zipPath = filepath.Join(distDir, choice.FilePath)
					needBuild = false
				}
			}
		}

		if needBuild {
			// モードを決定
			deployMode := flagDeployMode
			if !cmd.Flags().Changed("mode") && !flagDeployForce {
				// --mode が指定されていない場合は対話で選択
				selectedMode, err := askDeployBuildMode()
				if err != nil {
					return err
				}
				deployMode = selectedMode
			}

			// ビルドを実行
			isPre := deployMode == "pre"
			if isPre {
				fmt.Printf("%s プレビルドを開始...\n\n", cyan("→"))
			} else {
				fmt.Printf("%s 本番ビルドを開始...\n\n", cyan("→"))
			}
			fmt.Printf("○ バンドル中...")

			opts := &plugin.BuildOptions{
				Mode:          deployMode,
				Minify:        !isPre,
				RemoveConsole: !isPre,
			}

			zipPath, err = plugin.Build(cwd, opts)
			if err != nil {
				fmt.Println()
				return err
			}
			fmt.Printf(" %s\n\n", green("✓"))
		}
	}

	// 本番環境の設定を確認
	if len(cfg.Kintone.Prod) == 0 {
		fmt.Printf("%s 本番環境が設定されていません。\n\n", cyan("→"))

		// 対話形式で本番環境を追加
		prodEnv, err := prompt.AskProdEnvironment()
		if err != nil {
			return err
		}

		// 設定に追加
		cfg.Kintone.Prod = append(cfg.Kintone.Prod, config.ProdEnvConfig{
			Name:   prodEnv.Name,
			Domain: prodEnv.Domain,
			Auth: config.AuthConfig{
				Username: prodEnv.Username,
				Password: prodEnv.Password,
			},
		})

		// 設定を保存
		if err := cfg.Save(cwd); err != nil {
			return fmt.Errorf("設定の保存に失敗しました: %w", err)
		}

		fmt.Printf("\n%s 本番環境を追加しました: %s\n\n", green("✓"), prodEnv.Name)
	}

	// デプロイ先を選択
	var selectedIndices []int
	if flagDeployAll || flagDeployForce {
		// 全環境を選択（--all または --force）
		selectedIndices = make([]int, len(cfg.Kintone.Prod))
		for i := range cfg.Kintone.Prod {
			selectedIndices[i] = i
		}
	} else {
		// 対話形式で選択
		targets := make([]prompt.DeployTarget, len(cfg.Kintone.Prod))
		for i, prod := range cfg.Kintone.Prod {
			targets[i] = prompt.DeployTarget{
				Name:   prod.Name,
				Domain: prod.Domain,
			}
		}

		choice, err := prompt.AskDeployTargets(targets)
		if err != nil {
			return err
		}

		// 新規環境追加が選択された場合
		if choice.AddNew {
			prodEnv, err := prompt.AskProdEnvironment()
			if err != nil {
				return err
			}

			// 設定に追加
			newIndex := len(cfg.Kintone.Prod)
			cfg.Kintone.Prod = append(cfg.Kintone.Prod, config.ProdEnvConfig{
				Name:   prodEnv.Name,
				Domain: prodEnv.Domain,
				Auth: config.AuthConfig{
					Username: prodEnv.Username,
					Password: prodEnv.Password,
				},
			})

			// 設定を保存
			if err := cfg.Save(cwd); err != nil {
				return fmt.Errorf("設定の保存に失敗しました: %w", err)
			}

			fmt.Printf("\n%s 本番環境を追加しました: %s\n", green("✓"), prodEnv.Name)

			// 新規環境をデプロイ対象に追加
			selectedIndices = append(choice.Indices, newIndex)
		} else {
			selectedIndices = choice.Indices
		}
	}

	if len(selectedIndices) == 0 {
		fmt.Println("デプロイ先が選択されていません")
		return nil
	}

	// メタデータを読み込み（プラグインID表示用）
	meta, err := generator.LoadLoaderMeta(cwd)
	if err == nil {
		fmt.Printf("Plugin ID:\n")
		fmt.Printf("  %s\n\n", cyan(meta.PluginIDs.Prod))
	}

	fmt.Printf("%s プラグインをデプロイ中...\n\n", cyan("→"))

	// 選択された環境にデプロイ
	successCount := 0
	failCount := 0

	for _, idx := range selectedIndices {
		prod := cfg.Kintone.Prod[idx]

		fmt.Printf("○ %s にデプロイ中...", prod.Name)

		// 認証情報を取得
		username := prod.Auth.Username
		password := prod.Auth.Password

		// .envから取得を試みる（環境変数名は KPDEV_PROD_{NAME}_USERNAME 形式）
		// TODO: 環境変数からの認証情報取得を実装

		if username == "" || password == "" {
			fmt.Printf(" %s\n", red("✗"))
			fmt.Printf("  認証情報が設定されていません\n")
			failCount++
			continue
		}

		// kintoneクライアントを作成
		client := kintone.NewClient(prod.Domain, username, password)

		// ファイルをアップロード
		fileKey, err := client.UploadFile(zipPath)
		if err != nil {
			fmt.Printf(" %s\n", red("✗"))
			fmt.Printf("  アップロードエラー: %v\n", err)
			failCount++
			continue
		}

		// プラグインをインポート
		result, err := client.ImportPlugin(fileKey)
		if err != nil {
			fmt.Printf(" %s\n", red("✗"))
			fmt.Printf("  インポートエラー: %v\n", err)
			failCount++
			continue
		}

		fmt.Printf(" %s\n", green("✓"))
		fmt.Printf("  Plugin ID: %s (v%d)\n", result.ID, result.Version)
		successCount++
	}

	fmt.Println()

	// 結果サマリー
	if failCount == 0 {
		fmt.Printf("%s %d環境へのデプロイが完了しました\n", green("✓"), successCount)
	} else {
		fmt.Printf("%s %d環境へのデプロイが完了、%d環境で失敗\n", red("!"), successCount, failCount)
	}

	return nil
}

// findZipFiles は指定ディレクトリ内のZIPファイルを検索して返す
func findZipFiles(dir string) []string {
	var files []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return files
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) == ".zip" {
			files = append(files, name)
		}
	}

	return files
}

// findLatestZipFile は最新の更新日時を持つZIPファイルを返す
func findLatestZipFile(dir string, files []string) string {
	if len(files) == 0 {
		return ""
	}

	var latestFile string
	var latestTime int64

	for _, file := range files {
		info, err := os.Stat(filepath.Join(dir, file))
		if err != nil {
			continue
		}
		modTime := info.ModTime().Unix()
		if modTime > latestTime {
			latestTime = modTime
			latestFile = file
		}
	}

	return latestFile
}

func askDeployBuildMode() (string, error) {
	options := []huh.Option[string]{
		huh.NewOption("本番ビルド (minify + console削除)", "prod"),
		huh.NewOption("プレビルド (minifyなし + console残す + 名前に[開発]付与)", "pre"),
	}

	var answer string
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("ビルドモードを選択").
				Options(options...).
				Value(&answer),
		),
	).WithTheme(huh.ThemeCatppuccin()).Run()
	if err != nil {
		return "", err
	}

	return answer, nil
}

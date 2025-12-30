package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
	"github.com/kintone/kpdev/internal/config"
	"github.com/kintone/kpdev/internal/generator"
	"github.com/kintone/kpdev/internal/prompt"
	"github.com/spf13/cobra"
)

var (
	migrateForce bool
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "プロジェクトを最新仕様に更新",
	Long:  `既存プロジェクトを最新のkpdev仕様に更新します。`,
	RunE:  runMigrate,
}

func init() {
	rootCmd.AddCommand(migrateCmd)
	migrateCmd.Flags().BoolVarP(&migrateForce, "force", "f", false, "確認ダイアログをスキップ（CI/CD向け）")
}

func runMigrate(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	// 設定を読み込み
	cfg, err := config.Load(cwd)
	if err != nil {
		return fmt.Errorf("設定ファイルが見つかりません。先に kpdev init を実行してください: %w", err)
	}

	fmt.Printf("%s プロジェクトの更新チェック\n\n", cyan("→"))

	// 更新項目を検出
	var updates []string

	// 1. config.json の packageManager フィールド
	if cfg.PackageManager == "" {
		pm := config.DetectPackageManager(cwd)
		updates = append(updates, fmt.Sprintf("config.json に packageManager フィールドを追加 (%s)", pm))
	}

	// 2. ESLint設定の確認
	eslintPath := filepath.Join(cwd, "eslint.config.js")
	if _, err := os.Stat(eslintPath); os.IsNotExist(err) {
		updates = append(updates, "eslint.config.js を生成")
	}

	// 3. vite.config.ts の更新 (Vite 7対応)
	viteConfigPath := filepath.Join(config.GetConfigDir(cwd), "vite.config.ts")
	if _, err := os.Stat(viteConfigPath); err == nil {
		// 既存の vite.config.ts を確認
		data, err := os.ReadFile(viteConfigPath)
		if err == nil {
			content := string(data)
			if containsHelper(content, "handleHotUpdate") {
				updates = append(updates, "vite.config.ts を Vite 7 対応版に更新")
			}
		}
	}

	// 4. manifest.json のプロパティ順序
	manifestPath := filepath.Join(config.GetConfigDir(cwd), "manifest.json")
	if _, err := os.Stat(manifestPath); err == nil {
		updates = append(updates, "manifest.json のプロパティ順序を標準化")
	}

	if len(updates) == 0 {
		fmt.Printf("%s プロジェクトは最新の状態です\n", green("✓"))
		return nil
	}

	// 更新項目を表示
	fmt.Printf("以下の更新が検出されました:\n\n")
	for i, update := range updates {
		fmt.Printf("  %d. %s\n", i+1, update)
	}
	fmt.Println()

	// 確認
	if !migrateForce {
		var confirm bool
		confirmPrompt := &survey.Confirm{
			Message: "更新を実行しますか?（バックアップが作成されます）",
			Default: true,
		}
		if err := survey.AskOne(confirmPrompt, &confirm); err != nil {
			return err
		}

		if !confirm {
			fmt.Println("キャンセルしました")
			return nil
		}
	}

	// バックアップを作成
	backupDir := filepath.Join(cwd, ".kpdev-backup-"+time.Now().Format("20060102-150405"))
	fmt.Printf("\n%s バックアップを作成中...\n", cyan("→"))
	if err := createBackup(cwd, backupDir); err != nil {
		return fmt.Errorf("バックアップ作成エラー: %w", err)
	}
	fmt.Printf("  %s %s\n", green("✓"), backupDir)

	fmt.Printf("\n%s 更新を実行中...\n\n", cyan("→"))

	// フレームワークと言語を検出
	framework := detectCurrentFramework(cwd)
	language := detectCurrentLanguage(cwd)

	// 1. config.json の更新
	if cfg.PackageManager == "" {
		fmt.Printf("  config.json を更新中...")
		cfg.PackageManager = config.DetectPackageManager(cwd)
		if err := cfg.Save(cwd); err != nil {
			fmt.Printf(" %s\n", yellow("✗"))
			return fmt.Errorf("config.json 更新エラー: %w", err)
		}
		fmt.Printf(" %s\n", green("✓"))
	}

	// 2. ESLint設定の生成
	if _, err := os.Stat(eslintPath); os.IsNotExist(err) {
		fmt.Printf("  eslint.config.js を生成中...")
		if err := generator.GenerateESLintConfig(cwd, framework, language); err != nil {
			fmt.Printf(" %s\n", yellow("✗"))
			return fmt.Errorf("ESLint設定生成エラー: %w", err)
		}
		fmt.Printf(" %s\n", green("✓"))
	}

	// 3. vite.config.ts の更新
	if _, err := os.Stat(viteConfigPath); err == nil {
		data, _ := os.ReadFile(viteConfigPath)
		if containsHelper(string(data), "handleHotUpdate") {
			fmt.Printf("  vite.config.ts を更新中...")
			if err := generator.GenerateViteConfig(cwd, framework, language); err != nil {
				fmt.Printf(" %s\n", yellow("✗"))
				return fmt.Errorf("Vite設定更新エラー: %w", err)
			}
			fmt.Printf(" %s\n", green("✓"))
		}
	}

	// 4. manifest.json のプロパティ順序を標準化
	if _, err := os.Stat(manifestPath); err == nil {
		fmt.Printf("  manifest.json を標準化中...")
		if err := standardizeManifest(manifestPath); err != nil {
			fmt.Printf(" %s\n", yellow("✗"))
			return fmt.Errorf("manifest.json 標準化エラー: %w", err)
		}
		fmt.Printf(" %s\n", green("✓"))
	}

	fmt.Printf("\n%s プロジェクトを更新しました\n", green("✓"))
	fmt.Printf("\n%s バックアップは以下にあります:\n", cyan("→"))
	fmt.Printf("  %s\n", backupDir)

	return nil
}

func createBackup(projectDir, backupDir string) error {
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return err
	}

	// バックアップ対象ファイル
	backupFiles := []string{
		".kpdev/config.json",
		".kpdev/manifest.json",
		".kpdev/vite.config.ts",
		"eslint.config.js",
		"package.json",
	}

	for _, file := range backupFiles {
		srcPath := filepath.Join(projectDir, file)
		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			continue
		}

		dstPath := filepath.Join(backupDir, file)
		dstDir := filepath.Dir(dstPath)
		if err := os.MkdirAll(dstDir, 0755); err != nil {
			return err
		}

		data, err := os.ReadFile(srcPath)
		if err != nil {
			return err
		}

		if err := os.WriteFile(dstPath, data, 0644); err != nil {
			return err
		}
	}

	return nil
}

func detectCurrentLanguage(projectDir string) prompt.Language {
	// TypeScript の存在を確認
	tsConfigPath := filepath.Join(projectDir, "tsconfig.json")
	if _, err := os.Stat(tsConfigPath); err == nil {
		return prompt.LanguageTypeScript
	}

	// src/main/main.ts または main.tsx の存在を確認
	mainTsPath := filepath.Join(projectDir, "src", "main", "main.ts")
	mainTsxPath := filepath.Join(projectDir, "src", "main", "main.tsx")
	if _, err := os.Stat(mainTsPath); err == nil {
		return prompt.LanguageTypeScript
	}
	if _, err := os.Stat(mainTsxPath); err == nil {
		return prompt.LanguageTypeScript
	}

	return prompt.LanguageJavaScript
}

func standardizeManifest(manifestPath string) error {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}

	var manifest map[string]interface{}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return err
	}

	// 標準順序で再構築
	ordered := make(map[string]interface{})

	// 順序: version, manifest_version, type, icon, name, description, config, desktop, mobile
	keys := []string{"version", "manifest_version", "type", "icon", "name", "description", "config", "desktop", "mobile"}
	for _, key := range keys {
		if val, ok := manifest[key]; ok {
			ordered[key] = val
		}
	}

	// その他のキーを追加
	for key, val := range manifest {
		if _, exists := ordered[key]; !exists {
			ordered[key] = val
		}
	}

	output, err := json.MarshalIndent(ordered, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(manifestPath, output, 0644)
}

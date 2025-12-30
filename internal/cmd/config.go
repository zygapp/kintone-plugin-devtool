package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
	"github.com/kintone/kpdev/internal/config"
	"github.com/kintone/kpdev/internal/generator"
	"github.com/kintone/kpdev/internal/prompt"
	"github.com/kintone/kpdev/internal/ui"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "プロジェクト設定を変更",
	Long:  `対話形式でプロジェクトの各種設定を変更します。`,
	RunE:  runConfig,
}

func init() {
	rootCmd.AddCommand(configCmd)
}

func runConfig(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	cfg, err := config.Load(cwd)
	if err != nil {
		return fmt.Errorf("設定ファイルが見つかりません。先に kpdev init を実行してください: %w", err)
	}

	cyan := color.New(color.FgCyan).SprintFunc()

	for {
		// 画面をクリア
		fmt.Print("\033[H\033[2J")

		fmt.Printf("%s 設定メニュー\n\n", cyan("⚙"))

		action, err := askConfigAction()
		if err != nil {
			return err
		}

		switch action {
		case "view":
			showCurrentConfig(cfg, cwd)
		case "manifest":
			if err := editManifest(cwd); err != nil {
				return err
			}
		case "dev":
			if err := editDevConfig(cfg); err != nil {
				return err
			}
			if err := cfg.Save(cwd); err != nil {
				return err
			}
		case "prod":
			if err := manageProdConfig(cfg); err != nil {
				return err
			}
			if err := cfg.Save(cwd); err != nil {
				return err
			}
		case "targets":
			if err := editTargets(cfg); err != nil {
				return err
			}
			if err := cfg.Save(cwd); err != nil {
				return err
			}
		case "framework":
			if err := switchFramework(cwd, cfg); err != nil {
				return err
			}
			if err := cfg.Save(cwd); err != nil {
				return err
			}
		case "entry":
			if err := editEntryPoints(cwd, cfg); err != nil {
				return err
			}
			if err := cfg.Save(cwd); err != nil {
				return err
			}
		case "exit":
			fmt.Println("\n設定を終了します。")
			return nil
		}
	}
}

func askConfigAction() (string, error) {
	options := []string{
		"現在の設定を表示",
		"プラグイン情報 (manifest) の編集",
		"開発環境の設定",
		"本番環境の管理",
		"ターゲット (desktop/mobile) の設定",
		"フレームワークの切り替え",
		"エントリーポイントの設定",
		"終了",
	}

	var answer string
	prompt := &survey.Select{
		Message: "操作を選択してください:",
		Options: options,
	}
	if err := survey.AskOne(prompt, &answer); err != nil {
		return "", err
	}

	switch answer {
	case options[0]:
		return "view", nil
	case options[1]:
		return "manifest", nil
	case options[2]:
		return "dev", nil
	case options[3]:
		return "prod", nil
	case options[4]:
		return "targets", nil
	case options[5]:
		return "framework", nil
	case options[6]:
		return "entry", nil
	default:
		return "exit", nil
	}
}

func showCurrentConfig(cfg *config.Config, projectDir string) {
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	fmt.Printf("\n%s 現在の設定\n\n", cyan("📋"))

	// マニフェスト情報
	fmt.Printf("%s\n", cyan("プラグイン情報:"))
	manifest, err := loadManifest(projectDir)
	if err != nil {
		fmt.Printf("  %s\n", yellow("読み込みエラー"))
	} else {
		if name, ok := manifest["name"].(map[string]interface{}); ok {
			fmt.Printf("  名前: %v / %v\n", name["ja"], name["en"])
		}
		if desc, ok := manifest["description"].(map[string]interface{}); ok {
			fmt.Printf("  説明: %v\n", desc["ja"])
		}
		fmt.Printf("  バージョン: %v\n", manifest["version"])
	}

	// 開発環境
	fmt.Printf("\n%s\n", cyan("開発環境:"))
	fmt.Printf("  ドメイン: %s\n", cfg.Kintone.Dev.Domain)
	if cfg.Kintone.Dev.Auth.Username != "" {
		fmt.Printf("  ユーザー: %s\n", cfg.Kintone.Dev.Auth.Username)
		fmt.Printf("  パスワード: %s\n", "********")
	} else {
		fmt.Printf("  認証: %s\n", yellow("未設定"))
	}

	// 本番環境
	fmt.Printf("\n%s\n", cyan("本番環境:"))
	if len(cfg.Kintone.Prod) == 0 {
		fmt.Printf("  %s\n", yellow("未設定"))
	} else {
		for i, prod := range cfg.Kintone.Prod {
			fmt.Printf("  [%d] %s (%s)\n", i+1, prod.Name, prod.Domain)
			if prod.Auth.Username != "" {
				fmt.Printf("      ユーザー: %s\n", prod.Auth.Username)
			}
		}
	}

	// ターゲット
	fmt.Printf("\n%s\n", cyan("ターゲット:"))
	if cfg.Targets.Desktop {
		fmt.Printf("  %s デスクトップ\n", green("✓"))
	} else {
		fmt.Printf("  ✗ デスクトップ\n")
	}
	if cfg.Targets.Mobile {
		fmt.Printf("  %s モバイル\n", green("✓"))
	} else {
		fmt.Printf("  ✗ モバイル\n")
	}

	fmt.Println()
}

func loadManifest(projectDir string) (map[string]interface{}, error) {
	manifestPath := filepath.Join(config.GetConfigDir(projectDir), "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}

	var manifest map[string]interface{}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}

	return manifest, nil
}

func saveManifest(projectDir string, manifest map[string]interface{}) error {
	manifestPath := filepath.Join(config.GetConfigDir(projectDir), "manifest.json")
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(manifestPath, data, 0644)
}

func editManifest(projectDir string) error {
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	fmt.Printf("\n%s プラグイン情報の編集\n\n", cyan("🔧"))

	manifest, err := loadManifest(projectDir)
	if err != nil {
		return fmt.Errorf("manifest.json の読み込みに失敗しました: %w", err)
	}

	// 名前 (日本語)
	name := manifest["name"].(map[string]interface{})
	var nameJa string
	nameJaPrompt := &survey.Input{
		Message: "プラグイン名 (日本語):",
		Default: fmt.Sprintf("%v", name["ja"]),
	}
	if err := survey.AskOne(nameJaPrompt, &nameJa, survey.WithValidator(survey.Required)); err != nil {
		return err
	}
	name["ja"] = nameJa

	// 名前 (英語)
	var nameEn string
	nameEnPrompt := &survey.Input{
		Message: "プラグイン名 (English):",
		Default: fmt.Sprintf("%v", name["en"]),
	}
	if err := survey.AskOne(nameEnPrompt, &nameEn, survey.WithValidator(survey.Required)); err != nil {
		return err
	}
	name["en"] = nameEn

	// 説明 (日本語)
	desc := manifest["description"].(map[string]interface{})
	var descJa string
	descJaPrompt := &survey.Input{
		Message: "説明 (日本語):",
		Default: fmt.Sprintf("%v", desc["ja"]),
	}
	if err := survey.AskOne(descJaPrompt, &descJa); err != nil {
		return err
	}
	desc["ja"] = descJa

	// 説明 (英語)
	var descEn string
	descEnPrompt := &survey.Input{
		Message: "説明 (English):",
		Default: fmt.Sprintf("%v", desc["en"]),
	}
	if err := survey.AskOne(descEnPrompt, &descEn); err != nil {
		return err
	}
	desc["en"] = descEn

	// バージョン
	var version string
	versionPrompt := &survey.Input{
		Message: "バージョン:",
		Default: fmt.Sprintf("%v", manifest["version"]),
	}
	if err := survey.AskOne(versionPrompt, &version, survey.WithValidator(survey.Required)); err != nil {
		return err
	}
	manifest["version"] = version

	// 保存
	if err := saveManifest(projectDir, manifest); err != nil {
		return err
	}

	fmt.Printf("\n%s プラグイン情報を更新しました\n", green("✓"))
	return nil
}

func editDevConfig(cfg *config.Config) error {
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	fmt.Printf("\n%s 開発環境の設定\n\n", cyan("🔧"))

	// ドメイン
	domain, err := prompt.AskDomain(cfg.Kintone.Dev.Domain)
	if err != nil {
		return err
	}
	cfg.Kintone.Dev.Domain = domain

	// 認証情報を更新するか確認
	var updateAuth bool
	authPrompt := &survey.Confirm{
		Message: "認証情報を更新しますか?",
		Default: false,
	}
	if err := survey.AskOne(authPrompt, &updateAuth); err != nil {
		return err
	}

	if updateAuth {
		username, err := prompt.AskUsername()
		if err != nil {
			return err
		}
		password, err := prompt.AskPassword()
		if err != nil {
			return err
		}
		cfg.Kintone.Dev.Auth.Username = username
		cfg.Kintone.Dev.Auth.Password = password
	}

	fmt.Printf("\n%s 開発環境の設定を更新しました\n", green("✓"))
	return nil
}

func manageProdConfig(cfg *config.Config) error {
	cyan := color.New(color.FgCyan).SprintFunc()

	fmt.Printf("\n%s 本番環境の管理\n\n", cyan("🔧"))

	options := []string{
		"環境を追加",
		"環境を編集",
		"環境を削除",
		"戻る",
	}

	var answer string
	prompt := &survey.Select{
		Message: "操作を選択してください:",
		Options: options,
	}
	if err := survey.AskOne(prompt, &answer); err != nil {
		return err
	}

	switch answer {
	case options[0]:
		return addProdEnv(cfg)
	case options[1]:
		return editProdEnv(cfg)
	case options[2]:
		return deleteProdEnv(cfg)
	}

	return nil
}

func addProdEnv(cfg *config.Config) error {
	green := color.New(color.FgGreen).SprintFunc()

	prodEnv, err := prompt.AskProdEnvironment()
	if err != nil {
		return err
	}

	cfg.Kintone.Prod = append(cfg.Kintone.Prod, config.ProdEnvConfig{
		Name:   prodEnv.Name,
		Domain: prodEnv.Domain,
		Auth: config.AuthConfig{
			Username: prodEnv.Username,
			Password: prodEnv.Password,
		},
	})

	fmt.Printf("\n%s 本番環境を追加しました: %s\n", green("✓"), prodEnv.Name)
	return nil
}

func editProdEnv(cfg *config.Config) error {
	if len(cfg.Kintone.Prod) == 0 {
		fmt.Println("本番環境が設定されていません")
		return nil
	}

	green := color.New(color.FgGreen).SprintFunc()

	// 環境を選択
	options := make([]string, len(cfg.Kintone.Prod))
	for i, prod := range cfg.Kintone.Prod {
		options[i] = prod.Name + " (" + prod.Domain + ")"
	}

	var selected string
	selectPrompt := &survey.Select{
		Message: "編集する環境を選択:",
		Options: options,
	}
	if err := survey.AskOne(selectPrompt, &selected); err != nil {
		return err
	}

	// インデックスを特定
	var idx int
	for i, opt := range options {
		if opt == selected {
			idx = i
			break
		}
	}

	prod := &cfg.Kintone.Prod[idx]

	// 名前
	var name string
	namePrompt := &survey.Input{
		Message: "環境名:",
		Default: prod.Name,
	}
	if err := survey.AskOne(namePrompt, &name, survey.WithValidator(survey.Required)); err != nil {
		return err
	}
	prod.Name = name

	// ドメイン
	domain, err := prompt.AskDomain(prod.Domain)
	if err != nil {
		return err
	}
	prod.Domain = domain

	// 認証情報を更新するか確認
	var updateAuth bool
	authPrompt := &survey.Confirm{
		Message: "認証情報を更新しますか?",
		Default: false,
	}
	if err := survey.AskOne(authPrompt, &updateAuth); err != nil {
		return err
	}

	if updateAuth {
		username, err := prompt.AskUsername()
		if err != nil {
			return err
		}
		password, err := prompt.AskPassword()
		if err != nil {
			return err
		}
		prod.Auth.Username = username
		prod.Auth.Password = password
	}

	fmt.Printf("\n%s 本番環境を更新しました: %s\n", green("✓"), prod.Name)
	return nil
}

func deleteProdEnv(cfg *config.Config) error {
	if len(cfg.Kintone.Prod) == 0 {
		fmt.Println("本番環境が設定されていません")
		return nil
	}

	red := color.New(color.FgRed).SprintFunc()

	// 環境を選択
	options := make([]string, len(cfg.Kintone.Prod))
	for i, prod := range cfg.Kintone.Prod {
		options[i] = prod.Name + " (" + prod.Domain + ")"
	}

	var selected string
	selectPrompt := &survey.Select{
		Message: "削除する環境を選択:",
		Options: options,
	}
	if err := survey.AskOne(selectPrompt, &selected); err != nil {
		return err
	}

	// インデックスを特定
	var idx int
	for i, opt := range options {
		if opt == selected {
			idx = i
			break
		}
	}

	// 確認
	var confirm bool
	confirmPrompt := &survey.Confirm{
		Message: fmt.Sprintf("本当に「%s」を削除しますか?", cfg.Kintone.Prod[idx].Name),
		Default: false,
	}
	if err := survey.AskOne(confirmPrompt, &confirm); err != nil {
		return err
	}

	if !confirm {
		fmt.Println("削除をキャンセルしました")
		return nil
	}

	name := cfg.Kintone.Prod[idx].Name
	cfg.Kintone.Prod = append(cfg.Kintone.Prod[:idx], cfg.Kintone.Prod[idx+1:]...)

	fmt.Printf("\n%s 本番環境を削除しました: %s\n", red("✗"), name)
	return nil
}

func editTargets(cfg *config.Config) error {
	green := color.New(color.FgGreen).SprintFunc()

	fmt.Println()

	desktop, mobile, err := prompt.AskTargets(cfg.Targets.Desktop, cfg.Targets.Mobile)
	if err != nil {
		return err
	}

	cfg.Targets.Desktop = desktop
	cfg.Targets.Mobile = mobile

	fmt.Printf("\n%s ターゲットを更新しました\n", green("✓"))
	return nil
}

func switchFramework(projectDir string, cfg *config.Config) error {
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	fmt.Printf("\n%s フレームワークの切り替え\n\n", cyan("🔧"))

	// 現在のフレームワークを検出
	currentFramework := detectCurrentFramework(projectDir)
	fmt.Printf("現在のフレームワーク: %s\n\n", cyan(string(currentFramework)))

	// 新しいフレームワークを選択
	newFramework, err := prompt.AskFramework()
	if err != nil {
		return err
	}

	if newFramework == currentFramework {
		fmt.Printf("\n%s フレームワークは変更されていません\n", cyan("→"))
		return nil
	}

	// 言語を選択
	newLanguage, err := prompt.AskLanguage()
	if err != nil {
		return err
	}

	// パッケージマネージャーを取得
	pm := cfg.GetPackageManager(projectDir)

	// 確認
	var confirm bool
	confirmPrompt := &survey.Confirm{
		Message: fmt.Sprintf("%s から %s に切り替えますか? (パッケージの再インストールが必要です)", currentFramework, newFramework),
		Default: true,
	}
	if err := survey.AskOne(confirmPrompt, &confirm); err != nil {
		return err
	}

	if !confirm {
		fmt.Println("キャンセルしました")
		return nil
	}

	fmt.Printf("\n%s フレームワークを切り替え中...\n", cyan("→"))

	// 古いフレームワークのパッケージをアンインストール
	oldPkgs := getFrameworkPackages(currentFramework)
	if len(oldPkgs) > 0 {
		var uninstallArgs []string
		switch pm {
		case "npm":
			uninstallArgs = append([]string{"uninstall"}, oldPkgs...)
		case "pnpm":
			uninstallArgs = append([]string{"remove"}, oldPkgs...)
		case "yarn":
			uninstallArgs = append([]string{"remove"}, oldPkgs...)
		case "bun":
			uninstallArgs = append([]string{"remove"}, oldPkgs...)
		}
		// エラーは無視（パッケージが存在しない場合もある）
		ui.RunCommandWithSpinner("古いパッケージを削除中...", pm, uninstallArgs, projectDir)
	}

	// 新しいフレームワークのパッケージをインストール
	newPkgs := getFrameworkPackages(newFramework)
	if len(newPkgs) > 0 {
		var installArgs []string
		switch pm {
		case "npm":
			installArgs = append([]string{"install", "-D"}, newPkgs...)
		case "pnpm":
			installArgs = append([]string{"add", "-D"}, newPkgs...)
		case "yarn":
			installArgs = append([]string{"add", "-D"}, newPkgs...)
		case "bun":
			installArgs = append([]string{"add", "-d"}, newPkgs...)
		}
		if err := ui.RunCommandWithSpinner("新しいパッケージをインストール中...", pm, installArgs, projectDir); err != nil {
			return fmt.Errorf("パッケージインストールエラー: %w", err)
		}
	}

	// vite.config.ts を再生成
	fmt.Printf("  Vite設定を再生成中...")
	if err := generator.GenerateViteConfig(projectDir, newFramework, newLanguage); err != nil {
		return fmt.Errorf("Vite設定生成エラー: %w", err)
	}
	fmt.Printf(" %s\n", green("✓"))

	// eslint.config.js を再生成（既存ファイルを削除してから）
	fmt.Printf("  ESLint設定を再生成中...")
	eslintPath := filepath.Join(projectDir, "eslint.config.js")
	os.Remove(eslintPath)
	if err := generator.GenerateESLintConfig(projectDir, newFramework, newLanguage); err != nil {
		return fmt.Errorf("ESLint設定生成エラー: %w", err)
	}
	fmt.Printf(" %s\n", green("✓"))

	// config.json のエントリーパスを更新
	cfg.Dev.Entry.Main = generator.GetEntryPath(newFramework, newLanguage, "main")
	cfg.Dev.Entry.Config = generator.GetEntryPath(newFramework, newLanguage, "config")

	fmt.Printf("\n%s フレームワークを %s に切り替えました\n", green("✓"), newFramework)
	fmt.Printf("\n%s ソースファイルは手動で更新してください\n", cyan("→"))

	return nil
}

func detectCurrentFramework(projectDir string) prompt.Framework {
	pkgPath := filepath.Join(projectDir, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return prompt.FrameworkVanilla
	}

	content := string(data)
	if contains(content, `"react"`) {
		return prompt.FrameworkReact
	}
	if contains(content, `"vue"`) {
		return prompt.FrameworkVue
	}
	if contains(content, `"svelte"`) {
		return prompt.FrameworkSvelte
	}
	return prompt.FrameworkVanilla
}

func getFrameworkPackages(framework prompt.Framework) []string {
	switch framework {
	case prompt.FrameworkReact:
		return []string{"react", "react-dom", "@vitejs/plugin-react", "@types/react", "@types/react-dom"}
	case prompt.FrameworkVue:
		return []string{"vue", "@vitejs/plugin-vue"}
	case prompt.FrameworkSvelte:
		return []string{"svelte", "@sveltejs/vite-plugin-svelte"}
	default:
		return nil
	}
}

func editEntryPoints(projectDir string, cfg *config.Config) error {
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	fmt.Printf("\n%s エントリーポイントの設定\n\n", cyan("🔧"))

	fmt.Printf("現在のエントリーポイント:\n")
	fmt.Printf("  main:   %s\n", cyan(cfg.Dev.Entry.Main))
	fmt.Printf("  config: %s\n\n", cyan(cfg.Dev.Entry.Config))

	// mainエントリーポイント
	var mainEntry string
	mainPrompt := &survey.Input{
		Message: "main エントリーポイント:",
		Default: cfg.Dev.Entry.Main,
	}
	if err := survey.AskOne(mainPrompt, &mainEntry, survey.WithValidator(survey.Required)); err != nil {
		return err
	}

	// configエントリーポイント
	var configEntry string
	configPrompt := &survey.Input{
		Message: "config エントリーポイント:",
		Default: cfg.Dev.Entry.Config,
	}
	if err := survey.AskOne(configPrompt, &configEntry, survey.WithValidator(survey.Required)); err != nil {
		return err
	}

	cfg.Dev.Entry.Main = mainEntry
	cfg.Dev.Entry.Config = configEntry

	fmt.Printf("\n%s エントリーポイントを更新しました\n", green("✓"))
	return nil
}

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
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

	for {
		// 画面をクリア
		fmt.Print("\033[H\033[2J")

		fmt.Printf("%s 設定メニュー\n\n", ui.InfoStyle.Render("⚙"))

		action, err := askConfigAction()
		if err != nil {
			if errors.Is(err, huh.ErrUserAborted) {
				return nil
			}
			return err
		}

		switch action {
		case "view":
			showCurrentConfig(cfg, cwd)
		case "manifest":
			if err := editManifest(cwd); err != nil {
				if errors.Is(err, huh.ErrUserAborted) {
					continue
				}
				return err
			}
		case "dev":
			if err := editDevConfig(cfg); err != nil {
				if errors.Is(err, huh.ErrUserAborted) {
					continue
				}
				return err
			}
			if err := cfg.Save(cwd); err != nil {
				return err
			}
		case "prod":
			if err := manageProdConfig(cfg); err != nil {
				if errors.Is(err, huh.ErrUserAborted) {
					continue
				}
				return err
			}
			if err := cfg.Save(cwd); err != nil {
				return err
			}
		case "targets":
			if err := editTargets(cfg); err != nil {
				if errors.Is(err, huh.ErrUserAborted) {
					continue
				}
				return err
			}
			if err := cfg.Save(cwd); err != nil {
				return err
			}
		case "framework":
			if err := switchFramework(cwd, cfg); err != nil {
				if errors.Is(err, huh.ErrUserAborted) {
					continue
				}
				return err
			}
			if err := cfg.Save(cwd); err != nil {
				return err
			}
		case "entry":
			if err := editEntryPoints(cwd, cfg); err != nil {
				if errors.Is(err, huh.ErrUserAborted) {
					continue
				}
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
	type actionChoice struct {
		label  string
		action string
	}

	choices := []actionChoice{
		{"現在の設定を表示", "view"},
		{"プラグイン情報 (manifest) の編集", "manifest"},
		{"開発環境の設定", "dev"},
		{"本番環境の管理", "prod"},
		{"ターゲット (desktop/mobile) の設定", "targets"},
		{"フレームワークの切り替え", "framework"},
		{"エントリーポイントの設定", "entry"},
		{"終了", "exit"},
	}

	options := make([]huh.Option[string], len(choices))
	for i, c := range choices {
		options[i] = huh.NewOption(c.label, c.action)
	}

	var answer string
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("操作を選択してください").
				Options(options...).
				Value(&answer),
		),
	).WithTheme(huh.ThemeCatppuccin()).Run()
	if err != nil {
		return "", err
	}

	return answer, nil
}

func showCurrentConfig(cfg *config.Config, projectDir string) {
	fmt.Printf("\n%s 現在の設定\n\n", ui.InfoStyle.Render("📋"))

	// マニフェスト情報
	fmt.Printf("%s\n", ui.InfoStyle.Render("プラグイン情報:"))
	manifest, err := loadManifest(projectDir)
	if err != nil {
		fmt.Printf("  %s\n", ui.WarnStyle.Render("読み込みエラー"))
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
	fmt.Printf("\n%s\n", ui.InfoStyle.Render("開発環境:"))
	fmt.Printf("  ドメイン: %s\n", cfg.Kintone.Dev.Domain)
	if cfg.Kintone.Dev.Auth.Username != "" {
		fmt.Printf("  ユーザー: %s\n", cfg.Kintone.Dev.Auth.Username)
		fmt.Printf("  パスワード: %s\n", "********")
	} else {
		fmt.Printf("  認証: %s\n", ui.WarnStyle.Render("未設定"))
	}

	// 本番環境
	fmt.Printf("\n%s\n", ui.InfoStyle.Render("本番環境:"))
	if len(cfg.Kintone.Prod) == 0 {
		fmt.Printf("  %s\n", ui.WarnStyle.Render("未設定"))
	} else {
		for i, prod := range cfg.Kintone.Prod {
			fmt.Printf("  [%d] %s (%s)\n", i+1, prod.Name, prod.Domain)
			if prod.Auth.Username != "" {
				fmt.Printf("      ユーザー: %s\n", prod.Auth.Username)
			}
		}
	}

	// ターゲット
	fmt.Printf("\n%s\n", ui.InfoStyle.Render("ターゲット:"))
	if cfg.Targets.Desktop {
		fmt.Printf("  %s デスクトップ\n", ui.SuccessStyle.Render(ui.IconSuccess))
	} else {
		fmt.Printf("  ✗ デスクトップ\n")
	}
	if cfg.Targets.Mobile {
		fmt.Printf("  %s モバイル\n", ui.SuccessStyle.Render(ui.IconSuccess))
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
	fmt.Printf("\n%s プラグイン情報の編集\n\n", ui.InfoStyle.Render("🔧"))

	manifest, err := loadManifest(projectDir)
	if err != nil {
		return fmt.Errorf("manifest.json の読み込みに失敗しました: %w", err)
	}

	// 名前 (日本語)
	name := manifest["name"].(map[string]interface{})
	nameJa, err := askInput("プラグイン名 (日本語)", fmt.Sprintf("%v", name["ja"]), true)
	if err != nil {
		return err
	}
	name["ja"] = nameJa

	// 名前 (英語)
	nameEn, err := askInput("プラグイン名 (English)", fmt.Sprintf("%v", name["en"]), true)
	if err != nil {
		return err
	}
	name["en"] = nameEn

	// 説明 (日本語)
	desc := manifest["description"].(map[string]interface{})
	descJa, err := askInput("説明 (日本語)", fmt.Sprintf("%v", desc["ja"]), false)
	if err != nil {
		return err
	}
	desc["ja"] = descJa

	// 説明 (英語)
	descEn, err := askInput("説明 (English)", fmt.Sprintf("%v", desc["en"]), false)
	if err != nil {
		return err
	}
	desc["en"] = descEn

	// バージョン
	version, err := askInput("バージョン", fmt.Sprintf("%v", manifest["version"]), true)
	if err != nil {
		return err
	}
	manifest["version"] = version

	// 保存
	if err := saveManifest(projectDir, manifest); err != nil {
		return err
	}

	ui.Success("プラグイン情報を更新しました")
	return nil
}

func askInput(title, defaultVal string, required bool) (string, error) {
	var answer string
	input := huh.NewInput().
		Title(title).
		Value(&answer).
		Placeholder(defaultVal)

	if required {
		input = input.Validate(func(s string) error {
			if s == "" {
				return fmt.Errorf("入力必須です")
			}
			return nil
		})
	}

	err := huh.NewForm(
		huh.NewGroup(input),
	).WithTheme(huh.ThemeCatppuccin()).Run()
	if err != nil {
		return "", err
	}
	if answer == "" {
		answer = defaultVal
	}
	return answer, nil
}

func editDevConfig(cfg *config.Config) error {
	fmt.Printf("\n%s 開発環境の設定\n\n", ui.InfoStyle.Render("🔧"))

	// ドメイン
	domain, err := prompt.AskDomain(cfg.Kintone.Dev.Domain)
	if err != nil {
		return err
	}
	cfg.Kintone.Dev.Domain = domain

	// 認証情報を更新するか確認
	updateAuth, err := prompt.AskConfirm("認証情報を更新しますか?", false)
	if err != nil {
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

	ui.Success("開発環境の設定を更新しました")
	return nil
}

func manageProdConfig(cfg *config.Config) error {
	fmt.Printf("\n%s 本番環境の管理\n\n", ui.InfoStyle.Render("🔧"))

	type actionChoice struct {
		label  string
		action string
	}

	choices := []actionChoice{
		{"環境を追加", "add"},
		{"環境を編集", "edit"},
		{"環境を削除", "delete"},
		{"戻る", "back"},
	}

	options := make([]huh.Option[string], len(choices))
	for i, c := range choices {
		options[i] = huh.NewOption(c.label, c.action)
	}

	var answer string
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("操作を選択してください").
				Options(options...).
				Value(&answer),
		),
	).WithTheme(huh.ThemeCatppuccin()).Run()
	if err != nil {
		return err
	}

	switch answer {
	case "add":
		return addProdEnv(cfg)
	case "edit":
		return editProdEnv(cfg)
	case "delete":
		return deleteProdEnv(cfg)
	}

	return nil
}

func addProdEnv(cfg *config.Config) error {
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

	ui.Success(fmt.Sprintf("本番環境を追加しました: %s", prodEnv.Name))
	return nil
}

func editProdEnv(cfg *config.Config) error {
	if len(cfg.Kintone.Prod) == 0 {
		fmt.Println("本番環境が設定されていません")
		return nil
	}

	// 環境を選択
	options := make([]huh.Option[int], len(cfg.Kintone.Prod))
	for i, prod := range cfg.Kintone.Prod {
		options[i] = huh.NewOption(prod.Name+" ("+prod.Domain+")", i)
	}

	var idx int
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[int]().
				Title("編集する環境を選択").
				Options(options...).
				Value(&idx),
		),
	).WithTheme(huh.ThemeCatppuccin()).Run()
	if err != nil {
		return err
	}

	prod := &cfg.Kintone.Prod[idx]

	// 名前
	name, err := askInput("環境名", prod.Name, true)
	if err != nil {
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
	updateAuth, err := prompt.AskConfirm("認証情報を更新しますか?", false)
	if err != nil {
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

	ui.Success(fmt.Sprintf("本番環境を更新しました: %s", prod.Name))
	return nil
}

func deleteProdEnv(cfg *config.Config) error {
	if len(cfg.Kintone.Prod) == 0 {
		fmt.Println("本番環境が設定されていません")
		return nil
	}

	// 環境を選択
	options := make([]huh.Option[int], len(cfg.Kintone.Prod))
	for i, prod := range cfg.Kintone.Prod {
		options[i] = huh.NewOption(prod.Name+" ("+prod.Domain+")", i)
	}

	var idx int
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[int]().
				Title("削除する環境を選択").
				Options(options...).
				Value(&idx),
		),
	).WithTheme(huh.ThemeCatppuccin()).Run()
	if err != nil {
		return err
	}

	// 確認
	confirm, err := prompt.AskConfirm(fmt.Sprintf("本当に「%s」を削除しますか?", cfg.Kintone.Prod[idx].Name), false)
	if err != nil {
		return err
	}

	if !confirm {
		fmt.Println("削除をキャンセルしました")
		return nil
	}

	name := cfg.Kintone.Prod[idx].Name
	cfg.Kintone.Prod = append(cfg.Kintone.Prod[:idx], cfg.Kintone.Prod[idx+1:]...)

	ui.Error(fmt.Sprintf("本番環境を削除しました: %s", name))
	return nil
}

func editTargets(cfg *config.Config) error {
	fmt.Println()

	desktop, mobile, err := prompt.AskTargets(cfg.Targets.Desktop, cfg.Targets.Mobile)
	if err != nil {
		return err
	}

	cfg.Targets.Desktop = desktop
	cfg.Targets.Mobile = mobile

	ui.Success("ターゲットを更新しました")
	return nil
}

func switchFramework(projectDir string, cfg *config.Config) error {
	fmt.Printf("\n%s フレームワークの切り替え\n\n", ui.InfoStyle.Render("🔧"))

	// 現在のフレームワークを検出
	currentFramework := detectCurrentFramework(projectDir)
	fmt.Printf("現在のフレームワーク: %s\n\n", ui.InfoStyle.Render(string(currentFramework)))

	// 新しいフレームワークを選択（現在のフレームワークは除外）
	newFramework, err := prompt.AskFrameworkExcept(currentFramework)
	if err != nil {
		return err
	}

	// 言語を選択
	newLanguage, err := prompt.AskLanguage()
	if err != nil {
		return err
	}

	// パッケージマネージャーを取得
	pm := cfg.GetPackageManager(projectDir)

	// 確認
	confirm, err := prompt.AskConfirm(fmt.Sprintf("%s から %s に切り替えますか? (パッケージの再インストールが必要です)", currentFramework, newFramework), true)
	if err != nil {
		return err
	}

	if !confirm {
		fmt.Println("キャンセルしました")
		return nil
	}

	ui.Info("フレームワークを切り替え中...")

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
	fmt.Printf(" %s\n", ui.SuccessStyle.Render(ui.IconSuccess))

	// eslint.config.js を再生成（既存ファイルを削除してから）
	fmt.Printf("  ESLint設定を再生成中...")
	eslintPath := filepath.Join(projectDir, "eslint.config.js")
	os.Remove(eslintPath)
	if err := generator.GenerateESLintConfig(projectDir, newFramework, newLanguage); err != nil {
		return fmt.Errorf("ESLint設定生成エラー: %w", err)
	}
	fmt.Printf(" %s\n", ui.SuccessStyle.Render(ui.IconSuccess))

	// config.json のエントリーパスを更新
	cfg.Dev.Entry.Main = generator.GetEntryPath(newFramework, newLanguage, "main")
	cfg.Dev.Entry.Config = generator.GetEntryPath(newFramework, newLanguage, "config")

	fmt.Println()
	ui.Success(fmt.Sprintf("フレームワークを %s に切り替えました", newFramework))
	ui.Info("ソースファイルは手動で更新してください")

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
	fmt.Printf("\n%s エントリーポイントの設定\n\n", ui.InfoStyle.Render("🔧"))

	fmt.Printf("現在のエントリーポイント:\n")
	fmt.Printf("  main:   %s\n", ui.InfoStyle.Render(cfg.Dev.Entry.Main))
	fmt.Printf("  config: %s\n\n", ui.InfoStyle.Render(cfg.Dev.Entry.Config))

	// mainエントリーポイント
	mainEntry, err := askInput("main エントリーポイント", cfg.Dev.Entry.Main, true)
	if err != nil {
		return err
	}

	// configエントリーポイント
	configEntry, err := askInput("config エントリーポイント", cfg.Dev.Entry.Config, true)
	if err != nil {
		return err
	}

	cfg.Dev.Entry.Main = mainEntry
	cfg.Dev.Entry.Config = configEntry

	ui.Success("エントリーポイントを更新しました")
	return nil
}

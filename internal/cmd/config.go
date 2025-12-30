package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

		fmt.Printf("%s\n\n", ui.InfoStyle.Render("設定メニュー"))

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
			if err := editTargets(cwd, cfg); err != nil {
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
	fmt.Print("\033[H\033[2J")
	fmt.Printf("%s\n\n", ui.InfoStyle.Render("現在の設定"))

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

	// メニューに戻る前に一時停止
	prompt.AskConfirm("メニューに戻る", true)
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

	// 標準順序でJSONを生成
	data := orderedManifestJSON(manifest)

	return os.WriteFile(manifestPath, []byte(data), 0644)
}

// orderedManifestJSON はmanifestを標準順序でJSON文字列に変換する
func orderedManifestJSON(manifest map[string]interface{}) string {
	var sb strings.Builder
	sb.WriteString("{\n")

	// 順序: version, manifest_version, type, icon, name, description, config, desktop, mobile
	keys := []string{"version", "manifest_version", "type", "icon", "name", "description", "config", "desktop", "mobile"}

	first := true
	for _, key := range keys {
		if val, ok := manifest[key]; ok {
			if !first {
				sb.WriteString(",\n")
			}
			first = false
			writeJSONField(&sb, key, val, "  ")
		}
	}

	// その他のキーを追加
	for key, val := range manifest {
		found := false
		for _, k := range keys {
			if k == key {
				found = true
				break
			}
		}
		if !found {
			if !first {
				sb.WriteString(",\n")
			}
			first = false
			writeJSONField(&sb, key, val, "  ")
		}
	}

	sb.WriteString("\n}")
	return sb.String()
}

func writeJSONField(sb *strings.Builder, key string, val interface{}, indent string) {
	jsonVal, _ := json.MarshalIndent(val, indent, "  ")
	// インデントを調整
	jsonStr := string(jsonVal)
	sb.WriteString(indent)
	sb.WriteString("\"")
	sb.WriteString(key)
	sb.WriteString("\": ")
	sb.WriteString(jsonStr)
}

func editManifest(projectDir string) error {
	fmt.Print("\033[H\033[2J")
	fmt.Printf("%s\n\n", ui.InfoStyle.Render("プラグイン情報の編集"))

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
	fmt.Print("\033[H\033[2J")
	fmt.Printf("%s\n\n", ui.InfoStyle.Render("開発環境の設定"))

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
	fmt.Print("\033[H\033[2J")
	fmt.Printf("%s\n\n", ui.InfoStyle.Render("本番環境の管理"))

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

func editTargets(projectDir string, cfg *config.Config) error {
	fmt.Print("\033[H\033[2J")
	fmt.Printf("%s\n\n", ui.InfoStyle.Render("ターゲットの設定"))

	desktop, mobile, err := prompt.AskTargets(cfg.Targets.Desktop, cfg.Targets.Mobile)
	if err != nil {
		return err
	}

	cfg.Targets.Desktop = desktop
	cfg.Targets.Mobile = mobile

	// manifest.json も更新
	manifest, err := loadManifest(projectDir)
	if err != nil {
		return fmt.Errorf("manifest.json の読み込みに失敗しました: %w", err)
	}

	// desktop/mobile の設定を更新
	if desktop {
		if manifest["desktop"] == nil {
			manifest["desktop"] = map[string]interface{}{
				"js": []interface{}{"js/desktop.js"},
			}
		}
	} else {
		delete(manifest, "desktop")
	}

	if mobile {
		if manifest["mobile"] == nil {
			manifest["mobile"] = map[string]interface{}{
				"js": []interface{}{"js/mobile.js"},
			}
		}
	} else {
		delete(manifest, "mobile")
	}

	if err := saveManifest(projectDir, manifest); err != nil {
		return fmt.Errorf("manifest.json の保存に失敗しました: %w", err)
	}

	ui.Success("ターゲットを更新しました")
	return nil
}

func switchFramework(projectDir string, cfg *config.Config) error {
	fmt.Print("\033[H\033[2J")
	fmt.Printf("%s\n\n", ui.InfoStyle.Render("フレームワークの切り替え"))

	// 現在のフレームワークと言語を検出
	currentFramework := detectCurrentFramework(projectDir)
	currentLanguage := detectCurrentLanguage(projectDir)
	fmt.Printf("現在の構成: %s + %s\n\n", prompt.FormatFramework(currentFramework), prompt.FormatLanguage(currentLanguage))

	// 新しいフレームワークを選択
	newFramework, err := prompt.AskFramework()
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
	confirm, err := prompt.AskConfirm(fmt.Sprintf("%s + %s に切り替えますか? (パッケージの再インストールが必要です)", prompt.FormatFramework(newFramework), prompt.FormatLanguage(newLanguage)), true)
	if err != nil {
		return err
	}

	if !confirm {
		fmt.Println("キャンセルしました")
		return nil
	}

	ui.Info("フレームワークを切り替え中...")
	fmt.Println()

	// パッケージマネージャーごとのコマンドを設定
	var removeCmd, addCmd, addDevFlag string
	switch pm {
	case "yarn", "pnpm", "bun":
		removeCmd = "remove"
		addCmd = "add"
		addDevFlag = "-D"
	default: // npm
		removeCmd = "uninstall"
		addCmd = "install"
		addDevFlag = "-D"
	}

	// 全フレームワーク関連パッケージを削除（同じフレームワークでも言語変更に対応）
	allOldPkgs := getAllFrameworkPackages()
	existingPkgs := filterExistingPackages(projectDir, allOldPkgs)
	if len(existingPkgs) > 0 {
		args := append([]string{removeCmd}, existingPkgs...)
		ui.RunCommandWithSpinner("旧パッケージを削除中...", pm, args, projectDir)
	}

	// 新しいフレームワークのパッケージをインストール
	newDeps, newDevDeps := getFrameworkPackageNames(newFramework, newLanguage)

	if len(newDeps) > 0 {
		args := append([]string{addCmd}, newDeps...)
		if err := ui.RunCommandWithSpinner("依存パッケージをインストール中...", pm, args, projectDir); err != nil {
			return fmt.Errorf("依存パッケージインストールエラー: %w", err)
		}
	}

	if len(newDevDeps) > 0 {
		args := append([]string{addCmd, addDevFlag}, newDevDeps...)
		if err := ui.RunCommandWithSpinner("開発パッケージをインストール中...", pm, args, projectDir); err != nil {
			return fmt.Errorf("開発パッケージインストールエラー: %w", err)
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
	ui.Success(fmt.Sprintf("フレームワークを %s に切り替えました", prompt.FormatFramework(newFramework)))
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

// getAllFrameworkPackages は全フレームワーク関連パッケージを返す
func getAllFrameworkPackages() []string {
	allPkgs := []string{}
	seen := make(map[string]bool)

	frameworks := []prompt.Framework{
		prompt.FrameworkReact,
		prompt.FrameworkVue,
		prompt.FrameworkSvelte,
	}

	// TypeScriptとJavaScript両方のパッケージを収集（重複排除）
	for _, fw := range frameworks {
		for _, lang := range []prompt.Language{prompt.LanguageTypeScript, prompt.LanguageJavaScript} {
			deps, devDeps := getFrameworkPackageNames(fw, lang)
			for _, pkg := range deps {
				if !seen[pkg] {
					seen[pkg] = true
					allPkgs = append(allPkgs, pkg)
				}
			}
			for _, pkg := range devDeps {
				if !seen[pkg] {
					seen[pkg] = true
					allPkgs = append(allPkgs, pkg)
				}
			}
		}
	}

	return allPkgs
}

// filterExistingPackages はpackage.jsonに存在するパッケージのみをフィルタリングする
func filterExistingPackages(projectDir string, pkgs []string) []string {
	pkgPath := filepath.Join(projectDir, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return nil
	}

	var pkg map[string]interface{}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil
	}

	// dependencies と devDependencies を取得
	deps := make(map[string]bool)
	if d, ok := pkg["dependencies"].(map[string]interface{}); ok {
		for k := range d {
			deps[k] = true
		}
	}
	if d, ok := pkg["devDependencies"].(map[string]interface{}); ok {
		for k := range d {
			deps[k] = true
		}
	}

	// 存在するパッケージのみをフィルタリング
	existing := []string{}
	for _, p := range pkgs {
		if deps[p] {
			existing = append(existing, p)
		}
	}

	return existing
}

// getFrameworkPackageNames はフレームワークの依存パッケージと開発パッケージを返す
func getFrameworkPackageNames(fw prompt.Framework, lang prompt.Language) (deps []string, devDeps []string) {
	switch fw {
	case prompt.FrameworkReact:
		deps = []string{"react", "react-dom"}
		devDeps = []string{
			"@vitejs/plugin-react",
			"eslint-plugin-react-hooks",
			"eslint-plugin-react-refresh",
		}
		if lang == prompt.LanguageTypeScript {
			devDeps = append(devDeps, "@types/react", "@types/react-dom")
		}
	case prompt.FrameworkVue:
		deps = []string{"vue"}
		devDeps = []string{
			"@vitejs/plugin-vue",
			"eslint-plugin-vue",
		}
		if lang == prompt.LanguageTypeScript {
			devDeps = append(devDeps, "vue-tsc")
		}
	case prompt.FrameworkSvelte:
		deps = []string{"svelte"}
		devDeps = []string{
			"@sveltejs/vite-plugin-svelte",
			"eslint-plugin-svelte",
			"svelte-eslint-parser",
		}
		if lang == prompt.LanguageTypeScript {
			devDeps = append(devDeps, "svelte-check")
		}
	}
	return
}

func editEntryPoints(projectDir string, cfg *config.Config) error {
	fmt.Print("\033[H\033[2J")
	fmt.Printf("%s\n\n", ui.InfoStyle.Render("エントリーポイントの設定"))

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

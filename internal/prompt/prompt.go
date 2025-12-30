package prompt

import (
	"errors"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

var errRequired = errors.New("入力必須です")

// CompleteDomain はサブドメインのみの入力を完全なドメインに補完する
func CompleteDomain(domain string) string {
	domain = strings.TrimSpace(domain)
	if !strings.Contains(domain, ".") {
		return domain + ".cybozu.com"
	}
	return domain
}

type Framework string

const (
	FrameworkReact   Framework = "react"
	FrameworkVue     Framework = "vue"
	FrameworkSvelte  Framework = "svelte"
	FrameworkVanilla Framework = "vanilla"
)

type Language string

const (
	LanguageTypeScript Language = "typescript"
	LanguageJavaScript Language = "javascript"
)

type PackageManager string

const (
	PackageManagerNpm  PackageManager = "npm"
	PackageManagerPnpm PackageManager = "pnpm"
	PackageManagerYarn PackageManager = "yarn"
	PackageManagerBun  PackageManager = "bun"
)

type InitAnswers struct {
	ProjectName    string
	PluginNameJa   string
	PluginNameEn   string
	DescriptionJa  string
	DescriptionEn  string
	CreateDir      bool
	Domain         string
	Framework      Framework
	Language       Language
	Username       string
	Password       string
	PackageManager PackageManager
	TargetDesktop  bool
	TargetMobile   bool
}

// カラー定義
var (
	colorCyan   = lipgloss.Color("39")
	colorGreen  = lipgloss.Color("42")
	colorYellow = lipgloss.Color("214")
	colorRed    = lipgloss.Color("196")
	colorOrange = lipgloss.Color("208")
	colorBlue   = lipgloss.Color("33")
	colorWhite  = lipgloss.Color("255")
)

func newForm(groups ...*huh.Group) *huh.Form {
	return huh.NewForm(groups...).WithTheme(huh.ThemeCatppuccin())
}

// FormatFramework はフレームワーク名を色付きで返す
func FormatFramework(framework Framework) string {
	cyanStyle := lipgloss.NewStyle().Foreground(colorCyan)
	greenStyle := lipgloss.NewStyle().Foreground(colorGreen)
	orangeStyle := lipgloss.NewStyle().Foreground(colorOrange)
	yellowStyle := lipgloss.NewStyle().Foreground(colorYellow)

	switch framework {
	case FrameworkReact:
		return cyanStyle.Render("React")
	case FrameworkVue:
		return greenStyle.Render("Vue")
	case FrameworkSvelte:
		return orangeStyle.Render("Svelte")
	case FrameworkVanilla:
		return yellowStyle.Render("Vanilla")
	default:
		return string(framework)
	}
}

// FormatLanguage は言語名を色付きで返す
func FormatLanguage(language Language) string {
	cyanStyle := lipgloss.NewStyle().Foreground(colorCyan)
	yellowStyle := lipgloss.NewStyle().Foreground(colorYellow)

	switch language {
	case LanguageTypeScript:
		return cyanStyle.Render("TypeScript")
	case LanguageJavaScript:
		return yellowStyle.Render("JavaScript")
	default:
		return string(language)
	}
}

func AskCreateDir() (bool, error) {
	var answer bool
	err := newForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("プロジェクトディレクトリを作成しますか?").
				Affirmative("はい").
				Negative("いいえ").
				Value(&answer),
		),
	).Run()
	if err != nil {
		return false, err
	}
	return answer, nil
}

func AskProjectName(defaultVal string) (string, error) {
	var answer string
	err := newForm(
		huh.NewGroup(
			huh.NewInput().
				Title("プロジェクト名").
				Value(&answer).
				Placeholder(defaultVal).
				Validate(func(s string) error {
					if s == "" {
						return errRequired
					}
					return nil
				}),
		),
	).Run()
	if err != nil {
		return "", err
	}
	if answer == "" {
		answer = defaultVal
	}
	return answer, nil
}

func AskPluginNameJa(defaultVal string) (string, error) {
	var answer string
	err := newForm(
		huh.NewGroup(
			huh.NewInput().
				Title("プラグイン名 (日本語)").
				Value(&answer).
				Placeholder(defaultVal).
				Validate(func(s string) error {
					if s == "" {
						return errRequired
					}
					return nil
				}),
		),
	).Run()
	if err != nil {
		return "", err
	}
	if answer == "" {
		answer = defaultVal
	}
	return answer, nil
}

func AskPluginNameEn(defaultVal string) (string, error) {
	var answer string
	err := newForm(
		huh.NewGroup(
			huh.NewInput().
				Title("プラグイン名 (English)").
				Value(&answer).
				Placeholder(defaultVal).
				Validate(func(s string) error {
					if s == "" {
						return errRequired
					}
					return nil
				}),
		),
	).Run()
	if err != nil {
		return "", err
	}
	if answer == "" {
		answer = defaultVal
	}
	return answer, nil
}

func AskDomain(defaultVal string) (string, error) {
	var answer string
	placeholder := "example または example.cybozu.com"
	if defaultVal != "" {
		placeholder = defaultVal
	}
	err := newForm(
		huh.NewGroup(
			huh.NewInput().
				Title("kintone ドメイン").
				Description("例: example または example.cybozu.com").
				Value(&answer).
				Placeholder(placeholder).
				Validate(func(s string) error {
					if s == "" && defaultVal == "" {
						return errRequired
					}
					return nil
				}),
		),
	).Run()
	if err != nil {
		return "", err
	}
	if answer == "" {
		answer = defaultVal
	}
	return CompleteDomain(answer), nil
}

func AskDescriptionJa(defaultVal string) (string, error) {
	var answer string
	err := newForm(
		huh.NewGroup(
			huh.NewInput().
				Title("プラグインの説明 (日本語)").
				Value(&answer).
				Placeholder(defaultVal),
		),
	).Run()
	if err != nil {
		return "", err
	}
	if answer == "" {
		answer = defaultVal
	}
	return answer, nil
}

func AskDescriptionEn(defaultVal string) (string, error) {
	var answer string
	err := newForm(
		huh.NewGroup(
			huh.NewInput().
				Title("プラグインの説明 (English)").
				Value(&answer).
				Placeholder(defaultVal),
		),
	).Run()
	if err != nil {
		return "", err
	}
	if answer == "" {
		answer = defaultVal
	}
	return answer, nil
}

func AskFramework() (Framework, error) {
	return AskFrameworkExcept("")
}

func AskFrameworkExcept(exclude Framework) (Framework, error) {
	cyanStyle := lipgloss.NewStyle().Foreground(colorCyan)
	greenStyle := lipgloss.NewStyle().Foreground(colorGreen)
	orangeStyle := lipgloss.NewStyle().Foreground(colorOrange)
	yellowStyle := lipgloss.NewStyle().Foreground(colorYellow)

	var options []huh.Option[Framework]
	if exclude != FrameworkReact {
		options = append(options, huh.NewOption(cyanStyle.Render("React"), FrameworkReact))
	}
	if exclude != FrameworkVue {
		options = append(options, huh.NewOption(greenStyle.Render("Vue"), FrameworkVue))
	}
	if exclude != FrameworkSvelte {
		options = append(options, huh.NewOption(orangeStyle.Render("Svelte"), FrameworkSvelte))
	}
	if exclude != FrameworkVanilla {
		options = append(options, huh.NewOption(yellowStyle.Render("Vanilla"), FrameworkVanilla))
	}

	var answer Framework
	err := newForm(
		huh.NewGroup(
			huh.NewSelect[Framework]().
				Title("フレームワーク").
				Options(options...).
				Value(&answer),
		),
	).Run()
	if err != nil {
		return "", err
	}
	return answer, nil
}

func AskLanguage() (Language, error) {
	cyanStyle := lipgloss.NewStyle().Foreground(colorCyan)
	yellowStyle := lipgloss.NewStyle().Foreground(colorYellow)

	var answer Language
	err := newForm(
		huh.NewGroup(
			huh.NewSelect[Language]().
				Title("言語").
				Options(
					huh.NewOption(cyanStyle.Render("TypeScript"), LanguageTypeScript),
					huh.NewOption(yellowStyle.Render("JavaScript"), LanguageJavaScript),
				).
				Value(&answer),
		),
	).Run()
	if err != nil {
		return "", err
	}
	return answer, nil
}

func AskUsername() (string, error) {
	var answer string
	err := newForm(
		huh.NewGroup(
			huh.NewInput().
				Title("kintone ユーザー名").
				Value(&answer).
				Validate(func(s string) error {
					if s == "" {
						return errRequired
					}
					return nil
				}),
		),
	).Run()
	if err != nil {
		return "", err
	}
	return answer, nil
}

func AskPassword() (string, error) {
	var answer string
	err := newForm(
		huh.NewGroup(
			huh.NewInput().
				Title("kintone パスワード").
				EchoMode(huh.EchoModePassword).
				Value(&answer).
				Validate(func(s string) error {
					if s == "" {
						return errRequired
					}
					return nil
				}),
		),
	).Run()
	if err != nil {
		return "", err
	}
	return answer, nil
}

func AskPackageManager() (PackageManager, error) {
	redStyle := lipgloss.NewStyle().Foreground(colorRed)
	cyanStyle := lipgloss.NewStyle().Foreground(colorCyan)
	blueStyle := lipgloss.NewStyle().Foreground(colorBlue)
	whiteStyle := lipgloss.NewStyle().Foreground(colorWhite)

	var answer PackageManager
	err := newForm(
		huh.NewGroup(
			huh.NewSelect[PackageManager]().
				Title("パッケージマネージャー").
				Options(
					huh.NewOption(redStyle.Render("npm"), PackageManagerNpm),
					huh.NewOption(cyanStyle.Render("pnpm"), PackageManagerPnpm),
					huh.NewOption(blueStyle.Render("yarn"), PackageManagerYarn),
					huh.NewOption(whiteStyle.Render("bun"), PackageManagerBun),
				).
				Value(&answer),
		),
	).Run()
	if err != nil {
		return "", err
	}
	return answer, nil
}

func AskTargets(defaultDesktop, defaultMobile bool) (desktop bool, mobile bool, err error) {
	var answers []string

	err = newForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("対象画面").
				Options(
					huh.NewOption("デスクトップ", "desktop").Selected(defaultDesktop),
					huh.NewOption("モバイル", "mobile").Selected(defaultMobile),
				).
				Value(&answers).
				Validate(func(s []string) error {
					if len(s) == 0 {
						return errRequired
					}
					return nil
				}),
		),
	).Run()
	if err != nil {
		return false, false, err
	}

	for _, a := range answers {
		if a == "desktop" {
			desktop = true
		}
		if a == "mobile" {
			mobile = true
		}
	}

	return desktop, mobile, nil
}

// ProdEnvironment は本番環境の設定を表す
type ProdEnvironment struct {
	Name     string
	Domain   string
	Username string
	Password string
}

// AskProdEnvironment は本番環境の設定を対話形式で取得する
func AskProdEnvironment() (*ProdEnvironment, error) {
	env := &ProdEnvironment{}

	// 環境名
	err := newForm(
		huh.NewGroup(
			huh.NewInput().
				Title("本番環境名").
				Description("例: production").
				Value(&env.Name).
				Placeholder("production").
				Validate(func(s string) error {
					if s == "" {
						return errRequired
					}
					return nil
				}),
		),
	).Run()
	if err != nil {
		return nil, err
	}
	if env.Name == "" {
		env.Name = "production"
	}

	// ドメイン
	err = newForm(
		huh.NewGroup(
			huh.NewInput().
				Title("kintone ドメイン").
				Description("例: example または example.cybozu.com").
				Value(&env.Domain).
				Validate(func(s string) error {
					if s == "" {
						return errRequired
					}
					return nil
				}),
		),
	).Run()
	if err != nil {
		return nil, err
	}
	env.Domain = CompleteDomain(env.Domain)

	// ユーザー名
	err = newForm(
		huh.NewGroup(
			huh.NewInput().
				Title("kintone ユーザー名").
				Value(&env.Username).
				Validate(func(s string) error {
					if s == "" {
						return errRequired
					}
					return nil
				}),
		),
	).Run()
	if err != nil {
		return nil, err
	}

	// パスワード
	err = newForm(
		huh.NewGroup(
			huh.NewInput().
				Title("kintone パスワード").
				EchoMode(huh.EchoModePassword).
				Value(&env.Password).
				Validate(func(s string) error {
					if s == "" {
						return errRequired
					}
					return nil
				}),
		),
	).Run()
	if err != nil {
		return nil, err
	}

	return env, nil
}

// ZipFileChoice はZIPファイル選択の結果を表す
type ZipFileChoice struct {
	FilePath string
	BuildNew bool
}

// AskZipFile はdist内のZIPファイルを選択するプロンプトを表示する
func AskZipFile(zipFiles []string) (*ZipFileChoice, error) {
	greenStyle := lipgloss.NewStyle().Foreground(colorGreen)

	// 選択肢を作成（新規ビルドを先頭に）
	options := make([]huh.Option[string], len(zipFiles)+1)
	options[0] = huh.NewOption(greenStyle.Render("新規ビルド"), "_build_new_")
	for i, f := range zipFiles {
		options[i+1] = huh.NewOption(f, f)
	}

	var answer string
	err := newForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("デプロイするファイルを選択").
				Options(options...).
				Value(&answer),
		),
	).Run()
	if err != nil {
		return nil, err
	}

	if answer == "_build_new_" {
		return &ZipFileChoice{BuildNew: true}, nil
	}

	return &ZipFileChoice{FilePath: answer}, nil
}

// DeployTarget はデプロイ先を表す
type DeployTarget struct {
	Name   string
	Domain string
}

// DeployTargetChoice はデプロイ先選択の結果を表す
type DeployTargetChoice struct {
	Indices []int
	AddNew  bool
}

// AskDeployTargets はデプロイ先を選択するプロンプトを表示する
func AskDeployTargets(targets []DeployTarget) (*DeployTargetChoice, error) {
	greenStyle := lipgloss.NewStyle().Foreground(colorGreen)

	// 選択肢を作成（新規環境追加を先頭に）
	options := make([]huh.Option[string], len(targets)+1)
	options[0] = huh.NewOption(greenStyle.Render("+ 新規環境を追加"), "_add_new_")
	for i, t := range targets {
		options[i+1] = huh.NewOption(t.Name+" ("+t.Domain+")", t.Name)
	}

	var selected []string
	err := newForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("デプロイ先を選択").
				Options(options...).
				Value(&selected).
				Validate(func(s []string) error {
					if len(s) == 0 {
						return errRequired
					}
					return nil
				}),
		),
	).Run()
	if err != nil {
		return nil, err
	}

	// 新規環境追加が選択されたかチェック
	addNew := false
	for _, s := range selected {
		if s == "_add_new_" {
			addNew = true
			break
		}
	}

	// 選択されたインデックスを返す（新規環境追加以外）
	indices := make([]int, 0, len(selected))
	for _, s := range selected {
		if s == "_add_new_" {
			continue
		}
		for i, t := range targets {
			if s == t.Name {
				indices = append(indices, i)
				break
			}
		}
	}

	return &DeployTargetChoice{
		Indices: indices,
		AddNew:  addNew,
	}, nil
}

// AskConfirm は確認プロンプトを表示する
func AskConfirm(message string, defaultVal bool) (bool, error) {
	var answer bool = defaultVal
	err := newForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(message).
				Affirmative("はい").
				Negative("いいえ").
				Value(&answer),
		),
	).Run()
	if err != nil {
		return false, err
	}
	return answer, nil
}

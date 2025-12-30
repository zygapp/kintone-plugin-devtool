package prompt

import (
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
)

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
	ProjectName      string
	PluginNameJa     string
	PluginNameEn     string
	DescriptionJa    string
	DescriptionEn    string
	CreateDir        bool
	Domain           string
	Framework        Framework
	Language         Language
	Username         string
	Password         string
	PackageManager   PackageManager
	TargetDesktop    bool
	TargetMobile     bool
}

func AskCreateDir() (bool, error) {
	var answer bool
	prompt := &survey.Confirm{
		Message: "プロジェクトディレクトリを作成しますか?",
		Default: true,
	}
	if err := survey.AskOne(prompt, &answer); err != nil {
		return false, err
	}
	return answer, nil
}

func AskProjectName(defaultVal string) (string, error) {
	var answer string
	prompt := &survey.Input{
		Message: "プロジェクト名:",
		Default: defaultVal,
	}
	if err := survey.AskOne(prompt, &answer, survey.WithValidator(survey.Required)); err != nil {
		return "", err
	}
	return answer, nil
}

func AskPluginNameJa(defaultVal string) (string, error) {
	var answer string
	prompt := &survey.Input{
		Message: "プラグイン名 (日本語):",
		Default: defaultVal,
	}
	if err := survey.AskOne(prompt, &answer, survey.WithValidator(survey.Required)); err != nil {
		return "", err
	}
	return answer, nil
}

func AskPluginNameEn(defaultVal string) (string, error) {
	var answer string
	prompt := &survey.Input{
		Message: "プラグイン名 (English):",
		Default: defaultVal,
	}
	if err := survey.AskOne(prompt, &answer, survey.WithValidator(survey.Required)); err != nil {
		return "", err
	}
	return answer, nil
}

func AskDomain(defaultVal string) (string, error) {
	var answer string
	prompt := &survey.Input{
		Message: "kintone ドメイン (例: example.cybozu.com):",
		Default: defaultVal,
	}
	if err := survey.AskOne(prompt, &answer, survey.WithValidator(survey.Required)); err != nil {
		return "", err
	}
	return CompleteDomain(answer), nil
}

func AskDescriptionJa(defaultVal string) (string, error) {
	var answer string
	prompt := &survey.Input{
		Message: "プラグインの説明 (日本語):",
		Default: defaultVal,
	}
	if err := survey.AskOne(prompt, &answer); err != nil {
		return "", err
	}
	return answer, nil
}

func AskDescriptionEn(defaultVal string) (string, error) {
	var answer string
	prompt := &survey.Input{
		Message: "プラグインの説明 (English):",
		Default: defaultVal,
	}
	if err := survey.AskOne(prompt, &answer); err != nil {
		return "", err
	}
	return answer, nil
}

func AskFramework() (Framework, error) {
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	hiRed := color.New(color.FgHiRed).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	options := []string{
		cyan("React"),
		green("Vue"),
		hiRed("Svelte"),
		yellow("Vanilla"),
	}

	var answer string
	prompt := &survey.Select{
		Message: "フレームワーク:",
		Options: options,
		Default: options[0],
	}
	if err := survey.AskOne(prompt, &answer); err != nil {
		return "", err
	}

	switch answer {
	case options[0]:
		return FrameworkReact, nil
	case options[1]:
		return FrameworkVue, nil
	case options[2]:
		return FrameworkSvelte, nil
	case options[3]:
		return FrameworkVanilla, nil
	}
	return FrameworkReact, nil
}

func AskLanguage() (Language, error) {
	cyan := color.New(color.FgCyan).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	options := []string{
		cyan("TypeScript"),
		yellow("JavaScript"),
	}

	var answer string
	prompt := &survey.Select{
		Message: "言語:",
		Options: options,
		Default: options[0],
	}
	if err := survey.AskOne(prompt, &answer); err != nil {
		return "", err
	}

	switch answer {
	case options[0]:
		return LanguageTypeScript, nil
	case options[1]:
		return LanguageJavaScript, nil
	}
	return LanguageTypeScript, nil
}

func AskUsername() (string, error) {
	var answer string
	prompt := &survey.Input{
		Message: "kintone ユーザー名:",
	}
	if err := survey.AskOne(prompt, &answer, survey.WithValidator(survey.Required)); err != nil {
		return "", err
	}
	return answer, nil
}

func AskPassword() (string, error) {
	var answer string
	prompt := &survey.Password{
		Message: "kintone パスワード:",
	}
	if err := survey.AskOne(prompt, &answer, survey.WithValidator(survey.Required)); err != nil {
		return "", err
	}
	return answer, nil
}

func AskPackageManager() (PackageManager, error) {
	red := color.New(color.FgRed).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	blue := color.New(color.FgBlue).SprintFunc()
	white := color.New(color.FgWhite).SprintFunc()

	options := []string{
		red("npm"),
		cyan("pnpm"),
		blue("yarn"),
		white("bun"),
	}

	var answer string
	prompt := &survey.Select{
		Message: "パッケージマネージャー:",
		Options: options,
		Default: options[0],
	}
	if err := survey.AskOne(prompt, &answer); err != nil {
		return "", err
	}

	switch answer {
	case options[0]:
		return PackageManagerNpm, nil
	case options[1]:
		return PackageManagerPnpm, nil
	case options[2]:
		return PackageManagerYarn, nil
	case options[3]:
		return PackageManagerBun, nil
	}
	return PackageManagerNpm, nil
}

func AskTargets(defaultDesktop, defaultMobile bool) (desktop bool, mobile bool, err error) {
	options := []string{
		"デスクトップ",
		"モバイル",
	}

	defaults := []string{}
	if defaultDesktop {
		defaults = append(defaults, options[0])
	}
	if defaultMobile {
		defaults = append(defaults, options[1])
	}

	var answers []string
	prompt := &survey.MultiSelect{
		Message: "対象画面:",
		Options: options,
		Default: defaults,
	}
	if err := survey.AskOne(prompt, &answers, survey.WithValidator(survey.MinItems(1))); err != nil {
		return false, false, err
	}

	for _, a := range answers {
		if a == options[0] {
			desktop = true
		}
		if a == options[1] {
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
	namePrompt := &survey.Input{
		Message: "本番環境名 (例: production):",
		Default: "production",
	}
	if err := survey.AskOne(namePrompt, &env.Name, survey.WithValidator(survey.Required)); err != nil {
		return nil, err
	}

	// ドメイン
	domainPrompt := &survey.Input{
		Message: "kintone ドメイン (例: example.cybozu.com):",
	}
	if err := survey.AskOne(domainPrompt, &env.Domain, survey.WithValidator(survey.Required)); err != nil {
		return nil, err
	}
	env.Domain = CompleteDomain(env.Domain)

	// ユーザー名
	usernamePrompt := &survey.Input{
		Message: "kintone ユーザー名:",
	}
	if err := survey.AskOne(usernamePrompt, &env.Username, survey.WithValidator(survey.Required)); err != nil {
		return nil, err
	}

	// パスワード
	passwordPrompt := &survey.Password{
		Message: "kintone パスワード:",
	}
	if err := survey.AskOne(passwordPrompt, &env.Password, survey.WithValidator(survey.Required)); err != nil {
		return nil, err
	}

	return env, nil
}

// ZipFileChoice はZIPファイル選択の結果を表す
type ZipFileChoice struct {
	FilePath    string
	BuildNew    bool
}

// AskZipFile はdist内のZIPファイルを選択するプロンプトを表示する
func AskZipFile(zipFiles []string) (*ZipFileChoice, error) {
	green := color.New(color.FgGreen).SprintFunc()

	// 選択肢を作成（新規ビルドを先頭に）
	options := make([]string, len(zipFiles)+1)
	options[0] = green("新規ビルド")
	for i, f := range zipFiles {
		options[i+1] = f
	}

	var answer string
	prompt := &survey.Select{
		Message: "デプロイするファイルを選択:",
		Options: options,
		Default: options[0],
	}
	if err := survey.AskOne(prompt, &answer); err != nil {
		return nil, err
	}

	if answer == options[0] {
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
	green := color.New(color.FgGreen).SprintFunc()

	// 選択肢を作成（新規環境追加を先頭に）
	options := make([]string, len(targets)+1)
	options[0] = green("+ 新規環境を追加")
	for i, t := range targets {
		options[i+1] = t.Name + " (" + t.Domain + ")"
	}

	var selected []string
	prompt := &survey.MultiSelect{
		Message: "デプロイ先を選択してください:",
		Options: options,
	}
	if err := survey.AskOne(prompt, &selected, survey.WithValidator(survey.MinItems(1))); err != nil {
		return nil, err
	}

	// 新規環境追加が選択されたかチェック
	addNew := false
	for _, s := range selected {
		if s == options[0] {
			addNew = true
			break
		}
	}

	// 選択されたインデックスを返す（新規環境追加以外）
	indices := make([]int, 0, len(selected))
	for _, s := range selected {
		if s == options[0] {
			continue
		}
		for i, t := range targets {
			if s == t.Name+" ("+t.Domain+")" {
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

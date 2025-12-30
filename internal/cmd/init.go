package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/kintone/kpdev/internal/config"
	"github.com/kintone/kpdev/internal/generator"
	"github.com/kintone/kpdev/internal/prompt"
	"github.com/kintone/kpdev/internal/ui"
	"github.com/spf13/cobra"
)

var (
	flagName           string
	flagNameJa         string
	flagNameEn         string
	flagDescriptionJa  string
	flagDescriptionEn  string
	flagDomain         string
	flagFramework      string
	flagLanguage       string
	flagUsername       string
	flagPassword       string
	flagCreateDir      bool
	flagNoCreateDir    bool
	flagDesktop        bool
	flagMobile         bool
	flagPackageManager string
)

var initCmd = &cobra.Command{
	Use:   "init [project-name]",
	Short: "新しいプラグインプロジェクトを初期化",
	Long:  `kintone プラグイン開発用の新しいプロジェクトを作成します。`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().StringVarP(&flagName, "name", "n", "", "プロジェクト名")
	initCmd.Flags().StringVar(&flagNameJa, "name-ja", "", "プラグイン名（日本語）")
	initCmd.Flags().StringVar(&flagNameEn, "name-en", "", "プラグイン名（英語）")
	initCmd.Flags().StringVar(&flagDescriptionJa, "description-ja", "", "プラグイン説明（日本語）")
	initCmd.Flags().StringVar(&flagDescriptionEn, "description-en", "", "プラグイン説明（英語）")
	initCmd.Flags().StringVarP(&flagDomain, "domain", "d", "", "kintone ドメイン")
	initCmd.Flags().StringVarP(&flagFramework, "framework", "f", "", "フレームワーク (react|vue|svelte|vanilla)")
	initCmd.Flags().StringVarP(&flagLanguage, "language", "l", "", "言語 (typescript|javascript)")
	initCmd.Flags().StringVarP(&flagUsername, "username", "u", "", "kintone ユーザー名")
	initCmd.Flags().StringVarP(&flagPassword, "password", "p", "", "kintone パスワード")
	initCmd.Flags().BoolVar(&flagCreateDir, "create-dir", false, "プロジェクトディレクトリを作成")
	initCmd.Flags().BoolVar(&flagNoCreateDir, "no-create-dir", false, "カレントディレクトリに展開")
	initCmd.Flags().BoolVar(&flagDesktop, "desktop", false, "デスクトップを対象に含める")
	initCmd.Flags().BoolVar(&flagMobile, "mobile", false, "モバイルを対象に含める")
	initCmd.Flags().StringVarP(&flagPackageManager, "package-manager", "m", "", "パッケージマネージャー (npm|pnpm|yarn|bun)")
}

func runInit(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	var projectName string
	if len(args) > 0 {
		projectName = args[0]
	}

	answers, err := collectAnswers(cwd, projectName)
	if err != nil {
		return err
	}

	var projectDir string
	if answers.CreateDir {
		projectDir = filepath.Join(cwd, answers.ProjectName)
		if err := os.MkdirAll(projectDir, 0755); err != nil {
			return err
		}
	} else {
		projectDir = cwd
	}

	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	// 既存プロジェクトかどうか判定
	isExisting := false
	if _, err := os.Stat(filepath.Join(projectDir, "package.json")); err == nil {
		isExisting = true
	}

	if isExisting {
		fmt.Printf("\n%s 既存プロジェクトを再初期化中...\n", cyan("→"))
		// 既存プロジェクトでも manifest.json がなければ生成
		manifestPath := filepath.Join(config.GetConfigDir(projectDir), "manifest.json")
		if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
			fmt.Printf("  manifest.json...")
			if err := generator.GenerateManifest(projectDir, answers); err != nil {
				fmt.Println()
				return fmt.Errorf("manifest生成エラー: %w", err)
			}
			fmt.Printf(" %s\n", green("✓"))
		}
	} else {
		fmt.Printf("\n%s プロジェクトを作成中...\n", cyan("→"))
		fmt.Printf("  テンプレート...")
		if err := generator.GenerateProject(projectDir, answers); err != nil {
			fmt.Println()
			return fmt.Errorf("プロジェクト生成エラー: %w", err)
		}
		fmt.Printf(" %s\n", green("✓"))
	}

	fmt.Printf("  Vite設定...")
	if err := generator.GenerateViteConfig(projectDir, answers.Framework, answers.Language); err != nil {
		fmt.Println()
		return fmt.Errorf("Vite設定生成エラー: %w", err)
	}
	fmt.Printf(" %s\n", green("✓"))

	fmt.Printf("  証明書...")
	if err := generator.GenerateCerts(projectDir); err != nil {
		fmt.Println()
		return fmt.Errorf("証明書生成エラー: %w", err)
	}
	fmt.Printf(" %s\n", green("✓"))

	fmt.Printf("  秘密鍵...")
	if err := generator.GenerateKeys(projectDir); err != nil {
		fmt.Println()
		return fmt.Errorf("秘密鍵生成エラー: %w", err)
	}
	fmt.Printf(" %s\n", green("✓"))

	fmt.Printf("  ローダー...")
	if err := generator.GenerateLoader(projectDir, answers, version); err != nil {
		fmt.Println()
		return fmt.Errorf("ローダー生成エラー: %w", err)
	}
	fmt.Printf(" %s\n", green("✓"))

	fmt.Printf("  ESLint設定...")
	if err := generator.GenerateESLintConfig(projectDir, answers.Framework, answers.Language); err != nil {
		fmt.Println()
		return fmt.Errorf("ESLint設定生成エラー: %w", err)
	}
	fmt.Printf(" %s\n", green("✓"))

	// 設定保存
	cfg := &config.Config{
		SchemaVersion: config.CurrentSchemaVersion,
		Kintone: config.KintoneConfig{
			Dev: config.DevEnvConfig{
				Domain: answers.Domain,
				Auth: config.AuthConfig{
					Username: answers.Username,
					Password: answers.Password,
				},
			},
		},
		Dev: config.DevConfig{
			Origin: generator.DevOrigin,
			Entry: config.EntryConfig{
				Main:   generator.GetEntryPath(answers.Framework, answers.Language, "main"),
				Config: generator.GetEntryPath(answers.Framework, answers.Language, "config"),
			},
		},
		Targets: config.TargetsConfig{
			Desktop: answers.TargetDesktop,
			Mobile:  answers.TargetMobile,
		},
		PackageManager: string(answers.PackageManager),
	}
	if err := cfg.Save(projectDir); err != nil {
		return fmt.Errorf("設定保存エラー: %w", err)
	}

	// 新規プロジェクトの場合、パッケージをインストール
	if !isExisting && answers.PackageManager != "" {
		fmt.Println()
		pm := string(answers.PackageManager)
		err := ui.RunCommandWithSpinner(
			fmt.Sprintf("パッケージをインストール中... (%s)", pm),
			pm,
			[]string{"install"},
			projectDir,
		)
		if err != nil {
			return fmt.Errorf("インストールエラー: %w", err)
		}
	}

	printSuccess(projectDir, answers, isExisting)
	return nil
}

func collectAnswers(projectDir string, projectName string) (*prompt.InitAnswers, error) {
	answers := &prompt.InitAnswers{}

	// 既存プロジェクトかどうか判定
	isExisting := false
	if _, err := os.Stat(filepath.Join(projectDir, "package.json")); err == nil {
		isExisting = true
	}

	// loader.meta.json から既存の設定を読み込み
	meta, _ := generator.LoadLoaderMeta(projectDir)

	// ディレクトリ作成（既存プロジェクトでは不要）
	if isExisting {
		answers.CreateDir = false
	} else if flagCreateDir {
		answers.CreateDir = true
	} else if flagNoCreateDir {
		answers.CreateDir = false
	} else {
		createDir, err := prompt.AskCreateDir()
		if err != nil {
			return nil, err
		}
		answers.CreateDir = createDir
	}

	// プロジェクト名
	if flagName != "" {
		answers.ProjectName = flagName
	} else if projectName != "" {
		answers.ProjectName = projectName
	} else if meta != nil && meta.Project.Name != "" {
		answers.ProjectName = meta.Project.Name
	} else {
		defaultName := filepath.Base(projectDir)
		if answers.CreateDir {
			defaultName = "my-kintone-plugin"
		}
		name, err := prompt.AskProjectName(defaultName)
		if err != nil {
			return nil, err
		}
		answers.ProjectName = name
	}

	// プラグイン名（デフォルトはプロジェクト名）
	if flagNameJa != "" {
		answers.PluginNameJa = flagNameJa
	} else {
		pluginNameJa, err := prompt.AskPluginNameJa(answers.ProjectName)
		if err != nil {
			return nil, err
		}
		answers.PluginNameJa = pluginNameJa
	}

	if flagNameEn != "" {
		answers.PluginNameEn = flagNameEn
	} else {
		pluginNameEn, err := prompt.AskPluginNameEn(answers.ProjectName)
		if err != nil {
			return nil, err
		}
		answers.PluginNameEn = pluginNameEn
	}

	// プラグイン説明
	if flagDescriptionJa != "" {
		answers.DescriptionJa = flagDescriptionJa
	} else {
		descJa, err := prompt.AskDescriptionJa("")
		if err != nil {
			return nil, err
		}
		answers.DescriptionJa = descJa
	}

	if flagDescriptionEn != "" {
		answers.DescriptionEn = flagDescriptionEn
	} else {
		descEn, err := prompt.AskDescriptionEn("")
		if err != nil {
			return nil, err
		}
		answers.DescriptionEn = descEn
	}

	// ドメイン
	if flagDomain != "" {
		answers.Domain = prompt.CompleteDomain(flagDomain)
	} else if meta != nil && meta.Kintone.Domain != "" {
		answers.Domain = meta.Kintone.Domain
	} else if cfg, err := config.Load(projectDir); err == nil && cfg.Kintone.Dev.Domain != "" {
		answers.Domain = cfg.Kintone.Dev.Domain
	} else {
		domain, err := prompt.AskDomain("")
		if err != nil {
			return nil, err
		}
		answers.Domain = domain
	}

	// フレームワーク・言語
	if flagFramework != "" && flagLanguage != "" {
		answers.Framework = prompt.Framework(flagFramework)
		answers.Language = prompt.Language(flagLanguage)
	} else if meta != nil && meta.Project.Framework != "" {
		answers.Framework = prompt.Framework(meta.Project.Framework)
		answers.Language = prompt.Language(meta.Project.Language)
	} else if fw, lang := detectFromPackageJSON(projectDir); fw != "" {
		answers.Framework = fw
		answers.Language = lang
	} else {
		if flagFramework != "" {
			answers.Framework = prompt.Framework(flagFramework)
		} else {
			framework, err := prompt.AskFramework()
			if err != nil {
				return nil, err
			}
			answers.Framework = framework
		}

		if flagLanguage != "" {
			answers.Language = prompt.Language(flagLanguage)
		} else {
			language, err := prompt.AskLanguage()
			if err != nil {
				return nil, err
			}
			answers.Language = language
		}
	}

	// 対象画面
	if flagDesktop || flagMobile {
		answers.TargetDesktop = flagDesktop
		answers.TargetMobile = flagMobile
	} else {
		defaultDesktop := true
		defaultMobile := false
		if cfg, err := config.Load(projectDir); err == nil {
			defaultDesktop = cfg.Targets.Desktop
			defaultMobile = cfg.Targets.Mobile
			if !defaultDesktop && !defaultMobile {
				defaultDesktop = true
			}
		}
		desktop, mobile, err := prompt.AskTargets(defaultDesktop, defaultMobile)
		if err != nil {
			return nil, err
		}
		answers.TargetDesktop = desktop
		answers.TargetMobile = mobile
	}

	// 認証情報
	if flagUsername != "" && flagPassword != "" {
		answers.Username = flagUsername
		answers.Password = flagPassword
	} else {
		envCfg, _ := config.LoadEnv(projectDir)
		if envCfg != nil && envCfg.HasAuth() {
			answers.Username = envCfg.Username
			answers.Password = envCfg.Password
		} else {
			if flagUsername != "" {
				answers.Username = flagUsername
			} else {
				username, err := prompt.AskUsername()
				if err != nil {
					return nil, err
				}
				answers.Username = username
			}

			if flagPassword != "" {
				answers.Password = flagPassword
			} else {
				password, err := prompt.AskPassword()
				if err != nil {
					return nil, err
				}
				answers.Password = password
			}
		}
	}

	// パッケージマネージャー（新規プロジェクトのみ）
	if !isExisting {
		if flagPackageManager != "" {
			switch flagPackageManager {
			case "npm":
				answers.PackageManager = prompt.PackageManagerNpm
			case "pnpm":
				answers.PackageManager = prompt.PackageManagerPnpm
			case "yarn":
				answers.PackageManager = prompt.PackageManagerYarn
			case "bun":
				answers.PackageManager = prompt.PackageManagerBun
			default:
				return nil, fmt.Errorf("無効なパッケージマネージャー: %s (npm|pnpm|yarn|bun)", flagPackageManager)
			}
		} else {
			pm, err := prompt.AskPackageManager()
			if err != nil {
				return nil, err
			}
			answers.PackageManager = pm
		}
	}

	return answers, nil
}

func detectFromPackageJSON(projectDir string) (prompt.Framework, prompt.Language) {
	pkgPath := filepath.Join(projectDir, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return "", ""
	}

	content := string(data)

	var framework prompt.Framework
	if strings.Contains(content, `"react"`) {
		framework = prompt.FrameworkReact
	} else if strings.Contains(content, `"vue"`) {
		framework = prompt.FrameworkVue
	} else if strings.Contains(content, `"svelte"`) {
		framework = prompt.FrameworkSvelte
	} else {
		return "", ""
	}

	var language prompt.Language
	if strings.Contains(content, `"typescript"`) {
		language = prompt.LanguageTypeScript
	} else if _, err := os.Stat(filepath.Join(projectDir, "src/main/main.ts")); err == nil {
		language = prompt.LanguageTypeScript
	} else if _, err := os.Stat(filepath.Join(projectDir, "src/main/main.tsx")); err == nil {
		language = prompt.LanguageTypeScript
	} else {
		language = prompt.LanguageJavaScript
	}

	return framework, language
}

func printSuccess(projectDir string, answers *prompt.InitAnswers, isExisting bool) {
	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	// プラグインIDを取得
	meta, _ := generator.LoadLoaderMeta(projectDir)

	if isExisting {
		fmt.Printf("\n%s プロジェクトを再初期化しました!\n", green("✓"))
	} else {
		fmt.Printf("\n%s プロジェクトが作成されました!\n", green("✓"))
	}

	if meta != nil {
		fmt.Printf("\nPlugin ID:\n")
		fmt.Printf("  Dev:  %s\n", cyan(meta.PluginIDs.Dev))
		fmt.Printf("  Prod: %s\n", cyan(meta.PluginIDs.Prod))
	}

	fmt.Printf("\n%s 証明書を信頼する必要があります:\n", yellow("⚠"))
	fmt.Printf("  %s を開いて証明書を承認してください\n", cyan("https://localhost:3000"))

	fmt.Printf("\n次のステップ:\n")
	if answers.CreateDir {
		fmt.Printf("  %s %s\n", cyan("cd"), answers.ProjectName)
	}
	fmt.Printf("  %s\n", cyan("kpdev dev"))
	fmt.Println()
}

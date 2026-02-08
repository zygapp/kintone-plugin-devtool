package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/kintone/kpdev/internal/config"
	"github.com/kintone/kpdev/internal/generator"
	"github.com/kintone/kpdev/internal/plugin"
	"github.com/kintone/kpdev/internal/ui"
	"github.com/spf13/cobra"
)

var (
	flagBuildMode   string
	flagSkipVersion bool
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "本番用プラグインをビルド",
	Long: `本番用プラグインZIPを生成します。

モード:
  prod (デフォルト) - 本番用ビルド (minify + console削除)
  pre              - プレビルド (minifyなし + console残す + 名前に[開発]付与)`,
	RunE: runBuild,
}

func init() {
	rootCmd.AddCommand(buildCmd)

	buildCmd.Flags().StringVar(&flagBuildMode, "mode", "prod", "ビルドモード (prod|pre)")
	buildCmd.Flags().BoolVar(&flagSkipVersion, "skip-version", false, "バージョン確認をスキップ")
}

func runBuild(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	// モードを決定
	buildMode := flagBuildMode
	if !cmd.Flags().Changed("mode") {
		// --mode が指定されていない場合は対話で選択
		selectedMode, err := askBuildMode()
		if err != nil {
			if errors.Is(err, huh.ErrUserAborted) {
				return nil
			}
			return err
		}
		buildMode = selectedMode
	} else {
		// モードを検証
		if buildMode != "prod" && buildMode != "pre" {
			return fmt.Errorf("無効なモードです: %s (prod または pre を指定してください)", buildMode)
		}
	}

	isPre := buildMode == "pre"

	// メタデータを読み込み
	meta, err := generator.LoadLoaderMeta(cwd)
	if err != nil {
		return fmt.Errorf("loader.meta.json が見つかりません。先に kpdev init を実行してください: %w", err)
	}

	// マニフェストを読み込み、バージョン確認
	manifest, err := loadBuildManifest(cwd)
	if err != nil {
		return fmt.Errorf("manifest.json の読み込みに失敗しました: %w", err)
	}

	currentVersion := fmt.Sprintf("%v", manifest["version"])

	// --skip-version でなければバージョン確認
	if !flagSkipVersion {
		newVersion, err := askVersion(currentVersion)
		if err != nil {
			if errors.Is(err, huh.ErrUserAborted) {
				return nil
			}
			return err
		}

		// バージョンが変更された場合は保存
		if newVersion != currentVersion {
			manifest["version"] = newVersion
			if err := saveBuildManifest(cwd, manifest); err != nil {
				return fmt.Errorf("manifest.json の保存に失敗しました: %w", err)
			}
			// package.json のバージョンも同期
			if err := updatePackageJSONVersion(cwd, newVersion); err != nil {
				return fmt.Errorf("package.json の更新に失敗しました: %w", err)
			}
			ui.Success(fmt.Sprintf("バージョンを更新: %s → %s", currentVersion, newVersion))
			fmt.Println()
		}
	}

	// モード表示
	if isPre {
		ui.Info("プレビルドを開始... (minifyなし, console残す, 名前に[開発]付与)")
	} else {
		ui.Info("本番ビルドを開始...")
	}
	fmt.Println()

	opts := &plugin.BuildOptions{
		Mode:          buildMode,
		Minify:        !isPre,
		RemoveConsole: !isPre,
	}

	var zipPath string
	err = ui.SpinnerWithResult("バンドル中...", func() error {
		var buildErr error
		zipPath, buildErr = plugin.Build(cwd, opts)
		return buildErr
	})
	if err != nil {
		fmt.Println()
		return err
	}

	// 結果を表示
	fmt.Println()
	if isPre {
		ui.Success("プレビルド完了!")
	} else {
		ui.Success("ビルド完了!")
	}

	fmt.Printf("\nPlugin ID:\n")
	if isPre {
		fmt.Printf("  %s\n", ui.InfoStyle.Render(meta.PluginIDs.Dev))
	} else {
		fmt.Printf("  %s\n", ui.InfoStyle.Render(meta.PluginIDs.Prod))
	}

	fmt.Printf("\n出力ファイル:\n")
	fmt.Printf("  %s\n\n", ui.InfoStyle.Render(zipPath))

	return nil
}

func loadBuildManifest(projectDir string) (map[string]interface{}, error) {
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

func saveBuildManifest(projectDir string, manifest map[string]interface{}) error {
	manifestPath := filepath.Join(config.GetConfigDir(projectDir), "manifest.json")
	data := config.MarshalManifestJSON(manifest)
	return os.WriteFile(manifestPath, []byte(data), 0644)
}

func updatePackageJSONVersion(projectDir string, newVersion string) error {
	pkgPath := filepath.Join(projectDir, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		// package.json がない場合はスキップ
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	// JSONをパースせず正規表現でバージョンのみ置換（プロパティ順序を維持）
	re := regexp.MustCompile(`("version"\s*:\s*)"[^"]*"`)
	replaced := re.ReplaceAll(data, []byte(`${1}"`+newVersion+`"`))

	return os.WriteFile(pkgPath, replaced, 0644)
}

func askVersion(currentVersion string) (string, error) {
	cyanStyle := lipgloss.NewStyle().Foreground(ui.ColorCyan)

	// バージョンをパース
	parts := strings.Split(currentVersion, ".")
	major, minor, patch := 0, 0, 0
	if len(parts) >= 1 {
		major, _ = strconv.Atoi(parts[0])
	}
	if len(parts) >= 2 {
		minor, _ = strconv.Atoi(parts[1])
	}
	if len(parts) >= 3 {
		patch, _ = strconv.Atoi(parts[2])
	}

	// バージョン選択肢を作成
	patchVersion := fmt.Sprintf("%d.%d.%d", major, minor, patch+1)
	minorVersion := fmt.Sprintf("%d.%d.%d", major, minor+1, 0)
	majorVersion := fmt.Sprintf("%d.%d.%d", major+1, 0, 0)

	fmt.Printf("現在のバージョン: %s\n\n", cyanStyle.Render(currentVersion))

	type versionChoice struct {
		label   string
		version string
	}

	choices := []versionChoice{
		{fmt.Sprintf("現在のまま (%s)", currentVersion), currentVersion},
		{fmt.Sprintf("パッチ更新 (%s)", patchVersion), patchVersion},
		{fmt.Sprintf("マイナー更新 (%s)", minorVersion), minorVersion},
		{fmt.Sprintf("メジャー更新 (%s)", majorVersion), majorVersion},
		{"カスタム入力", "_custom_"},
	}

	options := make([]huh.Option[string], len(choices))
	for i, c := range choices {
		options[i] = huh.NewOption(c.label, c.version)
	}

	var answer string
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("バージョンを選択").
				Options(options...).
				Value(&answer),
		),
	).WithTheme(huh.ThemeCatppuccin()).Run()
	if err != nil {
		return "", err
	}

	if answer == "_custom_" {
		var customVersion string
		err := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("バージョンを入力").
					Value(&customVersion).
					Placeholder(currentVersion).
					Validate(func(s string) error {
						if s == "" {
							return fmt.Errorf("入力必須です")
						}
						return nil
					}),
			),
		).WithTheme(huh.ThemeCatppuccin()).Run()
		if err != nil {
			return "", err
		}
		if customVersion == "" {
			customVersion = currentVersion
		}
		return customVersion, nil
	}

	return answer, nil
}

func askBuildMode() (string, error) {
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

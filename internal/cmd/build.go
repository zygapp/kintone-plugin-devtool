package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
	"github.com/kintone/kpdev/internal/config"
	"github.com/kintone/kpdev/internal/generator"
	"github.com/kintone/kpdev/internal/plugin"
	"github.com/spf13/cobra"
)

var (
	flagNoMinify      bool
	flagRemoveConsole bool
	flagSkipVersion   bool
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "本番用プラグインをビルド",
	Long:  `本番用プラグインZIPを生成します。`,
	RunE:  runBuild,
}

func init() {
	rootCmd.AddCommand(buildCmd)

	buildCmd.Flags().BoolVar(&flagNoMinify, "no-minify", false, "minify を無効化")
	buildCmd.Flags().BoolVar(&flagRemoveConsole, "remove-console", true, "console.* を削除")
	buildCmd.Flags().BoolVar(&flagSkipVersion, "skip-version", false, "バージョン確認をスキップ")
}

func runBuild(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

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
			return err
		}

		// バージョンが変更された場合は保存
		if newVersion != currentVersion {
			manifest["version"] = newVersion
			if err := saveBuildManifest(cwd, manifest); err != nil {
				return fmt.Errorf("manifest.json の保存に失敗しました: %w", err)
			}
			fmt.Printf("%s バージョンを更新: %s → %s\n\n", green("✓"), currentVersion, newVersion)
		}
	}

	fmt.Printf("%s ビルドを開始...\n\n", cyan("→"))

	fmt.Printf("○ バンドル中...")

	opts := &plugin.BuildOptions{
		Minify:        !flagNoMinify,
		RemoveConsole: flagRemoveConsole,
	}

	zipPath, err := plugin.Build(cwd, opts)
	if err != nil {
		fmt.Println()
		return err
	}

	fmt.Printf(" %s\n", green("✓"))

	// 結果を表示
	fmt.Printf("\n%s ビルド完了!\n", green("✓"))

	fmt.Printf("\nPlugin ID:\n")
	fmt.Printf("  %s\n", cyan(meta.PluginIDs.Prod))

	fmt.Printf("\n出力ファイル:\n")
	fmt.Printf("  %s\n", cyan(zipPath))

	// dist/plugin/ の内容を表示
	pluginDir := filepath.Join(cwd, "dist", "plugin")
	fmt.Printf("\nプラグイン構造:\n")
	filepath.Walk(pluginDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		relPath, _ := filepath.Rel(pluginDir, path)
		if relPath == "." {
			return nil
		}
		indent := ""
		for i := 0; i < len(filepath.SplitList(relPath))-1; i++ {
			indent += "  "
		}
		if info.IsDir() {
			fmt.Printf("  %s%s/\n", indent, info.Name())
		} else {
			fmt.Printf("  %s%s\n", indent, info.Name())
		}
		return nil
	})

	fmt.Println()

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
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(manifestPath, data, 0644)
}

func askVersion(currentVersion string) (string, error) {
	cyan := color.New(color.FgCyan).SprintFunc()

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

	options := []string{
		fmt.Sprintf("現在のまま (%s)", currentVersion),
		fmt.Sprintf("パッチ更新 (%s)", patchVersion),
		fmt.Sprintf("マイナー更新 (%s)", minorVersion),
		fmt.Sprintf("メジャー更新 (%s)", majorVersion),
		"カスタム入力",
	}

	fmt.Printf("現在のバージョン: %s\n\n", cyan(currentVersion))

	var answer string
	prompt := &survey.Select{
		Message: "バージョンを選択:",
		Options: options,
		Default: options[0],
	}
	if err := survey.AskOne(prompt, &answer); err != nil {
		return "", err
	}

	switch answer {
	case options[0]:
		return currentVersion, nil
	case options[1]:
		return patchVersion, nil
	case options[2]:
		return minorVersion, nil
	case options[3]:
		return majorVersion, nil
	default:
		// カスタム入力
		var customVersion string
		inputPrompt := &survey.Input{
			Message: "バージョンを入力:",
			Default: currentVersion,
		}
		if err := survey.AskOne(inputPrompt, &customVersion, survey.WithValidator(survey.Required)); err != nil {
			return "", err
		}
		return customVersion, nil
	}
}

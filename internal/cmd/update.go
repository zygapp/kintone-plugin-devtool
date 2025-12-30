package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/kintone/kpdev/internal/config"
	"github.com/kintone/kpdev/internal/ui"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "開発依存パッケージを更新",
	Long:  `Vite やフレームワークプラグインを最新バージョンに更新します。`,
	RunE:  runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

// 更新対象のパッケージパターン
var updatePackages = []string{
	"vite",
	"@vitejs/plugin-react",
	"@vitejs/plugin-vue",
	"@sveltejs/vite-plugin-svelte",
	"react",
	"react-dom",
	"vue",
	"svelte",
	"typescript",
	"@types/react",
	"@types/react-dom",
	"@kintone/rest-api-client",
	"@kintone/dts-gen",
}

func runUpdate(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	// 設定を読み込み
	cfg, err := config.Load(cwd)
	if err != nil {
		return fmt.Errorf("設定ファイルが見つかりません。先に kpdev init を実行してください: %w", err)
	}

	// パッケージマネージャーを取得
	pm := cfg.GetPackageManager(cwd)
	fmt.Printf("%s パッケージマネージャー: %s\n\n", cyan("→"), pm)

	// package.json から実際にインストールされているパッケージを抽出
	installedPkgs, err := getInstalledUpdatePackages(cwd)
	if err != nil {
		return fmt.Errorf("package.json の読み込みに失敗しました: %w", err)
	}

	if len(installedPkgs) == 0 {
		fmt.Printf("%s 更新対象のパッケージが見つかりませんでした\n", cyan("→"))
		return nil
	}

	fmt.Printf("%s 以下のパッケージを更新します:\n", cyan("→"))
	for _, pkg := range installedPkgs {
		fmt.Printf("  - %s\n", pkg)
	}
	fmt.Println()

	// 更新コマンドを構築
	var updateArgs []string
	switch pm {
	case "npm":
		updateArgs = append([]string{"update"}, installedPkgs...)
	case "pnpm":
		updateArgs = append([]string{"update"}, installedPkgs...)
	case "yarn":
		updateArgs = append([]string{"upgrade"}, installedPkgs...)
	case "bun":
		updateArgs = append([]string{"update"}, installedPkgs...)
	default:
		updateArgs = append([]string{"update"}, installedPkgs...)
	}

	fmt.Println()
	if err := ui.RunCommandWithSpinner("パッケージを更新中...", pm, updateArgs, cwd); err != nil {
		return fmt.Errorf("更新エラー: %w", err)
	}

	fmt.Printf("\n%s パッケージの更新が完了しました\n", green("✓"))

	return nil
}

func getInstalledUpdatePackages(projectDir string) ([]string, error) {
	pkgPath := projectDir + "/package.json"
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return nil, err
	}

	// パッケージ名をチェック
	content := string(data)
	var installed []string

	for _, pkg := range updatePackages {
		// パッケージがdependenciesまたはdevDependenciesに含まれているかチェック
		if contains(content, `"`+pkg+`"`) {
			installed = append(installed, pkg)
		}
	}

	return installed, nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

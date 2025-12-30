package cmd

import (
	"fmt"

	"github.com/kintone/kpdev/internal/ui"
	"github.com/spf13/cobra"
)

// version はビルド時に -ldflags で注入される
var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "kpdev",
	Short: "kintone プラグイン開発ツール",
	Long: `kpdev は kintone プラグイン開発を Vite + HMR で行うための CLI ツールです。

主要コマンド:
  init    プロジェクトを初期化
  dev     開発サーバーを起動
  build   本番用プラグインをビルド
  deploy  プラグインをデプロイ`,
	Version: version,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.SetVersionTemplate(fmt.Sprintf("kpdev version %s\n", version))
	rootCmd.PersistentFlags().BoolVarP(&ui.Quiet, "quiet", "q", false, "出力を最小限に抑制（CI/CD向け）")
}

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kintone/kpdev/internal/config"
	"github.com/kintone/kpdev/internal/ui"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "環境診断を実行",
	Long:  `開発環境の状態を診断し、問題があれば報告します。`,
	RunE:  runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

type checkResult struct {
	name    string
	status  string // "ok", "warn", "error"
	message string
}

func runDoctor(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	fmt.Println()
	ui.Title("環境診断")
	fmt.Println()

	results := []checkResult{}

	// 1. Node.js バージョン確認
	results = append(results, checkNodeVersion())

	// 2. npm/pnpm/yarn/bun バージョン確認
	results = append(results, checkPackageManager(cwd))

	// 3. 証明書の確認
	results = append(results, checkCertificates(cwd))

	// 4. 設定ファイルの整合性確認
	results = append(results, checkConfigFiles(cwd)...)

	// 結果を表示
	fmt.Println()
	hasError := false
	hasWarn := false

	for _, r := range results {
		switch r.status {
		case "ok":
			fmt.Printf("  %s %s: %s\n", ui.SuccessStyle.Render(ui.IconSuccess), r.name, r.message)
		case "warn":
			fmt.Printf("  %s %s: %s\n", ui.WarnStyle.Render(ui.IconWarn), r.name, r.message)
			hasWarn = true
		case "error":
			fmt.Printf("  %s %s: %s\n", ui.ErrorStyle.Render(ui.IconError), r.name, r.message)
			hasError = true
		}
	}

	fmt.Println()

	if hasError {
		ui.Error("問題が検出されました。上記の内容を確認してください。")
	} else if hasWarn {
		ui.Warn("警告があります。必要に応じて対応してください。")
	} else {
		ui.Success("すべてのチェックに合格しました!")
	}

	return nil
}

func checkNodeVersion() checkResult {
	cmd := exec.Command("node", "--version")
	output, err := cmd.Output()
	if err != nil {
		return checkResult{
			name:    "Node.js",
			status:  "error",
			message: "インストールされていません",
		}
	}

	version := strings.TrimSpace(string(output))
	// v18以上を推奨
	if strings.HasPrefix(version, "v18") || strings.HasPrefix(version, "v19") ||
		strings.HasPrefix(version, "v20") || strings.HasPrefix(version, "v21") ||
		strings.HasPrefix(version, "v22") || strings.HasPrefix(version, "v23") {
		return checkResult{
			name:    "Node.js",
			status:  "ok",
			message: version,
		}
	}

	return checkResult{
		name:    "Node.js",
		status:  "warn",
		message: fmt.Sprintf("%s (v18以上を推奨)", version),
	}
}

func checkPackageManager(projectDir string) checkResult {
	pm := config.DetectPackageManager(projectDir)

	var cmd *exec.Cmd
	switch pm {
	case "pnpm":
		cmd = exec.Command("pnpm", "--version")
	case "yarn":
		cmd = exec.Command("yarn", "--version")
	case "bun":
		cmd = exec.Command("bun", "--version")
	default:
		cmd = exec.Command("npm", "--version")
		pm = "npm"
	}

	output, err := cmd.Output()
	if err != nil {
		return checkResult{
			name:    "パッケージマネージャー",
			status:  "error",
			message: fmt.Sprintf("%s がインストールされていません", pm),
		}
	}

	version := strings.TrimSpace(string(output))
	return checkResult{
		name:    "パッケージマネージャー",
		status:  "ok",
		message: fmt.Sprintf("%s %s", pm, version),
	}
}

func checkCertificates(projectDir string) checkResult {
	certsDir := filepath.Join(config.GetConfigDir(projectDir), "certs")

	certFile := filepath.Join(certsDir, "localhost.pem")
	keyFile := filepath.Join(certsDir, "localhost-key.pem")

	certExists := fileExists(certFile)
	keyExists := fileExists(keyFile)

	if !certExists && !keyExists {
		return checkResult{
			name:    "HTTPS証明書",
			status:  "warn",
			message: "未生成（kpdev dev 初回実行時に生成されます）",
		}
	}

	if certExists && keyExists {
		return checkResult{
			name:    "HTTPS証明書",
			status:  "ok",
			message: "生成済み",
		}
	}

	return checkResult{
		name:    "HTTPS証明書",
		status:  "error",
		message: "証明書ファイルが不完全です",
	}
}

func checkConfigFiles(projectDir string) []checkResult {
	results := []checkResult{}
	configDir := config.GetConfigDir(projectDir)

	// config.json
	configPath := filepath.Join(configDir, "config.json")
	if fileExists(configPath) {
		cfg, err := config.Load(projectDir)
		if err != nil {
			results = append(results, checkResult{
				name:    "config.json",
				status:  "error",
				message: "読み込みエラー: " + err.Error(),
			})
		} else {
			// 必須フィールドの確認
			issues := []string{}
			if cfg.Kintone.Dev.Domain == "" {
				issues = append(issues, "開発ドメイン未設定")
			}
			if cfg.Dev.Entry.Main == "" {
				issues = append(issues, "mainエントリー未設定")
			}

			if len(issues) > 0 {
				results = append(results, checkResult{
					name:    "config.json",
					status:  "warn",
					message: strings.Join(issues, ", "),
				})
			} else {
				results = append(results, checkResult{
					name:    "config.json",
					status:  "ok",
					message: "正常",
				})
			}
		}
	} else {
		results = append(results, checkResult{
			name:    "config.json",
			status:  "warn",
			message: "未作成（kpdev init を実行してください）",
		})
	}

	// manifest.json
	manifestPath := filepath.Join(configDir, "manifest.json")
	if fileExists(manifestPath) {
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			results = append(results, checkResult{
				name:    "manifest.json",
				status:  "error",
				message: "読み込みエラー",
			})
		} else {
			var manifest map[string]interface{}
			if err := json.Unmarshal(data, &manifest); err != nil {
				results = append(results, checkResult{
					name:    "manifest.json",
					status:  "error",
					message: "JSONパースエラー",
				})
			} else {
				// 必須フィールドの確認
				issues := []string{}
				if manifest["version"] == nil {
					issues = append(issues, "version未設定")
				}
				if manifest["name"] == nil {
					issues = append(issues, "name未設定")
				}

				if len(issues) > 0 {
					results = append(results, checkResult{
						name:    "manifest.json",
						status:  "warn",
						message: strings.Join(issues, ", "),
					})
				} else {
					results = append(results, checkResult{
						name:    "manifest.json",
						status:  "ok",
						message: fmt.Sprintf("v%v", manifest["version"]),
					})
				}
			}
		}
	} else {
		// config.jsonがあればmanifest.jsonも必要
		if fileExists(configPath) {
			results = append(results, checkResult{
				name:    "manifest.json",
				status:  "error",
				message: "未作成",
			})
		}
	}

	// 秘密鍵
	devKeyPath := filepath.Join(configDir, "keys", "private.dev.ppk")
	prodKeyPath := filepath.Join(configDir, "keys", "private.prod.ppk")

	if fileExists(devKeyPath) && fileExists(prodKeyPath) {
		results = append(results, checkResult{
			name:    "秘密鍵",
			status:  "ok",
			message: "開発用・本番用ともに存在",
		})
	} else if fileExists(devKeyPath) || fileExists(prodKeyPath) {
		missing := "開発用"
		if fileExists(devKeyPath) {
			missing = "本番用"
		}
		results = append(results, checkResult{
			name:    "秘密鍵",
			status:  "warn",
			message: fmt.Sprintf("%sが未生成", missing),
		})
	} else if fileExists(configPath) {
		results = append(results, checkResult{
			name:    "秘密鍵",
			status:  "warn",
			message: "未生成（kpdev dev または build 時に生成されます）",
		})
	}

	return results
}


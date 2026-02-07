package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"github.com/kintone/kpdev/internal/config"
	"github.com/kintone/kpdev/internal/generator"
)

type BuildOptions struct {
	Mode          string // "prod" or "pre"
	Minify        bool
	RemoveConsole bool
}

// Build は本番用プラグインをビルドする
func Build(projectDir string, opts *BuildOptions) (string, error) {
	distDir := filepath.Join(projectDir, "dist")
	pluginDir := filepath.Join(distDir, "plugin")

	// dist/ をクリーン
	if err := os.RemoveAll(distDir); err != nil {
		return "", err
	}
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		return "", err
	}

	// 設定を読み込み
	cfg, err := config.Load(projectDir)
	if err != nil {
		return "", fmt.Errorf("設定読み込みエラー: %w", err)
	}

	// Vite でビルド
	if err := runViteBuild(projectDir, opts); err != nil {
		return "", fmt.Errorf("Viteビルドエラー: %w", err)
	}

	// ビルド成果物を整理
	if err := organizeDistFiles(projectDir, pluginDir, cfg); err != nil {
		return "", fmt.Errorf("ファイル整理エラー: %w", err)
	}

	// 一時ビルドファイルを削除
	cleanupTempFiles(distDir)

	// manifest.json を生成
	if err := generateProdManifest(projectDir, pluginDir, cfg, opts); err != nil {
		return "", fmt.Errorf("manifest生成エラー: %w", err)
	}

	// icon.png をコピー（存在しない場合は生成）
	srcIcon := filepath.Join(projectDir, "icon.png")
	if _, err := os.Stat(srcIcon); os.IsNotExist(err) {
		if err := generator.GenerateIcon(projectDir); err != nil {
			return "", fmt.Errorf("アイコン生成エラー: %w", err)
		}
	}
	dstIcon := filepath.Join(pluginDir, "icon.png")
	if err := copyFile(srcIcon, dstIcon); err != nil {
		return "", fmt.Errorf("iconコピーエラー: %w", err)
	}

	// config.html を生成
	if err := generateProdConfigHTML(projectDir, pluginDir); err != nil {
		return "", fmt.Errorf("config.html生成エラー: %w", err)
	}

	// プラグインZIPを作成（署名付き）
	version := getManifestVersion(projectDir)
	nameEn := getManifestNameEn(projectDir)
	safeName := sanitizeFilename(nameEn)
	modeLabel := "prod"
	if opts.Mode == "pre" {
		modeLabel = "pre"
	}
	zipPath := filepath.Join(distDir, fmt.Sprintf("%s-%s-v%s.zip", safeName, modeLabel, version))

	var keyPath string
	if opts.Mode == "pre" {
		keyPath = generator.GetDevKeyPath(projectDir)
	} else {
		keyPath = generator.GetProdKeyPath(projectDir)
	}
	privateKey, err := generator.LoadPrivateKey(keyPath)
	if err != nil {
		return "", fmt.Errorf("秘密鍵読み込みエラー: %w", err)
	}

	if err := createPluginZip(pluginDir, zipPath, privateKey); err != nil {
		return "", fmt.Errorf("ZIP作成エラー: %w", err)
	}

	return zipPath, nil
}

func runViteBuild(projectDir string, opts *BuildOptions) error {
	viteConfigPath := filepath.Join(config.GetConfigDir(projectDir), "vite.config.ts")

	// main をビルド
	if err := runSingleViteBuild(projectDir, viteConfigPath, "main", opts); err != nil {
		return err
	}

	// config をビルド
	return runSingleViteBuild(projectDir, viteConfigPath, "config", opts)
}

func runSingleViteBuild(projectDir, viteConfigPath, entry string, opts *BuildOptions) error {
	args := []string{"vite", "build", "--config", viteConfigPath}

	if !opts.Minify {
		args = append(args, "--minify", "false")
	}

	cmd := exec.Command("npx", args...)
	cmd.Dir = projectDir
	cmd.Env = append(os.Environ(), "VITE_BUILD_ENTRY="+entry)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w\n%s", err, string(output))
	}

	return nil
}

func organizeDistFiles(projectDir, pluginDir string, cfg *config.Config) error {
	distDir := filepath.Join(projectDir, "dist")

	// js/ ディレクトリ作成
	jsDir := filepath.Join(pluginDir, "js")
	if err := os.MkdirAll(jsDir, 0755); err != nil {
		return err
	}

	// css/ ディレクトリ作成
	cssDir := filepath.Join(pluginDir, "css")
	if err := os.MkdirAll(cssDir, 0755); err != nil {
		return err
	}

	// html/ ディレクトリ作成
	htmlDir := filepath.Join(pluginDir, "html")
	if err := os.MkdirAll(htmlDir, 0755); err != nil {
		return err
	}

	// main.js → desktop.js, mobile.js にコピー
	mainJS := filepath.Join(distDir, "main.js")
	if _, err := os.Stat(mainJS); err == nil {
		if cfg.Targets.Desktop {
			if err := copyFile(mainJS, filepath.Join(jsDir, "desktop.js")); err != nil {
				return err
			}
		}
		if cfg.Targets.Mobile {
			if err := copyFile(mainJS, filepath.Join(jsDir, "mobile.js")); err != nil {
				return err
			}
		}
	}

	// config.js をコピー
	configJS := filepath.Join(distDir, "config.js")
	if _, err := os.Stat(configJS); err == nil {
		if err := copyFile(configJS, filepath.Join(jsDir, "config.js")); err != nil {
			return err
		}
	}

	// CSS ファイルをコピー（存在する場合）
	mainCSS := filepath.Join(distDir, "main.css")
	if _, err := os.Stat(mainCSS); err == nil {
		if cfg.Targets.Desktop {
			if err := copyFile(mainCSS, filepath.Join(cssDir, "desktop.css")); err != nil {
				return err
			}
		}
		if cfg.Targets.Mobile {
			if err := copyFile(mainCSS, filepath.Join(cssDir, "mobile.css")); err != nil {
				return err
			}
		}
	}

	configCSS := filepath.Join(distDir, "config.css")
	if _, err := os.Stat(configCSS); err == nil {
		if err := copyFile(configCSS, filepath.Join(cssDir, "config.css")); err != nil {
			return err
		}
	}

	return nil
}

func generateProdManifest(projectDir, pluginDir string, cfg *config.Config, opts *BuildOptions) error {
	// .kpdev/manifest.json を読み込み
	srcManifest := filepath.Join(config.GetConfigDir(projectDir), "manifest.json")
	data, err := os.ReadFile(srcManifest)
	if err != nil {
		return err
	}

	var manifest map[string]interface{}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return err
	}

	// preモードの場合、名前に[開発]を付与
	if opts.Mode == "pre" {
		if name, ok := manifest["name"].(map[string]interface{}); ok {
			if ja, ok := name["ja"].(string); ok {
				name["ja"] = "[開発] " + ja
			}
			if en, ok := name["en"].(string); ok {
				name["en"] = "[DEV] " + en
			}
		}
	}

	// パスを更新
	if cfg.Targets.Desktop {
		manifest["desktop"] = map[string]interface{}{
			"js": []string{"js/desktop.js"},
		}
		// CSS が存在する場合のみ追加
		if _, err := os.Stat(filepath.Join(pluginDir, "css", "desktop.css")); err == nil {
			desktop := manifest["desktop"].(map[string]interface{})
			desktop["css"] = []string{"css/desktop.css"}
		}
	} else {
		delete(manifest, "desktop")
	}

	if cfg.Targets.Mobile {
		manifest["mobile"] = map[string]interface{}{
			"js": []string{"js/mobile.js"},
		}
		// CSS が存在する場合のみ追加
		if _, err := os.Stat(filepath.Join(pluginDir, "css", "mobile.css")); err == nil {
			mobile := manifest["mobile"].(map[string]interface{})
			mobile["css"] = []string{"css/mobile.css"}
		}
	} else {
		delete(manifest, "mobile")
	}

	// 既存のrequired_paramsを保持（config内を優先、トップレベルもフォールバック）
	var existingRequiredParams interface{}
	if existingConfig, ok := manifest["config"].(map[string]interface{}); ok {
		existingRequiredParams = existingConfig["required_params"]
	}
	if existingRequiredParams == nil {
		existingRequiredParams = manifest["required_params"]
	}
	delete(manifest, "required_params")

	configMap := map[string]interface{}{
		"html": "html/config.html",
		"js":   []string{"js/config.js"},
	}
	// CSS が存在する場合のみ追加
	if _, err := os.Stat(filepath.Join(pluginDir, "css", "config.css")); err == nil {
		configMap["css"] = []string{"css/config.css"}
	}
	// required_paramsをconfig内に復元
	if existingRequiredParams != nil {
		configMap["required_params"] = existingRequiredParams
	}
	manifest["config"] = configMap

	// 標準順序で保存
	outData := config.MarshalManifestJSON(manifest)

	return os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(outData), 0644)
}

func generateProdConfigHTML(projectDir, pluginDir string) error {
	htmlDir := filepath.Join(pluginDir, "html")
	if err := os.MkdirAll(htmlDir, 0755); err != nil {
		return err
	}

	// src/config/index.html からコピー
	srcPath := filepath.Join(projectDir, "src", "config", "index.html")
	content, err := os.ReadFile(srcPath)
	if err != nil {
		// ファイルが存在しない場合はデフォルト
		content = []byte("<div id=\"config-root\"></div>\n")
	}

	return os.WriteFile(filepath.Join(htmlDir, "config.html"), content, 0644)
}

func getManifestVersion(projectDir string) string {
	manifestPath := filepath.Join(config.GetConfigDir(projectDir), "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return "1.0.0"
	}

	var manifest struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return "1.0.0"
	}

	if manifest.Version == "" {
		return "1.0.0"
	}

	return manifest.Version
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

func cleanupTempFiles(distDir string) {
	// Vite が出力した一時ファイルを削除
	tempFiles := []string{
		"main.js",
		"main.css",
		"config.js",
		"config.css",
	}

	for _, file := range tempFiles {
		path := filepath.Join(distDir, file)
		os.Remove(path) // エラーは無視（存在しない場合がある）
	}
}

// getManifestNameEn は manifest.json から英語名を取得する
func getManifestNameEn(projectDir string) string {
	manifestPath := filepath.Join(config.GetConfigDir(projectDir), "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return "plugin"
	}

	var manifest struct {
		Name struct {
			En string `json:"en"`
		} `json:"name"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return "plugin"
	}

	if manifest.Name.En == "" {
		return "plugin"
	}

	return manifest.Name.En
}

// sanitizeFilename は英数字以外をアンダースコアに変換する
func sanitizeFilename(name string) string {
	// 英数字以外をアンダースコアに置換
	re := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	sanitized := re.ReplaceAllString(name, "_")

	// 先頭・末尾のアンダースコアを削除
	sanitized = regexp.MustCompile(`^_+|_+$`).ReplaceAllString(sanitized, "")

	if sanitized == "" {
		return "plugin"
	}

	return sanitized
}

package generator

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kintone/kpdev/internal/config"
	"github.com/kintone/kpdev/internal/prompt"
)

const (
	LoaderSchemaVersion = 1
	DevOrigin           = "https://localhost:3000"
)

type LoaderMeta struct {
	SchemaVersion int       `json:"schemaVersion"`
	KpdevVersion  string    `json:"kpdevVersion"`
	GeneratedAt   time.Time `json:"generatedAt"`
	Dev           struct {
		Origin string `json:"origin"`
	} `json:"dev"`
	Project struct {
		Name      string `json:"name"`
		Framework string `json:"framework"`
		Language  string `json:"language"`
	} `json:"project"`
	Targets struct {
		Desktop bool `json:"desktop"`
		Mobile  bool `json:"mobile"`
	} `json:"targets"`
	Entries struct {
		Main   string `json:"main"`
		Config string `json:"config"`
	} `json:"entries"`
	Kintone struct {
		Domain string `json:"domain"`
	} `json:"kintone"`
	PluginIDs struct {
		Dev  string `json:"dev"`
		Prod string `json:"prod"`
	} `json:"pluginIds"`
	Files struct {
		LoaderZipPath  string `json:"loaderZipPath"`
		LoaderZipSHA256 string `json:"loaderZipSha256"`
		DevKeyPath     string `json:"devKeyPath"`
		ProdKeyPath    string `json:"prodKeyPath"`
		CertKeyPath    string `json:"certKeyPath"`
		CertCertPath   string `json:"certCertPath"`
	} `json:"files"`
}

func GenerateLoader(projectDir string, answers *prompt.InitAnswers, version string) error {
	managedDir := filepath.Join(config.GetConfigDir(projectDir), "managed")
	devPluginDir := filepath.Join(managedDir, "dev-plugin")

	if err := os.MkdirAll(devPluginDir, 0755); err != nil {
		return err
	}

	// プラグインIDを生成
	devKeyPath := GetDevKeyPath(projectDir)
	devKey, err := LoadPrivateKey(devKeyPath)
	if err != nil {
		return fmt.Errorf("開発用秘密鍵の読み込みに失敗: %w", err)
	}
	devPluginID, err := GeneratePluginID(devKey)
	if err != nil {
		return fmt.Errorf("プラグインIDの生成に失敗: %w", err)
	}

	prodKeyPath := GetProdKeyPath(projectDir)
	prodKey, err := LoadPrivateKey(prodKeyPath)
	if err != nil {
		return fmt.Errorf("本番用秘密鍵の読み込みに失敗: %w", err)
	}
	prodPluginID, err := GeneratePluginID(prodKey)
	if err != nil {
		return fmt.Errorf("プラグインIDの生成に失敗: %w", err)
	}

	// manifest.json
	if err := generateDevManifest(devPluginDir, answers); err != nil {
		return err
	}

	// desktop.js
	if answers.TargetDesktop {
		if err := generateLoaderJS(devPluginDir, "desktop"); err != nil {
			return err
		}
	}

	// mobile.js
	if answers.TargetMobile {
		if err := generateLoaderJS(devPluginDir, "mobile"); err != nil {
			return err
		}
	}

	// config-loader.js
	if err := generateConfigLoaderJS(devPluginDir); err != nil {
		return err
	}

	// config.html (src/config/index.html からコピー)
	if err := generateConfigHTML(projectDir, devPluginDir); err != nil {
		return err
	}

	// icon.png をコピー（存在しない場合は生成）
	srcIcon := filepath.Join(projectDir, "icon.png")
	if _, err := os.Stat(srcIcon); os.IsNotExist(err) {
		if err := GenerateIcon(projectDir); err != nil {
			return fmt.Errorf("アイコン生成に失敗: %w", err)
		}
	}
	dstIcon := filepath.Join(devPluginDir, "icon.png")
	if err := copyFile(srcIcon, dstIcon); err != nil {
		return err
	}

	// loader.meta.json
	meta := &LoaderMeta{
		SchemaVersion: LoaderSchemaVersion,
		KpdevVersion:  version,
		GeneratedAt:   time.Now(),
	}
	meta.Dev.Origin = DevOrigin
	meta.Project.Name = answers.ProjectName
	meta.Project.Framework = string(answers.Framework)
	meta.Project.Language = string(answers.Language)
	meta.Targets.Desktop = answers.TargetDesktop
	meta.Targets.Mobile = answers.TargetMobile
	meta.Entries.Main = GetEntryPath(answers.Framework, answers.Language, "main")
	meta.Entries.Config = GetEntryPath(answers.Framework, answers.Language, "config")
	meta.Kintone.Domain = answers.Domain
	meta.PluginIDs.Dev = devPluginID
	meta.PluginIDs.Prod = prodPluginID
	meta.Files.LoaderZipPath = ".kpdev/managed/dev-plugin.zip"
	meta.Files.DevKeyPath = ".kpdev/keys/" + DevKeyFile
	meta.Files.ProdKeyPath = ".kpdev/keys/" + ProdKeyFile
	meta.Files.CertKeyPath = ".kpdev/certs/localhost-key.pem"
	meta.Files.CertCertPath = ".kpdev/certs/localhost.pem"

	metaData, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}

	metaPath := filepath.Join(managedDir, "loader.meta.json")
	return os.WriteFile(metaPath, metaData, 0644)
}

func generateDevManifest(dir string, answers *prompt.InitAnswers) error {
	// プラグイン名（デフォルトはプロジェクト名）
	nameJa := answers.PluginNameJa
	if nameJa == "" {
		nameJa = answers.ProjectName
	}
	nameEn := answers.PluginNameEn
	if nameEn == "" {
		nameEn = answers.ProjectName
	}

	manifest := map[string]interface{}{
		"manifest_version": 1,
		"version":          1,
		"type":             "APP",
		"name": map[string]string{
			"ja": "[DEV] " + nameJa,
			"en": "[DEV] " + nameEn,
		},
		"description": map[string]string{
			"ja": "kpdev開発用ローダープラグイン",
			"en": "kpdev development loader plugin",
		},
		"icon": "icon.png",
	}

	if answers.TargetDesktop {
		manifest["desktop"] = map[string]interface{}{
			"js": []string{"desktop.js"},
		}
	}

	if answers.TargetMobile {
		manifest["mobile"] = map[string]interface{}{
			"js": []string{"mobile.js"},
		}
	}

	manifest["config"] = map[string]interface{}{
		"html": "config.html",
		"js":   []string{"config-loader.js"},
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, "manifest.json"), data, 0644)
}

func generateLoaderJS(dir string, target string) error {
	// main-loader.js を生成
	mainLoaderContent := fmt.Sprintf(`(() => {
  const origin = "%s";
  const t = Date.now();

  const xhr = new XMLHttpRequest();
  xhr.open("GET", origin + "/main.js?t=" + t, false);
  xhr.send();
  if (xhr.status === 200) {
    eval(xhr.responseText);
  }

  // Vite HMR client をscriptタグで読み込み
  const script = document.createElement("script");
  script.type = "module";
  script.src = origin + "/@vite/client";
  document.head.appendChild(script);
})();
`, DevOrigin)

	if err := os.WriteFile(filepath.Join(dir, "main-loader.js"), []byte(mainLoaderContent), 0644); err != nil {
		return err
	}

	// desktop.js / mobile.js は main-loader.js と同じ内容
	return os.WriteFile(filepath.Join(dir, target+".js"), []byte(mainLoaderContent), 0644)
}

func generateConfigLoaderJS(dir string) error {
	content := fmt.Sprintf(`(() => {
  const origin = "%s";
  const t = Date.now();

  // JSをフェッチして実行
  const xhr = new XMLHttpRequest();
  xhr.open("GET", origin + "/config.js?t=" + t, false);
  xhr.send();
  if (xhr.status === 200) {
    eval(xhr.responseText);
  }

  // Vite HMR client をscriptタグで読み込み
  const script = document.createElement("script");
  script.type = "module";
  script.src = origin + "/@vite/client";
  document.head.appendChild(script);
})();
`, DevOrigin)

	return os.WriteFile(filepath.Join(dir, "config-loader.js"), []byte(content), 0644)
}

func generateConfigHTML(projectDir, devPluginDir string) error {
	// src/config/index.html からコピー
	srcPath := filepath.Join(projectDir, "src", "config", "index.html")
	content, err := os.ReadFile(srcPath)
	if err != nil {
		// ファイルが存在しない場合はデフォルト
		content = []byte("<div id=\"config-root\"></div>\n")
	}
	return os.WriteFile(filepath.Join(devPluginDir, "config.html"), content, 0644)
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

func LoadLoaderMeta(projectDir string) (*LoaderMeta, error) {
	metaPath := filepath.Join(config.GetConfigDir(projectDir), "managed", "loader.meta.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, err
	}

	var meta LoaderMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}

	return &meta, nil
}

func ComputeFileSHA256(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

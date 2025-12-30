package generator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	"github.com/kintone/kpdev/internal/config"
	"github.com/kintone/kpdev/internal/prompt"
)

func GenerateProject(projectDir string, answers *prompt.InitAnswers) error {
	// package.json
	if err := generatePackageJSON(projectDir, answers); err != nil {
		return err
	}

	// manifest.json
	if err := GenerateManifest(projectDir, answers); err != nil {
		return err
	}

	// src/main/
	if err := generateMainEntry(projectDir, answers); err != nil {
		return err
	}

	// src/config/
	if err := generateConfigEntry(projectDir, answers); err != nil {
		return err
	}

	// icon.png (placeholder)
	if err := GenerateIcon(projectDir); err != nil {
		return err
	}

	// .gitignore
	if err := generateGitignore(projectDir); err != nil {
		return err
	}

	// README.md
	if err := generateReadme(projectDir, answers); err != nil {
		return err
	}

	return nil
}

// packageJSON はpackage.jsonの構造を定義する（フィールド順序を保証）
type packageJSON struct {
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	Private         bool              `json:"private"`
	Type            string            `json:"type"`
	Scripts         packageScripts    `json:"scripts"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

// packageScripts はpackage.jsonのscriptsセクションを定義する
type packageScripts struct {
	Dev    string `json:"dev"`
	Build  string `json:"build"`
	Deploy string `json:"deploy"`
}

func generatePackageJSON(projectDir string, answers *prompt.InitAnswers) error {
	deps := map[string]string{}
	devDeps := map[string]string{
		"vite": "^6.0.0",
	}

	switch answers.Framework {
	case prompt.FrameworkReact:
		deps["react"] = "^18.3.0"
		deps["react-dom"] = "^18.3.0"
		devDeps["@vitejs/plugin-react"] = "^4.3.0"
		if answers.Language == prompt.LanguageTypeScript {
			devDeps["@types/react"] = "^18.3.0"
			devDeps["@types/react-dom"] = "^18.3.0"
		}
	case prompt.FrameworkVue:
		deps["vue"] = "^3.5.0"
		devDeps["@vitejs/plugin-vue"] = "^5.2.0"
		if answers.Language == prompt.LanguageTypeScript {
			devDeps["vue-tsc"] = "^2.0.0"
		}
	case prompt.FrameworkSvelte:
		deps["svelte"] = "^5.0.0"
		devDeps["@sveltejs/vite-plugin-svelte"] = "^4.0.0"
		if answers.Language == prompt.LanguageTypeScript {
			devDeps["svelte-check"] = "^4.0.0"
		}
	}

	if answers.Language == prompt.LanguageTypeScript {
		devDeps["typescript"] = "^5.6.0"
	}

	pkg := packageJSON{
		Name:    answers.ProjectName,
		Version: "1.0.0",
		Private: true,
		Type:    "module",
		Scripts: packageScripts{
			Dev:    "kpdev dev",
			Build:  "kpdev build",
			Deploy: "kpdev deploy",
		},
		Dependencies:    deps,
		DevDependencies: devDeps,
	}

	data, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(projectDir, "package.json"), data, 0644)
}

// GenerateManifest generates .kpdev/manifest.json
func GenerateManifest(projectDir string, answers *prompt.InitAnswers) error {
	// プラグイン名（デフォルトはプロジェクト名）
	nameJa := answers.PluginNameJa
	if nameJa == "" {
		nameJa = answers.ProjectName
	}
	nameEn := answers.PluginNameEn
	if nameEn == "" {
		nameEn = answers.ProjectName
	}

	// 説明（デフォルトはプラグイン名 + プラグイン）
	descJa := answers.DescriptionJa
	if descJa == "" {
		descJa = nameJa + " プラグイン"
	}
	descEn := answers.DescriptionEn
	if descEn == "" {
		descEn = nameEn + " plugin"
	}

	manifest := map[string]interface{}{
		"manifest_version": 1,
		"version":          "1.0.0",
		"type":             "APP",
		"name": map[string]string{
			"ja": nameJa,
			"en": nameEn,
		},
		"description": map[string]string{
			"ja": descJa,
			"en": descEn,
		},
		"icon": "icon.png",
	}

	if answers.TargetDesktop {
		manifest["desktop"] = map[string]interface{}{
			"js":  []string{"js/desktop.js"},
			"css": []string{"css/desktop.css"},
		}
	}

	if answers.TargetMobile {
		manifest["mobile"] = map[string]interface{}{
			"js":  []string{"js/mobile.js"},
			"css": []string{"css/mobile.css"},
		}
	}

	manifest["config"] = map[string]interface{}{
		"html": "html/config.html",
		"js":   []string{"js/config.js"},
		"css":  []string{"css/config.css"},
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}

	// .kpdev ディレクトリを作成
	configDir := config.GetConfigDir(projectDir)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	// .kpdev/manifest.json に保存
	manifestPath := filepath.Join(configDir, "manifest.json")
	return os.WriteFile(manifestPath, data, 0644)
}

func generateMainEntry(projectDir string, answers *prompt.InitAnswers) error {
	mainDir := filepath.Join(projectDir, "src", "main")
	if err := os.MkdirAll(mainDir, 0755); err != nil {
		return err
	}

	ext := getEntryExtension(answers.Framework, answers.Language)
	entryFile := filepath.Join(mainDir, "main"+ext)

	var content string
	switch answers.Framework {
	case prompt.FrameworkReact:
		content = generateReactMain(answers.Language)
	case prompt.FrameworkVue:
		content = generateVueMain(answers.Language)
	case prompt.FrameworkSvelte:
		content = generateSvelteMain(answers.Language)
	default:
		content = generateVanillaMain(answers.Language)
	}

	if err := os.WriteFile(entryFile, []byte(content), 0644); err != nil {
		return err
	}

	// App component
	if answers.Framework != prompt.FrameworkVanilla {
		if err := generateAppComponent(mainDir, answers, "main"); err != nil {
			return err
		}
	}

	// style.css
	css := `/* メインスタイル */
.kpdev-main-root {
  padding: 16px;
}
`
	return os.WriteFile(filepath.Join(mainDir, "style.css"), []byte(css), 0644)
}

func generateConfigEntry(projectDir string, answers *prompt.InitAnswers) error {
	configDir := filepath.Join(projectDir, "src", "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	ext := getEntryExtension(answers.Framework, answers.Language)
	entryFile := filepath.Join(configDir, "main"+ext)

	var content string
	switch answers.Framework {
	case prompt.FrameworkReact:
		content = generateReactConfigMain(answers.Language)
	case prompt.FrameworkVue:
		content = generateVueConfigMain(answers.Language)
	case prompt.FrameworkSvelte:
		content = generateSvelteConfigMain(answers.Language)
	default:
		content = generateVanillaConfigMain(answers.Language)
	}

	if err := os.WriteFile(entryFile, []byte(content), 0644); err != nil {
		return err
	}

	// App component
	if answers.Framework != prompt.FrameworkVanilla {
		if err := generateAppComponent(configDir, answers, "config"); err != nil {
			return err
		}
	}

	// style.css
	css := `/* 設定画面スタイル */
#config-root {
  padding: 16px;
}
`
	if err := os.WriteFile(filepath.Join(configDir, "style.css"), []byte(css), 0644); err != nil {
		return err
	}

	// index.html
	html := `<div id="config-root"></div>
`
	return os.WriteFile(filepath.Join(configDir, "index.html"), []byte(html), 0644)
}

func generateAppComponent(dir string, answers *prompt.InitAnswers, target string) error {
	var content string
	var filename string

	switch answers.Framework {
	case prompt.FrameworkReact:
		if answers.Language == prompt.LanguageTypeScript {
			filename = "App.tsx"
		} else {
			filename = "App.jsx"
		}
		content = generateReactApp(target)
	case prompt.FrameworkVue:
		filename = "App.vue"
		content = generateVueApp(target)
	case prompt.FrameworkSvelte:
		filename = "App.svelte"
		content = generateSvelteApp(target)
	}

	return os.WriteFile(filepath.Join(dir, filename), []byte(content), 0644)
}

// GenerateIcon generates a default 56x56 plugin icon
func GenerateIcon(projectDir string) error {
	// 56x56 サンプルアイコンを生成
	img := image.NewRGBA(image.Rect(0, 0, 56, 56))

	// 背景色（グラデーション風）
	for y := 0; y < 56; y++ {
		for x := 0; x < 56; x++ {
			// 青系グラデーション
			r := uint8(70 + y)
			g := uint8(130 + y/2)
			b := uint8(220)
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}

	// 角丸風に角を透明に
	cornerRadius := 8
	for y := 0; y < cornerRadius; y++ {
		for x := 0; x < cornerRadius; x++ {
			dx := cornerRadius - x - 1
			dy := cornerRadius - y - 1
			if dx*dx+dy*dy > cornerRadius*cornerRadius {
				img.Set(x, y, color.RGBA{0, 0, 0, 0})
				img.Set(55-x, y, color.RGBA{0, 0, 0, 0})
				img.Set(x, 55-y, color.RGBA{0, 0, 0, 0})
				img.Set(55-x, 55-y, color.RGBA{0, 0, 0, 0})
			}
		}
	}

	// "P" の文字を描画（プラグインの P）
	white := color.RGBA{255, 255, 255, 255}
	// P の縦線
	for y := 16; y <= 40; y++ {
		for x := 18; x <= 22; x++ {
			img.Set(x, y, white)
		}
	}
	// P の上部横線
	for x := 22; x <= 34; x++ {
		for y := 16; y <= 20; y++ {
			img.Set(x, y, white)
		}
	}
	// P の中央横線
	for x := 22; x <= 34; x++ {
		for y := 26; y <= 30; y++ {
			img.Set(x, y, white)
		}
	}
	// P の右縦線（上部のみ）
	for y := 20; y <= 26; y++ {
		for x := 34; x <= 38; x++ {
			img.Set(x, y, white)
		}
	}

	// PNGとして保存
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(projectDir, "icon.png"), buf.Bytes(), 0644)
}

func generateGitignore(projectDir string) error {
	content := `# Dependencies
node_modules/

# Build output
dist/

# Environment
.env

# kpdev managed files
.kpdev/config.json
.kpdev/certs/

# IDE
.idea/
.vscode/
*.swp
*.swo

# OS
.DS_Store
Thumbs.db
`
	return os.WriteFile(filepath.Join(projectDir, ".gitignore"), []byte(content), 0644)
}

func generateReadme(projectDir string, answers *prompt.InitAnswers) error {
	content := fmt.Sprintf(`# %s

kintone プラグイン

## 開発

%s
%s dev
%s

## ビルド

%s
%s build
%s

## デプロイ

%s
%s deploy
%s
`, answers.ProjectName, "```bash", "kpdev", "```", "```bash", "kpdev", "```", "```bash", "kpdev", "```")
	return os.WriteFile(filepath.Join(projectDir, "README.md"), []byte(content), 0644)
}

func getEntryExtension(framework prompt.Framework, language prompt.Language) string {
	switch framework {
	case prompt.FrameworkReact:
		if language == prompt.LanguageTypeScript {
			return ".tsx"
		}
		return ".jsx"
	case prompt.FrameworkVue:
		return ".ts"
	case prompt.FrameworkSvelte:
		if language == prompt.LanguageTypeScript {
			return ".ts"
		}
		return ".js"
	default:
		if language == prompt.LanguageTypeScript {
			return ".ts"
		}
		return ".js"
	}
}

func GetEntryPath(framework prompt.Framework, language prompt.Language, target string) string {
	ext := getEntryExtension(framework, language)
	return fmt.Sprintf("/src/%s/main%s", target, ext)
}

// React templates
func generateReactMain(language prompt.Language) string {
	return `import { createRoot } from 'react-dom/client'
import App from './App'
import './style.css'

kintone.events.on(['app.record.index.show', 'app.record.detail.show'], (event) => {
  const el = kintone.app.getHeaderSpaceElement()
  if (el && !el.querySelector('#kpdev-main-root')) {
    const container = document.createElement('div')
    container.id = 'kpdev-main-root'
    el.appendChild(container)
    createRoot(container).render(<App />)
  }
  return event
})
`
}

func generateReactConfigMain(language prompt.Language) string {
	return `import { createRoot } from 'react-dom/client'
import App from './App'
import './style.css'

const container = document.getElementById('config-root')
if (container) {
  createRoot(container).render(<App />)
}
`
}

func generateReactApp(target string) string {
	if target == "config" {
		return `export default function App() {
  return (
    <div>
      <h1>プラグイン設定</h1>
      <p>設定画面のコンテンツをここに実装してください</p>
    </div>
  )
}
`
	}
	return `export default function App() {
  return (
    <div className="kpdev-main-root">
      <h1>Hello from kpdev!</h1>
    </div>
  )
}
`
}

// Vue templates
func generateVueMain(language prompt.Language) string {
	return `import { createApp } from 'vue'
import App from './App.vue'
import './style.css'

kintone.events.on(['app.record.index.show', 'app.record.detail.show'], (event) => {
  const el = kintone.app.getHeaderSpaceElement()
  if (el && !el.querySelector('#kpdev-main-root')) {
    const container = document.createElement('div')
    container.id = 'kpdev-main-root'
    el.appendChild(container)
    createApp(App).mount(container)
  }
  return event
})
`
}

func generateVueConfigMain(language prompt.Language) string {
	return `import { createApp } from 'vue'
import App from './App.vue'
import './style.css'

const container = document.getElementById('config-root')
if (container) {
  createApp(App).mount(container)
}
`
}

func generateVueApp(target string) string {
	if target == "config" {
		return `<template>
  <div>
    <h1>プラグイン設定</h1>
    <p>設定画面のコンテンツをここに実装してください</p>
  </div>
</template>

<script setup lang="ts">
</script>
`
	}
	return `<template>
  <div class="kpdev-main-root">
    <h1>Hello from kpdev!</h1>
  </div>
</template>

<script setup lang="ts">
</script>
`
}

// Svelte templates
func generateSvelteMain(language prompt.Language) string {
	return `import App from './App.svelte'
import './style.css'

kintone.events.on(['app.record.index.show', 'app.record.detail.show'], (event) => {
  const el = kintone.app.getHeaderSpaceElement()
  if (el && !el.querySelector('#kpdev-main-root')) {
    const container = document.createElement('div')
    container.id = 'kpdev-main-root'
    el.appendChild(container)
    new App({ target: container })
  }
  return event
})
`
}

func generateSvelteConfigMain(language prompt.Language) string {
	return `import App from './App.svelte'
import './style.css'

const container = document.getElementById('config-root')
if (container) {
  new App({ target: container })
}
`
}

func generateSvelteApp(target string) string {
	if target == "config" {
		return `<script>
</script>

<div>
  <h1>プラグイン設定</h1>
  <p>設定画面のコンテンツをここに実装してください</p>
</div>
`
	}
	return `<script>
</script>

<div class="kpdev-main-root">
  <h1>Hello from kpdev!</h1>
</div>
`
}

// Vanilla templates
func generateVanillaMain(language prompt.Language) string {
	return `import './style.css'

kintone.events.on(['app.record.index.show', 'app.record.detail.show'], (event) => {
  const el = kintone.app.getHeaderSpaceElement()
  if (el && !el.querySelector('#kpdev-main-root')) {
    const container = document.createElement('div')
    container.id = 'kpdev-main-root'
    container.innerHTML = '<h1>Hello from kpdev!</h1>'
    el.appendChild(container)
  }
  return event
})
`
}

func generateVanillaConfigMain(language prompt.Language) string {
	return `import './style.css'

const container = document.getElementById('config-root')
if (container) {
  container.innerHTML = '<h1>プラグイン設定</h1><p>設定画面のコンテンツをここに実装してください</p>'
}
`
}

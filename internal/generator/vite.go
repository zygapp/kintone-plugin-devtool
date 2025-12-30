package generator

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kintone/kpdev/internal/config"
	"github.com/kintone/kpdev/internal/prompt"
)

func GenerateViteConfig(projectDir string, framework prompt.Framework, language prompt.Language) error {
	configDir := config.GetConfigDir(projectDir)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	content := generateViteConfigContent(framework, language)
	configPath := filepath.Join(configDir, "vite.config.ts")

	return os.WriteFile(configPath, []byte(content), 0644)
}

func generateViteConfigContent(framework prompt.Framework, language prompt.Language) string {
	var pluginImport string
	var pluginUse string

	switch framework {
	case prompt.FrameworkReact:
		pluginImport = `import react from '@vitejs/plugin-react'`
		pluginUse = "react()"
	case prompt.FrameworkVue:
		pluginImport = `import vue from '@vitejs/plugin-vue'`
		pluginUse = "vue()"
	case prompt.FrameworkSvelte:
		pluginImport = `import { svelte } from '@sveltejs/vite-plugin-svelte'`
		pluginUse = "svelte()"
	default:
		pluginImport = ""
		pluginUse = ""
	}

	// エントリポイントの拡張子を決定
	ext := getEntryExtension(framework, language)

	var plugins string
	if pluginUse != "" {
		plugins = fmt.Sprintf(`
  plugins: [%s, kpdevMiddleware()],`, pluginUse)
	} else {
		plugins = `
  plugins: [kpdevMiddleware()],`
	}

	return fmt.Sprintf(`import { defineConfig } from 'vite'
import path from 'path'
import fs from 'fs'
import { build } from 'vite'
%s

// 証明書パス
const certDir = path.resolve(__dirname, 'certs')
const keyPath = path.join(certDir, 'localhost-key.pem')
const certPath = path.join(certDir, 'localhost.pem')

// loader.meta.json からドメインを取得
function getKintoneDomain(): string {
  try {
    const metaPath = path.resolve(__dirname, 'managed/loader.meta.json')
    const meta = JSON.parse(fs.readFileSync(metaPath, 'utf-8'))
    return meta.kintone?.domain || ''
  } catch {
    return ''
  }
}

// kpdev 開発サーバー用ミドルウェア
function kpdevMiddleware() {
  const cache = new Map<string, { code: string; time: number }>()
  const CACHE_TTL = 1000 // 1秒キャッシュ

  return {
    name: 'kpdev-middleware',
    configureServer(server: any) {
      // Vite 7 対応: handleHotUpdate の代わりに watcher を使用
      server.watcher.on('change', (file: string) => {
        // src/config/index.html は kpdev が監視して再デプロイ後に HMR 発火するので無視
        if (file.includes('src/config') && file.endsWith('.html')) {
          console.log('[kpdev] HTML change detected, waiting for redeploy...')
          return
        }
        cache.clear()
        server.ws.send({ type: 'full-reload' })
      })
      server.middlewares.use(async (req: any, res: any, next: any) => {
        const url = req.url?.split('?')[0]

        // ルートアクセス時は kintone へリダイレクト
        if (url === '/' || url === '/index.html') {
          const domain = getKintoneDomain()
          const kintoneUrl = domain ? 'https://' + domain + '/k/' : ''

          const html = '<!DOCTYPE html>' +
'<html>' +
'<head>' +
'  <meta charset="UTF-8">' +
'  <title>kpdev - Redirecting...</title>' +
'  <style>' +
'    body {' +
'      font-family: system-ui, -apple-system, sans-serif;' +
'      display: flex;' +
'      justify-content: center;' +
'      align-items: center;' +
'      height: 100vh;' +
'      margin: 0;' +
'      background: #f5f5f5;' +
'    }' +
'    .container {' +
'      text-align: center;' +
'      padding: 40px;' +
'      background: white;' +
'      border-radius: 8px;' +
'      box-shadow: 0 2px 10px rgba(0,0,0,0.1);' +
'    }' +
'    h1 { color: #333; margin-bottom: 16px; }' +
'    p { color: #666; margin-bottom: 24px; }' +
'    a {' +
'      display: inline-block;' +
'      padding: 12px 24px;' +
'      background: #0066cc;' +
'      color: white;' +
'      text-decoration: none;' +
'      border-radius: 4px;' +
'    }' +
'    a:hover { background: #0052a3; }' +
'  </style>' +
'</head>' +
'<body>' +
'  <div class="container">' +
'    <h1>kpdev Dev Server</h1>' +
'    <p>SSL証明書が許可されました。kintoneに移動します...</p>' +
(kintoneUrl ? '    <a href="' + kintoneUrl + '">kintoneを開く</a>' : '') +
'  </div>' +
(kintoneUrl ? '  <script>setTimeout(function() { window.location.href = "' + kintoneUrl + '"; }, 2000);</script>' : '') +
'</body>' +
'</html>'

          res.setHeader('Content-Type', 'text/html; charset=utf-8')
          res.end(html)
          return
        }

        // /config.html は src/config/index.html を配信
        if (url === '/config.html') {
          try {
            const htmlPath = path.resolve(__dirname, '../src/config/index.html')
            const html = fs.readFileSync(htmlPath, 'utf-8')
            res.setHeader('Content-Type', 'text/html; charset=utf-8')
            res.setHeader('Access-Control-Allow-Origin', '*')
            res.end(html)
            return
          } catch (e) {
            console.error('Config HTML read error:', e)
          }
        }

        // /main.js, /config.js は IIFE ビルドして配信
        if (url === '/main.js' || url === '/config.js') {
          const entry = url === '/main.js' ? 'main' : 'config'
          const entryPath = path.resolve(__dirname, '../src/' + entry + '/main%s')

          try {
            // キャッシュチェック
            const cached = cache.get(entry)
            const now = Date.now()
            if (cached && (now - cached.time) < CACHE_TTL) {
              res.setHeader('Content-Type', 'application/javascript')
              res.setHeader('Access-Control-Allow-Origin', '*')
              res.end(cached.code)
              return
            }

            // Vite でオンデマンドビルド
            const result = await build({
              configFile: false,
              root: path.resolve(__dirname, '..'),
              logLevel: 'silent',
              plugins: [%s].filter(Boolean),
              build: {
                write: false,
                minify: false,
                rollupOptions: {
                  input: entryPath,
                  output: {
                    format: 'iife',
                    entryFileNames: '[name].js',
                  },
                },
              },
            })

            const output = (result as any).output || (result as any)[0]?.output
            if (output && output[0]) {
              const code = output[0].code
              cache.set(entry, { code, time: now })
              res.setHeader('Content-Type', 'application/javascript')
              res.setHeader('Access-Control-Allow-Origin', '*')
              res.end(code)
              return
            }
          } catch (e) {
            console.error('IIFE build error:', e)
          }
        }
        next()
      })
    },
  }
}

export default defineConfig({%s
  root: path.resolve(__dirname, '..'),
  server: {
    port: 3000,
    https: fs.existsSync(keyPath) && fs.existsSync(certPath)
      ? {
          key: fs.readFileSync(keyPath),
          cert: fs.readFileSync(certPath),
        }
      : undefined,
    cors: true,
    headers: {
      'Access-Control-Allow-Origin': '*',
    },
    watch: {
      ignored: ['**/src/config/index.html'],
    },
  },
  build: {
    outDir: 'dist',
    emptyOutDir: process.env.VITE_BUILD_ENTRY !== 'config',
    rollupOptions: {
      input: process.env.VITE_BUILD_ENTRY === 'config'
        ? { config: path.resolve(__dirname, '../src/config/main%s') }
        : { main: path.resolve(__dirname, '../src/main/main%s') },
      output: {
        format: 'iife',
        entryFileNames: '[name].js',
        assetFileNames: process.env.VITE_BUILD_ENTRY === 'config' ? 'config.[ext]' : 'main.[ext]',
      },
    },
    cssCodeSplit: false,
    minify: 'esbuild',
  },
  esbuild: {
    drop: ['console', 'debugger'],
  },
})
`, pluginImport, ext, pluginUse, plugins, ext, ext)
}


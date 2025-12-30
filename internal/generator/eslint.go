package generator

import (
	"os"
	"path/filepath"

	"github.com/kintone/kpdev/internal/prompt"
)

// GenerateESLintConfig はフレームワーク別のESLint設定を生成する
func GenerateESLintConfig(projectDir string, framework prompt.Framework, language prompt.Language) error {
	configPath := filepath.Join(projectDir, "eslint.config.js")

	// 既存ファイルがあればスキップ
	if _, err := os.Stat(configPath); err == nil {
		return nil
	}

	var content string
	switch framework {
	case prompt.FrameworkReact:
		content = getReactESLintConfig(language)
	case prompt.FrameworkVue:
		content = getVueESLintConfig(language)
	case prompt.FrameworkSvelte:
		content = getSvelteESLintConfig(language)
	default:
		content = getVanillaESLintConfig(language)
	}

	return os.WriteFile(configPath, []byte(content), 0644)
}

func getReactESLintConfig(language prompt.Language) string {
	if language == prompt.LanguageTypeScript {
		return `import js from "@eslint/js";
import globals from "globals";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";
import tseslint from "typescript-eslint";

export default tseslint.config(
  { ignores: ["dist", ".kpdev"] },
  {
    extends: [js.configs.recommended, ...tseslint.configs.recommended],
    files: ["**/*.{ts,tsx}"],
    languageOptions: {
      ecmaVersion: 2020,
      globals: globals.browser,
    },
    plugins: {
      "react-hooks": reactHooks,
      "react-refresh": reactRefresh,
    },
    rules: {
      ...reactHooks.configs.recommended.rules,
      "react-refresh/only-export-components": [
        "warn",
        { allowConstantExport: true },
      ],
    },
  }
);
`
	}
	return `import js from "@eslint/js";
import globals from "globals";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";

export default [
  { ignores: ["dist", ".kpdev"] },
  {
    extends: [js.configs.recommended],
    files: ["**/*.{js,jsx}"],
    languageOptions: {
      ecmaVersion: 2020,
      globals: globals.browser,
    },
    plugins: {
      "react-hooks": reactHooks,
      "react-refresh": reactRefresh,
    },
    rules: {
      ...reactHooks.configs.recommended.rules,
      "react-refresh/only-export-components": [
        "warn",
        { allowConstantExport: true },
      ],
    },
  },
];
`
}

func getVueESLintConfig(language prompt.Language) string {
	if language == prompt.LanguageTypeScript {
		return `import js from "@eslint/js";
import globals from "globals";
import tseslint from "typescript-eslint";
import pluginVue from "eslint-plugin-vue";

export default tseslint.config(
  { ignores: ["dist", ".kpdev"] },
  {
    extends: [
      js.configs.recommended,
      ...tseslint.configs.recommended,
      ...pluginVue.configs["flat/essential"],
    ],
    files: ["**/*.{ts,vue}"],
    languageOptions: {
      ecmaVersion: 2020,
      globals: globals.browser,
    },
  }
);
`
	}
	return `import js from "@eslint/js";
import globals from "globals";
import pluginVue from "eslint-plugin-vue";

export default [
  { ignores: ["dist", ".kpdev"] },
  {
    extends: [js.configs.recommended, ...pluginVue.configs["flat/essential"]],
    files: ["**/*.{js,vue}"],
    languageOptions: {
      ecmaVersion: 2020,
      globals: globals.browser,
    },
  },
];
`
}

func getSvelteESLintConfig(language prompt.Language) string {
	if language == prompt.LanguageTypeScript {
		return `import js from "@eslint/js";
import globals from "globals";
import tseslint from "typescript-eslint";
import svelte from "eslint-plugin-svelte";
import svelteParser from "svelte-eslint-parser";

export default tseslint.config(
  { ignores: ["dist", ".kpdev"] },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  ...svelte.configs["flat/recommended"],
  {
    files: ["**/*.svelte"],
    languageOptions: {
      parser: svelteParser,
      parserOptions: {
        parser: tseslint.parser,
      },
    },
  },
  {
    languageOptions: {
      globals: {
        ...globals.browser,
        ...globals.node,
      },
    },
  }
);
`
	}
	return `import js from "@eslint/js";
import globals from "globals";
import svelte from "eslint-plugin-svelte";

export default [
  { ignores: ["dist", ".kpdev"] },
  js.configs.recommended,
  ...svelte.configs["flat/recommended"],
  {
    languageOptions: {
      globals: {
        ...globals.browser,
        ...globals.node,
      },
    },
  },
];
`
}

func getVanillaESLintConfig(language prompt.Language) string {
	if language == prompt.LanguageTypeScript {
		return `import js from "@eslint/js";
import globals from "globals";
import tseslint from "typescript-eslint";

export default tseslint.config(
  { ignores: ["dist", ".kpdev"] },
  {
    extends: [js.configs.recommended, ...tseslint.configs.recommended],
    files: ["**/*.ts"],
    languageOptions: {
      ecmaVersion: 2020,
      globals: globals.browser,
    },
  }
);
`
	}
	return `import js from "@eslint/js";
import globals from "globals";

export default [
  { ignores: ["dist", ".kpdev"] },
  {
    extends: [js.configs.recommended],
    files: ["**/*.js"],
    languageOptions: {
      ecmaVersion: 2020,
      globals: globals.browser,
    },
  },
];
`
}

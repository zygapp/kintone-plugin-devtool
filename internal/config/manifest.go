package config

import (
	"encoding/json"
	"strings"
)

// ManifestKeyOrder はmanifest.jsonのプロパティ標準順序
var ManifestKeyOrder = []string{
	"version", "manifest_version", "type", "icon",
	"name", "description", "homepage_url",
	"config", "desktop", "mobile",
}

// MarshalManifestJSON はmanifestを標準順序でJSON文字列に変換する
func MarshalManifestJSON(manifest map[string]interface{}) string {
	var sb strings.Builder
	sb.WriteString("{\n")

	first := true
	for _, key := range ManifestKeyOrder {
		if val, ok := manifest[key]; ok {
			if !first {
				sb.WriteString(",\n")
			}
			first = false
			writeJSONField(&sb, key, val, "  ")
		}
	}

	// その他のキーを追加
	for key, val := range manifest {
		found := false
		for _, k := range ManifestKeyOrder {
			if k == key {
				found = true
				break
			}
		}
		if !found {
			if !first {
				sb.WriteString(",\n")
			}
			first = false
			writeJSONField(&sb, key, val, "  ")
		}
	}

	sb.WriteString("\n}")
	return sb.String()
}

func writeJSONField(sb *strings.Builder, key string, val interface{}, indent string) {
	jsonVal, _ := json.MarshalIndent(val, indent, "  ")
	jsonStr := string(jsonVal)
	sb.WriteString(indent)
	sb.WriteString("\"")
	sb.WriteString(key)
	sb.WriteString("\": ")
	sb.WriteString(jsonStr)
}

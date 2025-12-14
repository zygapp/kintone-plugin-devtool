package kintone

import (
	"encoding/json"
	"fmt"
)

// ImportPlugin は非公式APIでプラグインをインポートする
func (c *Client) ImportPlugin(fileKey string) (*PluginImportResult, error) {
	body := map[string]string{
		"item": fileKey,
	}

	// 非公式API: /k/api/dev/plugin/import.json を使用
	respBody, err := c.doRequest("POST", "/k/api/dev/plugin/import.json", body)
	if err != nil {
		return nil, err
	}

	var resp PluginImportResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("レスポンス解析エラー: %w (body: %s)", err, string(respBody))
	}

	if !resp.Success {
		return nil, fmt.Errorf("インポート失敗: %s", string(respBody))
	}

	return &resp.Result, nil
}

type PluginImportResponse struct {
	Success bool               `json:"success"`
	Result  PluginImportResult `json:"result"`
}

type PluginImportResult struct {
	ID      string `json:"id"`
	Version int    `json:"version"`
}

// GetPlugins はインストール済みプラグイン一覧を取得
func (c *Client) GetPlugins() ([]PluginInfo, error) {
	respBody, err := c.doRequest("GET", "/k/v1/plugins.json", nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Plugins []PluginInfo `json:"plugins"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	return result.Plugins, nil
}

type PluginInfo struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

// FindPluginByID は指定IDのプラグインを検索
func (c *Client) FindPluginByID(pluginID string) (*PluginInfo, error) {
	plugins, err := c.GetPlugins()
	if err != nil {
		return nil, err
	}

	for _, p := range plugins {
		if p.ID == pluginID {
			return &p, nil
		}
	}

	return nil, fmt.Errorf("プラグインが見つかりません: %s", pluginID)
}

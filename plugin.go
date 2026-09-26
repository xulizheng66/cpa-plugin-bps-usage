package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginabi"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
	"gopkg.in/yaml.v3"
)

const (
	pluginName        = "bps-usage"
	pluginDisplayName = "BPS 用量"
	pluginVersion     = "0.1.1"
	pluginAuthor      = "xulizheng66"
	pluginRepository  = "https://github.com/xulizheng66/cpa-plugin-bps-usage"
	resourcePath      = "/dashboard"
)

//go:embed web/dashboard.html
var webFS embed.FS

type pluginConfig struct {
	ShimURL        string `yaml:"shim_url"`
	ShimKey        string `yaml:"shim_key"`
	RefreshSeconds int    `yaml:"refresh_seconds"`
	Days           int    `yaml:"days"`
	Limit          int    `yaml:"limit"`
	Title          string `yaml:"title"`
}

type lifecycleRequest struct {
	ConfigYAML    []byte `json:"config_yaml"`
	SchemaVersion uint32 `json:"schema_version"`
}

type registration struct {
	SchemaVersion uint32                 `json:"schema_version"`
	Metadata      pluginapi.Metadata     `json:"metadata"`
	Capabilities  registrationCapability `json:"capabilities"`
}

type registrationCapability struct {
	ManagementAPI bool `json:"management_api"`
}

type managementRegistrationResponse struct {
	Resources []pluginapi.ResourceRoute `json:"resources,omitempty"`
}

var (
	stateMu sync.RWMutex
	cfg     = pluginConfig{
		ShimURL:        "/bps",
		RefreshSeconds: 30,
		Days:           7,
		Limit:          100,
		Title:          pluginDisplayName,
	}
)

func handleMethod(method string, request []byte) ([]byte, error) {
	switch method {
	case pluginabi.MethodPluginRegister, pluginabi.MethodPluginReconfigure:
		reloadConfig(request)
		return okEnvelope(pluginRegistration())
	case pluginabi.MethodManagementRegister:
		return okEnvelope(managementRegistrationResponse{Resources: []pluginapi.ResourceRoute{{
			Path:        resourcePath,
			Menu:        pluginDisplayName,
			Description: "查看 bps-shim 采集的用量（BPS 直连 + 转发 CPA）。",
		}}})
	case pluginabi.MethodManagementHandle:
		return okEnvelope(pluginapi.ManagementResponse{
			StatusCode: http.StatusOK,
			Headers:    http.Header{"Content-Type": []string{"text/html; charset=utf-8"}},
			Body:       dashboardHTML(),
		})
	default:
		return errorEnvelope("unknown_method", "unknown method: "+method), nil
	}
}

func pluginRegistration() registration {
	return registration{
		SchemaVersion: pluginabi.SchemaVersion,
		Metadata: pluginapi.Metadata{
			Name:             pluginName,
			Version:          pluginVersion,
			Author:           pluginAuthor,
			GitHubRepository: pluginRepository,
			ConfigFields: []pluginapi.ConfigField{
				{Name: "shim_url", Type: pluginapi.ConfigFieldTypeString,
					Description: "bps-shim 的地址。同源推荐 /bps；也可填完整 URL（需与 CPA 同源，否则浏览器会因 CORS 拒绝）。"},
				{Name: "shim_key", Type: pluginapi.ConfigFieldTypeString,
					Description: "bps-shim 的 BPS_SHIM_KEY（用于读取 /usage.json）。"},
				{Name: "refresh_seconds", Type: pluginapi.ConfigFieldTypeInteger,
					Description: "页面自动刷新间隔（秒），默认 30。"},
				{Name: "days", Type: pluginapi.ConfigFieldTypeInteger,
					Description: "汇总统计的天数，默认 7。"},
				{Name: "limit", Type: pluginapi.ConfigFieldTypeInteger,
					Description: "最近请求明细条数，默认 100。"},
				{Name: "title", Type: pluginapi.ConfigFieldTypeString,
					Description: "页面标题，默认「BPS 用量」。"},
			},
		},
		Capabilities: registrationCapability{ManagementAPI: true},
	}
}

func reloadConfig(request []byte) {
	var req lifecycleRequest
	if len(request) > 0 {
		_ = json.Unmarshal(request, &req)
	}
	next := pluginConfig{ShimURL: "/bps", RefreshSeconds: 30, Days: 7, Limit: 100, Title: pluginDisplayName}
	if len(req.ConfigYAML) > 0 {
		_ = yaml.Unmarshal(req.ConfigYAML, &next)
	}
	next.ShimURL = strings.TrimSpace(next.ShimURL)
	if next.ShimURL == "" {
		next.ShimURL = "/bps"
	}
	next.ShimURL = strings.TrimRight(next.ShimURL, "/")
	if next.RefreshSeconds <= 0 {
		next.RefreshSeconds = 30
	}
	if next.Days <= 0 {
		next.Days = 7
	}
	if next.Limit <= 0 {
		next.Limit = 100
	}
	if strings.TrimSpace(next.Title) == "" {
		next.Title = pluginDisplayName
	}
	stateMu.Lock()
	cfg = next
	stateMu.Unlock()
}

func currentConfig() pluginConfig {
	stateMu.RLock()
	defer stateMu.RUnlock()
	return cfg
}

// dashboardHTML 把配置注入到内嵌页面（shim_key 落在此页面里，页面本身需 CPA 管理鉴权才能访问）。
func dashboardHTML() []byte {
	raw, errRead := webFS.ReadFile("web/dashboard.html")
	if errRead != nil {
		return []byte("<h1>BPS 用量</h1><p>内嵌页面缺失: " + errRead.Error() + "</p>")
	}
	c := currentConfig()
	cfgJSON, _ := json.Marshal(map[string]any{
		"shimUrl":        c.ShimURL,
		"shimKey":        c.ShimKey,
		"refreshSeconds": c.RefreshSeconds,
		"days":           c.Days,
		"limit":          c.Limit,
		"title":          c.Title,
	})
	out := strings.ReplaceAll(string(raw), "__BPS_USAGE_CONFIG__", string(cfgJSON))
	return []byte(out)
}

var _ = fmt.Sprintf

# CPA 插件：BPS 用量（cpa-plugin-bps-usage）

给 **CPA（CLIProxyAPI）** 增加一个 「BPS 用量」 菜单：在 CPAMC 里直接查看
[bps-shim](https://github.com/xulizheng66) 采集的用量 —— **BPS 直连**与**转发 CPA** 两条路径的请求数、
tokens（含缓存读/推理）、TTFT、耗时、工具调用与状态。

> 数据来源是 bps-shim 自己的用量库（`/bps/usage.json`），与 CPA / CPAMP / Keeper 的内部用量管道完全解耦。
> 注意：CPA/CPAMP 的「请求监控」仍然只显示经过 CPA 的请求；BPS 路径的请求只在本插件里可见。

## 安装

### 方式 A：从自定义商店源安装（推荐）

1. 在 CPA 的 `config.yaml` 里加入本仓库的商店源：

```yaml
plugins:
  enabled: true
  store-sources:
    - https://raw.githubusercontent.com/xulizheng66/cpa-plugin-bps-usage/main/store/registry.json
```

2. 重启/重载 CPA，然后在 **CPAMC → 插件商店** 里会出现「BPS 用量」，点安装。

### 方式 B：手动放置

```bash
# 从 Release 下载对应架构（本机 CPA 为 linux/arm64）
curl -LO https://github.com/xulizheng66/cpa-plugin-bps-usage/releases/latest/download/bps-usage-v0.1.0-linux-arm64.so
sudo mv bps-usage-v0.1.0-linux-arm64.so /root/cliproxyapi-docker/plugins/linux/arm64/
```

然后在 `config.yaml` 里启用并配置：

```yaml
plugins:
  enabled: true
  configs:
    bps-usage:
      enabled: true
      config_yaml: |
        shim_url: /bps
        shim_key: "<BPS_SHIM_KEY>"
        refresh_seconds: 30
        days: 7
        limit: 100
```

## 配置项

| 字段 | 默认 | 说明 |
| --- | --- | --- |
| `shim_url` | `/bps` | bps-shim 的地址。**推荐同源相对路径**（如 `/bps`），这样浏览器 fetch 不会触发 CORS；也可写完整 URL（必须与 CPA 同源） |
| `shim_key` | 空 | bps-shim 的 `BPS_SHIM_KEY`，用于读取 `/usage.json` |
| `refresh_seconds` | `30` | 页面自动刷新间隔 |
| `days` | `7` | 汇总统计天数 |
| `limit` | `100` | 最近请求明细条数 |
| `title` | `BPS 用量` | 页面标题 |

> 前置条件：bps-shim 需暴露 `/usage.json` 且**同时接受 `?key=` 查询参数鉴权**（shim v3.4+ 已支持）。

## 前置：bps-shim 侧

```bash
# 确认面板与 API 可用（在 CPA 服务器上）
KEY=$(sudo grep -oP 'BPS_SHIM_KEY: "\K[^"]+' /opt/bps-shim/docker-compose.yml)
curl -s -H "Authorization: Bearer $KEY" http://127.0.0.1:8325/usage.json?days=1 | head -c 200
```

## 构建

```bash
# 本机（macOS，验证代码）
CGO_ENABLED=1 go build -buildmode=c-shared -o /tmp/bps-usage.dylib .

# 交叉编译 linux/arm64（Debian/Ubuntu 需 apt install gcc-aarch64-linux-gnu）
make build-arm64 VERSION=0.1.0
```

CI：推送 `v*` tag 会自动构建 `linux/arm64` + `linux/amd64` 两个 `.so`、上传 Release、
并重写 `store/registry.json`（含 size/sha256，供商店源使用）。

## 工作原理

- 插件实现 CPA 插件 ABI v1 的四个方法：`plugin.register`、`plugin.reconfigure`、
  `management.register`、`management.handle`
- 能力声明：`management_api: true`
- `management.register` 注册资源路由 `/dashboard`（菜单名「BPS 用量」），CPAMC 实际地址为
  `/v0/resource/plugins/bps-usage/dashboard`
- `management.handle` 返回内嵌的前端页面（`web/dashboard.html`，`go:embed` 打包）
- 前端用配置里的 `shim_url` + `shim_key` 调 shim 的 `/usage.json` 并渲染表格

## 故障排查

| 现象 | 原因 | 处理 |
| --- | --- | --- |
| 菜单不出现 | 插件未启用 / 未加载 | 看 CPA 日志 `plugin ... loaded`；`GET /v0/management/plugins` 是否列出 `bps-usage` |
| 页面 404 | 资源路径变化 | 确认注册路径 `/dashboard`，并核对 `/v0/resource/plugins/bps-usage/dashboard` |
| 页面报 `HTTP 401` | `shim_key` 错 / shim 未支持 `?key=` | 用 `curl` 验证 `/usage.json?key=...` 返回 200 |
| 页面报 CORS / 网络错误 | `shim_url` 跨源 | 改成同源相对路径（如 `/bps`） |
| 数据为空 | 尚无请求 / 时区 | shim 用量库保留 90 天；页面时间固定 UTC+8 |

## 版本与兼容

- 插件 ABI：`schema_version = 1`（与 CPA v7.x 一致）
- 依赖 SDK：`github.com/router-for-me/CLIProxyAPI/v7`
- CPA 升级后若 ABI 变化导致加载失败，重新打 tag 触发 CI 构建即可

## License

MIT

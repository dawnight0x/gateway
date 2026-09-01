# Local AI Gateway

Local AI Gateway 是一个面向本地开发环境的多上游 AI 网关。编程工具只需连接 `localhost`，真实的 OpenAI-compatible、Anthropic-compatible、Gemini-compatible、NewAPI 和 Sub2API Key 统一在本地管理后台中配置。

项目使用 Go 构建单进程服务，内嵌 Vue 管理后台，并提供模型发现、优先路由、故障切换、协议转换、请求日志和本地密钥保护。

## 主要功能

- 统一提供 OpenAI Responses、Chat Completions、Anthropic Messages 和 Gemini generateContent 接口。
- 自动发现并保存上游模型，支持 Provider 模型白名单和定时刷新。
- 按模型、Provider、Key 优先级进行路由，并在明确失败时切换可用候选。
- 优先使用上游原生协议，仅在端点明确不支持时进行协议转换。
- 提供并发限制、冷却恢复、请求超时和避免重复计费的安全重试策略。
- 内嵌中英文管理后台，支持 Provider、Key、逻辑模型、日志和运行状态管理。
- 使用 SQLite 保存本地数据，并加密存储上游 Key；支持便携备份与恢复。
- Windows 提供系统托盘和单实例保护，同时支持 Linux amd64/arm64 运行。

详细协议边界见[协议兼容矩阵](docs/protocol-compatibility.md)。

## 快速开始

从源码运行需要 Go 1.25 或更高版本。`go.mod` 指定 Go 1.26.6 工具链，支持自动工具链下载的 Go 会按需获取对应版本。

```bash
go run ./cmd/gateway
```

默认地址：

- 管理后台：`http://localhost:18787/admin`
- OpenAI Base URL：`http://localhost:18787/v1`
- Anthropic Base URL：`http://localhost:18787`
- Gemini Base URL：`http://localhost:18787`

首次使用：

1. 启动网关并打开管理后台。
2. 使用 `data/admin.token` 中自动生成的管理员 Token 登录。
3. 添加 Provider 和真实上游 Key，网关会自动发现可用模型。
4. 在“网关 Key”页面创建本地 `sk-...` Key，供编程工具使用。
5. 如需固定主模型和备用模型，在“模型优先路由”页面创建逻辑模型。

默认只监听 `127.0.0.1`，不会直接暴露到公网。远程监听和 TLS 配置请先阅读[运维指南](docs/operations.md)。

## 客户端配置

OpenAI-compatible 客户端：

```bash
export OPENAI_BASE_URL=http://localhost:18787/v1
export OPENAI_API_KEY=sk-xxx
```

Anthropic-compatible 客户端：

```bash
export ANTHROPIC_BASE_URL=http://localhost:18787
export ANTHROPIC_AUTH_TOKEN=sk-xxx
```

Gemini-compatible 客户端：

```bash
export GEMINI_BASE_URL=http://localhost:18787
export GEMINI_API_KEY=sk-xxx
```

真实上游 Key 不需要写入编程工具，只在本地管理后台中保存。

## 配置与文档

需要自定义端口、数据目录、超时、并发限制或模型发现策略时，复制并修改示例配置：

```bash
cp config.example.yaml config.yaml
```

Windows PowerShell 可使用：

```powershell
Copy-Item config.example.yaml config.yaml
```

- [示例配置](config.example.yaml)
- [运维指南](docs/operations.md)
- [Linux 部署](docs/linux.md)
- [协议兼容矩阵](docs/protocol-compatibility.md)
- [发行包使用说明](docs/README-distribution.md)
- [升级与回滚](docs/upgrade.md)
- [参与开发与测试](CONTRIBUTING.md)

## 参与贡献

欢迎参与项目建设。添加功能、完善兼容性、改进文档、反馈问题或提出建议都非常欢迎：

- 发现 Bug、遇到使用问题或有功能建议，请提交 [Issue](https://github.com/dawnight0x/gateway/issues)。
- 希望添加功能或改进现有实现，请提交 [Pull Request](https://github.com/dawnight0x/gateway/pulls)。
- 开发环境、测试命令和提交要求见 [CONTRIBUTING.md](CONTRIBUTING.md)。

## License

本项目基于 [MIT License](LICENSE) 授权。

安全问题请按 [SECURITY.md](SECURITY.md) 中的方式私下报告。

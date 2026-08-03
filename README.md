# Local AI Gateway

本地 AI 网关：Go 后端核心 + 内嵌 Vue 本地管理后台 + 系统托盘。所有编程软件统一连接 `localhost`，真实 newapi、sub2api、OpenAI-compatible、Anthropic-compatible、Gemini-compatible key 在后台管理，由网关自动处理优先级、失败切换、协议转换和状态展示。

## 快速开始

从源码运行或构建需要 Go 1.25 或更高版本；`go.mod` 指定 Go 1.26.5 工具链，支持自动工具链下载的 Go 会按需获取该版本。运行网关本身不需要 Node.js。只有修改管理后台模板时需要 Node.js 运行 `npm run build:admin`；编译后的静态资源已提交并内嵌进 Go 程序。

```powershell
go run .\cmd\gateway
```

或构建单文件程序：

```powershell
New-Item -ItemType Directory -Force bin
go build -buildvcs=false -o .\bin\gateway.exe .\cmd\gateway
.\bin\gateway.exe
```

构建完成后，也可以直接双击 `bin\gateway.exe`。当程序从 `bin` 目录双击启动时，会自动把工作目录切回项目根目录，确保配置、数据库、日志和单实例锁仍使用根目录下的 `data`。

默认地址：

- 管理后台：`http://localhost:18787/admin`
- OpenAI base URL：`http://localhost:18787/v1`
- Anthropic base URL：`http://localhost:18787`
- Gemini base URL：`http://localhost:18787`
- 后台管理员 Token：首次运行会生成到 `data/admin.token`；也可以通过 `config.yaml` 或 `GATEWAY_ADMIN_TOKEN` 指定强随机值。
- 本地 API key：进入 `网关 Key` 页面创建随机 `sk-...` Key 给编程软件使用。
- 兼容本地 API key：仅当你显式设置 `server.proxy_token` 或 `GATEWAY_PROXY_TOKEN` 时才启用；默认不再接受固定 `local-gateway-key`。

默认实际监听绑定 `127.0.0.1`，用户展示和复制配置使用 `localhost`。这样既避免公网暴露，也避免 Windows 上 `localhost` 的 IPv4/IPv6 差异影响服务健康检查。

默认会启动系统托盘。托盘右键菜单：

- `打开管理后台`：打开 `http://localhost:18787/admin`，并通过地址 fragment 自动导入后台 Token；Token 不会作为 URL 内容发送到服务器，前端仅保存到当前标签页会话并立刻清理地址栏。
- `复制管理地址`：复制不含 Token 的普通后台地址。
- `复制 OpenAI 配置` / `复制 Anthropic 配置` / `复制 Gemini 配置`
- `打开数据目录`：打开数据库、密钥、锁文件和日志目录
- `打开日志文件`：打开本地运行日志
- `重启网关`：退出当前进程并重新拉起
- `退出网关`：关闭网关进程并移除托盘图标

网关会创建单实例锁文件。重复启动时，如果已有实例健康运行，会直接打开已有管理后台并退出新进程，避免端口冲突。

无托盘运行：

```powershell
$env:GATEWAY_TRAY="false"
.\bin\gateway.exe
```

## 编程软件配置

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

真实上游 key 不需要填到编程软件里，在 `/admin` 网页里添加 provider 和 key。

第一次打开管理后台时，前端会要求输入后台管理员 Token。默认配置保持 `change-me` 或留空时，实际运行 Token 会自动生成到 `data/admin.token`。管理 API 会拒绝未带 Token 的请求，并限制为本机同源访问。

## 已实现范围

- Go 单进程网关，内嵌 Vue 3 管理网页，无 Node.js 服务依赖。
- Windows 系统托盘：右键打开管理网页、复制配置、打开日志/数据目录、重启、退出网关。
- 单实例保护：重复启动时打开已有后台，不再抢占端口。
- 本地运行日志：默认写入 `data/gateway.log`，按 10 MiB 轮转并保留 3 个备份。
- SQLite WAL 本地存储，真实上游 key 使用 AES-256-GCM 加密保存；Windows 使用 DPAPI，Linux 优先使用 Secret Service，macOS 优先使用 Keychain 保护主密钥。无可用系统密钥环时回退到权限为 `0600` 的密钥文件，也可通过 `GATEWAY_MASTER_KEY` 接入外部密钥管理器。
- 管理 API 鉴权：`/admin/api/*` 需要 `X-Admin-Token` 或 `Authorization: Bearer`，并带有本机 Host/Origin 防护。
- 本地管理网页：Dashboard、Provider、模型优先路由、Key、路由设置、连接向导、最近请求，支持中文 / English 和深色 / 浅色主题切换。
- Provider 类型：`openai-compatible`、`anthropic-compatible`、`gemini-compatible`、`new-api`、`sub2api`、`custom`。
- 代理接口：`/health`、`/status`、`/metrics`、`/v1/models`、`/v1/chat/completions`、`/v1/responses`、`/v1/messages`、`/messages`、`/v1beta/models/{model}:generateContent`、`/v1beta/models/{model}:streamGenerateContent`。
- 代理请求会严格校验 JSON 和 `model`，响应通过 `X-Gateway-Request-ID` 提供可关联到本地请求日志的 ID；客户端主动取消不会计入上游 Key 失败或触发冷却。
- 模型发现：添加或更新上游 Key 后自动读取模型列表并按 Key 持久保存模型库存，支持分页并合并同 Provider 下所有成功 Key 的模型集合，默认每 24 小时刷新；自动任务不会发起消耗 Token 的生成请求，单个 Key 发现失败或返回空列表时保留该 Key 上次成功库存。发现后可搜索、全选或仅选择推荐模型，并开启 Provider 模型白名单。
- 模型优先路由：客户端请求逻辑模型 ID 后，先按模型优先级，再按 Provider 优先级，最后按 Key 顺序尝试；同优先级模型按用户配置顺序排列，当前模型的所有候选不可用后才进入下一备用模型。未配置逻辑模型时继续使用原有 `model_map` 和 Key 路由。
- 路由健康：上游明确表示模型不存在或不支持时，冷却作用于 `(Provider, 上游模型)`，后续请求和后来新增的 Key 都不会重复探测；Key 无模型权限时只冷却对应的 `(Provider, Key, 上游模型)`，仍会尝试同 Provider 的其他 Key。鉴权、限流和网络类错误继续作用于 Key；恢复探测、连续失败阈值和 `Retry-After` 规则保持有效。
- 安全失败切换：401/403、429 等明确失败可切换到下一个 Key；网络错误、5xx、空 2xx 与流中断可能已在上游执行，默认不跨 Key 重试，避免重复生成和重复计费。仅在显式开启 `routing.retry_ambiguous_errors` 后恢复此类重试。
- 容量优先级：显式逻辑路由默认严格等待当前最高优先级候选，繁忙时返回 `gateway_busy`；只有显式开启 `routing.fallback_on_busy` 才会把容量溢出到后续 Provider 或备用模型。未配置逻辑路由的原有 Key 容量切换保持不变。
- 恢复探测：高优先级 key 首次达到失败阈值后 60 秒重试；仍失败则按 `cooldown_seconds`（默认 300 秒）继续探测，成功后自动回到高优先级 key。
- OpenAI 原生协议优先：`/v1/responses` 与 `/v1/chat/completions` 分别原样请求上游同名端点；仅在上游明确返回端点不支持时自动执行双向协议转换。
- Responses / Chat Completions 转换支持流式与非流式文本、函数工具调用与 tool result、多模态文本/图片/文件、JSON Schema、reasoning、usage 和标准 SSE 生命周期；上游忽略 `stream: true` 返回 JSON 时会自动转换为 SSE。
- Anthropic Messages、Gemini generateContent 与 OpenAI Chat Completions 支持基础文本协议互转；不支持的工具、多模态或供应商专属字段会明确拒绝，不会静默丢弃。完整边界见 [协议兼容矩阵](docs/protocol-compatibility.md)。
- 余额刷新基础层：Provider 可配置 `balance_path`；`new-api` / `sub2api` 未配置时会尝试常见默认路径，例如 `/api/user/self`、`/api/user/token`、`/dashboard/billing/subscription`，并保存余额、额度、币种和归一化错误状态。
- 请求日志异步批量写入 SQLite，并对数据库持续不可写时的内存缓冲设置硬上限；`/metrics` 的 `gateway_request_logs_dropped_total` 可用于发现日志丢弃。
- 全局、Provider、Key 三级并发控制和排队超时；过载返回 `503 gateway_busy` 与 `Retry-After`，恢复冷却结束后只允许单个 half-open 探针。
- 请求日志保存有界的逐次路由轨迹，包括每次 Provider、Key、实际模型、上游协议、状态、耗时和错误类型；Responses/Chat 端点内部回退也会单独记录。
- 数据库迁移校验和、升级前快照、完整性检查、WAL checkpoint、日志筛选/分页/CSV 导出，以及带加密主密钥的跨机器便携备份。

## 配置

复制示例配置后按需修改：

```powershell
Copy-Item config.example.yaml config.yaml
```

常用环境变量：

`GATEWAY_HOST` 与 `GATEWAY_PUBLIC_HOST` 只填写主机名或 IP（不含协议、路径和端口），IPv6 地址可直接填写 `::1`。

- `GATEWAY_HOST`
- `GATEWAY_PUBLIC_HOST`
- `GATEWAY_PORT`
- `GATEWAY_ALLOW_REMOTE`
- `GATEWAY_ALLOW_INSECURE_REMOTE`
- `GATEWAY_TLS_CERT_FILE`
- `GATEWAY_TLS_KEY_FILE`
- `GATEWAY_READ_TIMEOUT_SECONDS`
- `GATEWAY_IDLE_TIMEOUT_SECONDS`
- `GATEWAY_MAX_HEADER_BYTES`
- `GATEWAY_PROXY_TOKEN`
- `GATEWAY_ADMIN_TOKEN`
- `GATEWAY_ADMIN_TOKEN_FILE`
- `GATEWAY_DB`
- `GATEWAY_SECRET_PATH`
- `GATEWAY_MASTER_KEY`：Base64 编码的 32 字节 AES 主密钥；设置后不会创建 `secret.key`。
- `GATEWAY_TIMEZONE`
- `GATEWAY_LOG_RETENTION_DAYS`
- `GATEWAY_LOG_MAX_ENTRIES`
- `GATEWAY_BACKUP_BEFORE_MIGRATION`
- `GATEWAY_BACKUP_RETENTION`
- `GATEWAY_REQUEST_LOGGING_ENABLED`
- `GATEWAY_LOG_MAX_SIZE_MB`
- `GATEWAY_LOG_MAX_BACKUPS`
- `GATEWAY_TIMEOUT_SECONDS`
- `GATEWAY_STREAM_IDLE_TIMEOUT_SECONDS`
- `GATEWAY_STREAM_WRITE_TIMEOUT_SECONDS`
- `GATEWAY_STREAM_RETRY_BEFORE_FIRST_BYTE`
- `GATEWAY_RETRY_AMBIGUOUS_ERRORS`
- `GATEWAY_ALLOW_INSECURE_UPSTREAMS`
- `GATEWAY_MAX_CONCURRENT_REQUESTS`
- `GATEWAY_MAX_CONCURRENT_PER_PROVIDER`
- `GATEWAY_MAX_CONCURRENT_PER_KEY`
- `GATEWAY_QUEUE_TIMEOUT_MILLISECONDS`
- `GATEWAY_MODEL_DISCOVERY`
- `GATEWAY_MODEL_DISCOVERY_REFRESH_HOURS`
- `GATEWAY_MODEL_DISCOVERY_TIMEOUT_SECONDS`
- `GATEWAY_OPEN_BROWSER_ON_DUPLICATE`
- `GATEWAY_TRAY`
- `GATEWAY_WORKDIR`

远程监听、备份恢复、主密钥轮换和故障处置见 [运维指南](docs/operations.md)。

## 模型发现与优先路由

1. 在 Provider 页面添加 URL 和 Key。网关会自动获取并保存模型列表，也可以点击“刷新模型”。Key 的“测试”操作会额外执行一次最小生成探针；自动发现和“刷新模型”只读取模型接口。
2. 在 Key 编辑页的“模型路由”中可搜索并勾选允许参与调度的模型；“仅推荐”会按模型系列给出快速选择。该策略作用于 Key 所属 Provider 的全部 Key；开启“仅调度已选择模型”后，新发现模型不会自动加入，空白名单会让该 Provider 暂停参与模型调度。
3. 在“模型优先路由”页面创建客户端请求的逻辑模型，例如 `coding-auto`，为其添加主模型和备用模型，并在每层选择允许使用的 Provider 与实际模型名。
4. 客户端请求 `coding-auto` 时，候选顺序为“模型优先级 -> 同优先级配置顺序 -> Provider 优先级 -> Key 顺序”。一个模型层级中每个 Provider 只能配置一个实际模型目标；同一 Provider 的第二个备用模型应放入下一模型层级。显式逻辑路由会完整遍历确定性失败的候选；`routing.retry_per_request` 继续限制未配置逻辑路由的普通 Key 切换。网络错误和 5xx 等模糊失败仍受 `routing.retry_ambiguous_errors` 保护。

逻辑路由只影响与其 ID 完全匹配的请求，不会自动接管未选择的模型。Provider 白名单同时约束直连和逻辑路由；启用路由时若目标被白名单排除，或更新白名单会破坏现有启用路由，保存操作会返回明确错误。`/v1/models` 只公布至少拥有一个启用 Provider/Key 的逻辑路由和 Provider 模型；删除最后一个目标 Provider 时，对应空路由会自动停用。

## 管理后台构建

修改 `web/admin/index.template.html`、样式或前端逻辑后运行：

```powershell
npm ci
npm run test:admin
npm run build:admin
```

生成的 `web/admin/index.html`、`assets/render.js` 和 Vue runtime-only 文件会被 Go embed 使用。CSP 不允许 `unsafe-eval`，因此不要改回运行时模板编译版本。

## 测试

```powershell
go test ./...
```

发布版本号和包名统一维护在 `release.json`。本地打包脚本默认读取该文件，发布标签必须与其中的版本号一致。

完整本地烟测：

```powershell
.\scripts\smoke.ps1
```

烟测默认使用 `127.0.0.1:28787` 和系统临时目录启动无托盘实例，验证健康检查、后台资源、安全响应头、状态接口、日志创建和重复启动退出行为。端口被占用时会直接失败，不会终止任何既有进程；可用 `-Port` 指定其他端口。
烟测还会验证管理 API 未带 Token 时返回 401，带 `GATEWAY_ADMIN_TOKEN` 时正常访问。

## License

本项目基于 [MIT License](LICENSE) 授权。

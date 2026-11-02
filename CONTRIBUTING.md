# 参与贡献

欢迎为 Local AI Gateway 添加功能、修复问题、完善兼容性或改进文档。

## 提交 Issue

提交 Issue 前请先搜索已有问题，避免重复。问题报告建议包含：

- 使用的 Gateway 版本和构建提交。
- 操作系统及架构。
- 客户端、上游类型和所用协议。
- 可以复现问题的最小步骤。
- 实际结果与预期结果。
- 已脱敏的错误信息或相关日志。

请勿提交真实 API Key、管理员 Token、数据库、主密钥或其他敏感信息。

## 提交 Pull Request

- 保持修改范围清晰，不夹带无关重构或生成文件。
- 新功能和问题修复应补充与风险相称的测试。
- 行为、配置或兼容性发生变化时同步更新相关文档。
- 提交前运行适用于本次修改的检查，并说明未执行的验证。

## 本地开发

后端需要 Go 1.25 或更高版本。`go.mod` 指定 Go 1.26.10 工具链；测试和发行构建请设置 `GOTOOLCHAIN=go1.26.10`，避免本机较新的 Go 覆盖指定版本。

`scripts/package-beta.ps1` 固定使用 `go.mod` 的工具链并构建 Windows amd64、Linux amd64/arm64。默认要求干净工作区；`-AllowDirty` 仅用于本地验证，会标记 `commit` 为 `-dirty`。`SOURCE-MANIFEST.txt` 按实际工作区文件内容计算（含新增源码，不含已删除文件、忽略的数据和本地 agent 指令），`VERSION.json` 的 `sourceMode` / `sourceDirty` 明确来源；换行不同可能导致跨平台源码摘要不同。不要把未提交的构建包当成已发布标签对应的正式制品。

运行后端：

```bash
go run ./cmd/gateway
```

执行 Go 检查：

```bash
go test ./...
go vet ./...
go mod verify
```

只有修改管理后台模板、样式或前端逻辑时才需要 Node.js：

```bash
npm ci
npm run test:admin
npm run build:admin
```

生成的 `web/admin/index.html`、`web/admin/assets/render.js` 和 Vue runtime-only 文件由 Go embed 使用。管理后台 CSP 不允许 `unsafe-eval`，请勿改回运行时模板编译版本。

Windows 完整烟测：

```powershell
.\scripts\smoke.ps1
```

烟测使用系统临时目录启动隔离实例，不会终止已有 Gateway 或其他未知进程。端口被占用时会直接失败，可使用 `-Port` 指定其他端口。

## 发布元数据

版本号和包名统一维护在 `release.json`。本地打包脚本默认读取该文件，发布标签必须与其中的版本号一致。

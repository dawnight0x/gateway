# Local AI Gateway

Local AI Gateway 是一个本地 AI API 网关，管理后台已经内嵌在程序中，无需安装 Node.js、Go 或其他运行环境。

## 系统要求

- Windows 10/11 x64
- Linux amd64 或 arm64

## 包内容

- `gateway` / `gateway.exe`：网关主程序
- `gateway-backup` / `gateway-backup.exe`：离线恢复与主密钥轮换工具
- `config.example.yaml`：可选配置模板；不创建 `config.yaml` 也可使用安全默认值启动
- `docs/`：Linux 部署、运维和协议兼容说明
- `VERSION.json`、`SOURCE-MANIFEST.txt`、`sbom.cdx.json`、`SHA256SUMS`：版本、来源和完整性信息

## Windows

双击 `gateway.exe` 启动。程序会显示在系统托盘中，可通过托盘菜单打开管理后台或退出程序。

## Linux

首次运行前赋予执行权限：

```bash
chmod +x gateway gateway-backup
./gateway
```

需要在后台运行时，可使用 systemd、supervisord 等进程管理工具。

## 首次使用

启动后访问：

- 管理后台：`http://localhost:18787/admin`
- OpenAI Base URL：`http://localhost:18787/v1`
- Anthropic Base URL：`http://localhost:18787`
- Gemini Base URL：`http://localhost:18787`

首次启动会在程序所在目录创建 `data` 文件夹。后台管理员 Token 保存在 `data/admin.token`，在管理后台登录时使用该 Token。

登录后先添加 Provider 和上游 Key，等待模型列表自动同步；需要统一模型入口时，在“模型优先路由”页面配置逻辑模型及备用顺序。最后在“网关 Key”页面创建供客户端使用的 `sk-...` Key。真实上游 Key 只保存在本机网关中。

## 数据与升级

数据库、加密密钥、管理员 Token 和日志均保存在 `data` 文件夹。升级程序前请先停止网关并备份该文件夹，然后用新版执行程序替换旧版；不要删除或覆盖原有 `data` 文件夹。

默认仅监听 `127.0.0.1`，不会直接开放到局域网或公网。

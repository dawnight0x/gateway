# 运维指南

## 安全暴露服务

默认只监听 `127.0.0.1`。如需监听局域网地址，必须设置 `server.allow_remote: true`，并同时配置 `tls_cert_file` 与 `tls_key_file`。`allow_insecure_remote` 只适用于已有独立加密隧道的受信任网络，不应直接用于公网。

非回环上游默认必须使用 HTTPS。仅在上游位于隔离网络且风险已被接受时开启 `routing.allow_insecure_upstreams`。

## 备份类型

- 数据库快照 `.db`：适合同一台机器快速回滚。它不包含主密钥，不能单独迁移到另一台机器。
- 加密便携备份 `.zip`：包含一致性数据库快照与经口令加密的主密钥，可跨机器恢复。口令至少 12 个字节，遗失后无法恢复。
- 自动迁移快照：数据库 schema 升级前写入 `data/backups`，保留数量由 `storage.backup_retention` 控制。

创建便携备份可在管理后台“路由设置 / 数据维护”完成。备份前建议先执行数据库完整性检查。

## 离线恢复

恢复时必须先停止网关，避免覆盖正在使用的 SQLite 文件。构建恢复工具：

```powershell
.\.tools\go\bin\go.exe build -buildvcs=false -o .\bin\gateway-backup.exe .\cmd\gateway-backup
$env:GATEWAY_BACKUP_PASSPHRASE = "your-backup-passphrase"
.\bin\gateway-backup.exe restore --input .\gateway-portable-xxxx.zip --database .\data\gateway.db --secret .\data\secret.key
```

目标文件已存在时命令默认拒绝执行。确认网关已停止后使用 `--force`，工具会先生成带 `before-restore` 时间戳的旧文件副本，再写入恢复内容。恢复后启动网关并执行管理后台完整性检查与上游 Key 测试。

## 主密钥轮换

轮换必须离线进行。命令会先创建 `pre-key-rotation` 数据库快照，再在单个事务内重加密所有上游 Key，并在主密钥文件中保留上一把密钥作为崩溃恢复和旧快照回退用途：

```powershell
.\bin\gateway-backup.exe rotate-key --database .\data\gateway.db --secret .\data\secret.key
```

使用外部 `GATEWAY_MASTER_KEY` 时该命令会拒绝执行。外部密钥轮换需要由密钥管理系统提供旧、新密钥并安排离线迁移。

## 可观测性

`/metrics` 提供 Key 数量、冷却状态、今日请求/Token、请求日志丢弃、当前并发、上游尝试、重试和过载拒绝指标。请求可通过响应头 `X-Gateway-Request-ID` 与后台请求日志关联。

过载返回 `503 gateway_busy` 和 `Retry-After`。先检查 `gateway_in_flight_requests` 与三级并发配置，再决定扩容 Key、缩短上游超时或提高限制。不要仅通过开启模糊重试处理超时；该选项可能导致重复生成与重复计费。

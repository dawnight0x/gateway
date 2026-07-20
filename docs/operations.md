# 运维指南

## 安全暴露服务

默认只监听 `127.0.0.1`。如需监听局域网地址，必须设置 `server.allow_remote: true`，并同时配置 `tls_cert_file` 与 `tls_key_file`。`allow_insecure_remote` 只适用于已有独立加密隧道的受信任网络，不应直接用于公网。

非回环上游默认必须使用 HTTPS。仅在上游位于隔离网络且风险已被接受时开启 `routing.allow_insecure_upstreams`。

## 备份类型

- 数据库快照 `.db`：适合同一台机器快速回滚。它不包含主密钥，不能单独迁移到另一台机器。
- 加密便携备份 `.zip`：v2 使用 AES-256-GCM 同时加密数据库和主密钥，并认证 manifest，可跨机器恢复。口令至少 12 个字节，遗失后无法恢复；恢复工具继续兼容旧 v1 备份。
- 自动迁移快照：数据库 schema 升级前写入 `data/backups`，保留数量由 `storage.backup_retention` 控制。

创建便携备份可在管理后台“路由设置 / 数据维护”完成。备份前建议先执行数据库完整性检查。

## 离线恢复

恢复时必须先停止网关，避免覆盖正在使用的 SQLite 文件。发行包已经包含恢复工具。

Windows：

```powershell
$env:GATEWAY_BACKUP_PASSPHRASE = "your-backup-passphrase"
.\gateway-backup.exe restore --input .\gateway-portable-xxxx.zip --database .\data\gateway.db --secret .\data\secret.key
```

Linux：

```bash
export GATEWAY_BACKUP_PASSPHRASE='your-backup-passphrase'
./gateway-backup restore --input ./gateway-portable-xxxx.zip --database ./data/gateway.db --secret ./data/secret.key
```

目标文件已存在时命令默认拒绝执行。确认网关已停止后使用 `--force`，工具会先生成带 `before-restore` 时间戳的旧文件副本，再写入恢复内容。数据库与主密钥通过恢复事务日志提交；进程或系统在双文件替换期间中断时，网关下次启动会先自动回滚未完成的恢复。恢复后执行管理后台完整性检查与上游 Key 测试。

## 主密钥轮换

轮换必须离线进行。命令会先创建 `pre-key-rotation` 数据库快照，再在单个事务内重加密所有上游 Key，并在主密钥文件中保留上一把密钥作为崩溃恢复和旧快照回退用途：

```powershell
.\gateway-backup.exe rotate-key --database .\data\gateway.db --secret .\data\secret.key
```

使用外部 `GATEWAY_MASTER_KEY` 时该命令会拒绝执行。外部密钥轮换需要由密钥管理系统提供旧、新密钥并安排离线迁移。

## 可观测性

`/metrics` 提供 Key 数量、冷却状态、今日请求/Token、请求日志丢弃、当前并发、上游尝试、重试和过载拒绝指标。今日请求/Token 使用独立日聚合表，即使关闭详细请求日志也会保留。请求可通过响应头 `X-Gateway-Request-ID` 与后台请求日志关联。后台日志保存逐次路由轨迹，可展开查看每次 Provider、Key、实际上游模型、上游协议、状态、耗时和错误类型；轨迹最多保存 256 条并明确标记截断。

过载返回 `503 gateway_busy` 和 `Retry-After`。先检查 `gateway_in_flight_requests` 与三级并发配置，再决定扩容 Key、缩短上游超时或提高限制。不要仅通过开启模糊重试处理超时；该选项可能导致重复生成与重复计费。

## 模型发现与路由故障处置

自动模型发现由 `model_discovery` 配置控制，只对启用且拥有可用 Key 的 Provider 执行模型列表读取。发现任务会跟随 Gemini、Anthropic 等上游的分页游标，读取同 Provider 的全部启用 Key，并把各 Key 的成功结果合并显示。单个 Key 刷新失败或返回空列表不会删除该 Key 上次成功库存；后台会保留最后成功时间并显示本次错误。Key 编辑页的“模型路由”支持搜索、全选、仅推荐和清空模型；该策略作用于 Key 所属 Provider 的全部 Key。开启模型白名单后，仅选中模型参与直连和逻辑路由，新发现模型不会自动加入。空白名单表示该 Provider 暂停参与模型调度。

逻辑模型按“模型优先级、同优先级配置顺序、Provider 优先级、Key 顺序”生成候选。一个模型层级中每个 Provider 只能配置一个实际模型；同一 Provider 的其他备用模型放入后续模型层级。上游明确表示模型不存在或不支持时，冷却对应 Provider/模型组合并覆盖该 Provider 当前及后来新增的所有 Key；模型权限错误仍只冷却失败的 Key/模型组合。Provider 白名单同时约束直连和逻辑路由，启用路由目标与白名单冲突时会在保存路由或 Provider 时明确拒绝。401/403、429、网络和服务错误按现有 Key 健康策略处理。显式逻辑路由会完整遍历确定性失败的候选；`routing.retry_per_request` 限制普通 Key 路由。显式逻辑路由默认不会因高优先级候选达到并发上限而降级；需要容量溢出时开启 `routing.fallback_on_busy`。未配置逻辑路由的原有 Key 容量切换保持不变。后台模型路由页面会区分 Provider 模型冷却与 Key 模型冷却，并可手动重置目标的全部模型健康状态。

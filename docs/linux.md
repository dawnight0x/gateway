# Linux 部署

Linux amd64 分发包包含静态编译的 `gateway` 和离线维护工具 `gateway-backup`，不依赖 Node.js 或桌面托盘组件。

## 直接运行

```bash
chmod +x gateway gateway-backup
./gateway
```

程序默认以前台方式监听 `127.0.0.1:18787`。首次启动会在当前目录生成 `data/gateway.db`、主密钥、管理员 Token 和运行日志。管理员 Token 位于 `data/admin.token`。

## systemd

将分发目录放到 `/opt/local-ai-gateway`，确保服务用户对该目录拥有读写权限，然后创建 `/etc/systemd/system/local-ai-gateway.service`：

```ini
[Unit]
Description=Local AI Gateway
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=gateway
Group=gateway
WorkingDirectory=/opt/local-ai-gateway
ExecStart=/opt/local-ai-gateway/gateway
Environment=GATEWAY_TRAY=false
Restart=on-failure
RestartSec=3
NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now local-ai-gateway
sudo systemctl status local-ai-gateway
```

## 配置与维护

默认配置即可本机使用。需要自定义时先执行：

```bash
cp config.example.yaml config.yaml
```

校验分发文件：

```bash
sha256sum -c SHA256SUMS
```

便携备份恢复与主密钥轮换见 `docs/operations.md`。恢复或轮换前必须停止网关服务。

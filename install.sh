#!/bin/bash
# Go-Proxy One-Click Deployment & Tuning Script
# Targets: Ubuntu/Debian/CentOS
# Run with root privileges

echo "==================================================="
echo "  Go-Proxy 高并发矿池代理 - 一键部署与系统优化脚本"
echo "==================================================="

# 1. 检查 root 权限
if [ "$EUID" -ne 0 ]; then
  echo "[错误] 请使用 root 权限运行此脚本 (sudo bash install.sh)"
  exit 1
fi

# 2. 优化系统内核参数 (sysctl)
echo "[1/4] 正在优化系统内核参数，解除高并发网络拥堵..."
cat >> /etc/sysctl.conf << EOF

# ==== Go-Proxy Tuning ====
fs.file-max = 1000000
net.core.somaxconn = 65535
net.ipv4.tcp_tw_reuse = 1
net.ipv4.ip_local_port_range = 10000 65000
net.ipv4.tcp_max_syn_backlog = 65535
net.ipv4.tcp_fin_timeout = 15
EOF
sysctl -p > /dev/null 2>&1
echo "  -> 内核参数优化完成！"

# 3. 提升最大文件描述符 (ulimit)
echo "[2/4] 正在解除 Linux 最大并发连接数 (突破 65535 限制)..."
cat >> /etc/security/limits.conf << EOF

* soft nofile 1000000
* hard nofile 1000000
root soft nofile 1000000
root hard nofile 1000000
EOF

# 修改 systemd 的全局句柄限制
if [ -f "/etc/systemd/system.conf" ]; then
    sed -i 's/#DefaultLimitNOFILE=/DefaultLimitNOFILE=1000000/g' /etc/systemd/system.conf
fi
echo "  -> 并发限制解除完成！(支持百万级连接)"

# 4. 部署 Systemd 守护进程
echo "[3/4] 正在配置系统级守护进程 (防崩溃自动重启)..."
WORK_DIR=$(pwd)
PROXY_BIN="$WORK_DIR/proxy"

cat > /etc/systemd/system/go-proxy.service << EOF
[Unit]
Description=Go-Proxy Transparent Mining Proxy
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=$WORK_DIR
ExecStart=$PROXY_BIN
Restart=always
RestartSec=3
LimitNOFILE=1000000

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable go-proxy > /dev/null 2>&1
echo "  -> 守护进程注册完成！(开机自动启动已开启)"

# 5. 完成提示
echo "==================================================="
echo "[4/4] 部署环境准备完毕！"
echo ""
echo "⚠️ 下一步操作指南："
echo "1. 请确保您已经将 Linux 版本的 'proxy' 程序上传到了当前目录: $WORK_DIR"
echo "   (如果在 Windows 上，可以在源码目录打开终端运行: GOOS=linux GOARCH=amd64 go build -o proxy cmd/proxy/main.go 进行交叉编译)"
echo "2. 给程序赋予执行权限: chmod +x proxy"
echo "3. 启动代理引擎: systemctl start go-proxy"
echo ""
echo "常用维护命令："
echo "- 启动: systemctl start go-proxy"
echo "- 停止: systemctl stop go-proxy"
echo "- 重启: systemctl restart go-proxy"
echo "- 状态: systemctl status go-proxy"
echo "==================================================="

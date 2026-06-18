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

# 2. 基础组件与时间同步 (NTP)
echo "[1/7] 正在安装基础网络组件并同步全球时间..."
if command -v apt-get >/dev/null 2>&1; then
    apt-get update -y >/dev/null 2>&1
    apt-get install -y wget curl ufw chrony tzdata >/dev/null 2>&1
    systemctl enable chrony >/dev/null 2>&1
    systemctl restart chrony >/dev/null 2>&1
elif command -v yum >/dev/null 2>&1; then
    yum install -y wget curl firewalld chrony tzdata >/dev/null 2>&1
    systemctl enable chronyd >/dev/null 2>&1
    systemctl restart chronyd >/dev/null 2>&1
fi
# 强制设为东八区/UTC，保证与矿池一致
timedatectl set-timezone Asia/Shanghai >/dev/null 2>&1
if command -v chronyc >/dev/null 2>&1; then
    chronyc -a makestep >/dev/null 2>&1
fi
echo "  -> 时间强制同步完成！(防止挖出过期 Stale 份额)"

# 3. 交互式配置 Web 端口
echo "[2/7] 正在配置控制台端口..."
while true; do
  read -p "请输入您想要的网页控制台端口 (默认 8080): " WEB_PORT
  WEB_PORT=${WEB_PORT:-8080}
  
  if ! [[ "$WEB_PORT" =~ ^[0-9]+$ ]] || [ "$WEB_PORT" -lt 1 ] || [ "$WEB_PORT" -gt 65535 ]; then
    echo "[错误] 端口必须是 1 - 65535 之间的数字！"
    continue
  fi
  
  # 检查端口占用
  if command -v ss >/dev/null 2>&1; then
    if ss -tuln | grep -E ":$WEB_PORT\b" > /dev/null; then
      echo "[错误] 拒绝使用！检测到端口 $WEB_PORT 已被系统中其他程序占用，请换一个！"
      continue
    fi
  elif command -v netstat >/dev/null 2>&1; then
    if netstat -tuln | grep -E ":$WEB_PORT\b" > /dev/null; then
      echo "[错误] 拒绝使用！检测到端口 $WEB_PORT 已被系统中其他程序占用，请换一个！"
      continue
    fi
  fi
  
  echo "  -> 网页控制台端口将使用: $WEB_PORT"
  break
done

# 4. 防火墙自动放行
echo "[3/7] 正在自动配置防火墙放行策略..."
if command -v ufw >/dev/null 2>&1; then
    ufw allow $WEB_PORT/tcp >/dev/null 2>&1
    echo "  -> UFW 防火墙放行 $WEB_PORT 成功！"
elif command -v firewall-cmd >/dev/null 2>&1; then
    firewall-cmd --zone=public --add-port=$WEB_PORT/tcp --permanent >/dev/null 2>&1
    firewall-cmd --reload >/dev/null 2>&1
    echo "  -> Firewalld 防火墙放行 $WEB_PORT 成功！"
else
    echo "  -> 未检测到默认防火墙，已跳过。"
fi

# 5. 开启 Google BBR 拥塞控制
echo "[4/7] 正在开启 Google BBR 拥塞控制算法 (极大降低跨国丢包率)..."
sed -i '/net.core.default_qdisc/d' /etc/sysctl.conf
sed -i '/net.ipv4.tcp_congestion_control/d' /etc/sysctl.conf
echo "net.core.default_qdisc=fq" >> /etc/sysctl.conf
echo "net.ipv4.tcp_congestion_control=bbr" >> /etc/sysctl.conf
sysctl -p > /dev/null 2>&1
BBR_STATUS=$(sysctl net.ipv4.tcp_congestion_control | awk '{print $3}' 2>/dev/null)
if [[ "$BBR_STATUS" == *"bbr"* ]]; then
    echo "  -> BBR 加速开启成功！"
else
    echo "  -> BBR 加速开启失败 (您的内核可能过旧，但系统会继续安装)。"
fi

# 6. 优化系统内核参数 (sysctl & ulimit)
echo "[5/7] 正在优化系统内核与并发参数，解除高并发网络拥堵..."
sed -i '/# ==== Go-Proxy Tuning ====/,+7d' /etc/sysctl.conf 2>/dev/null || true
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

sed -i '/# ==== Go-Proxy Limits ====/,$d' /etc/security/limits.conf 2>/dev/null || true
cat >> /etc/security/limits.conf << EOF

# ==== Go-Proxy Limits ====
* soft nofile 1000000
* hard nofile 1000000
root soft nofile 1000000
root hard nofile 1000000
EOF

if [ -f "/etc/systemd/system.conf" ]; then
    sed -i 's/#DefaultLimitNOFILE=.*/DefaultLimitNOFILE=1000000/g' /etc/systemd/system.conf
fi
echo "  -> 并发限制解除完成！(支持百万级无感并发)"

# 7. 全自动拉取与部署 Systemd
echo "[6/7] 正在拉取最新版代理引擎并注册系统服务..."
WORK_DIR="/root/go-proxy"
PROXY_BIN="$WORK_DIR/proxy"
mkdir -p $WORK_DIR
cd $WORK_DIR

echo "  -> 正在从云端拉取最新版 proxy 程序 (请确保网络畅通)..."
# 自动检测是否为 beta 分支或 main 分支，此处默认为主仓库占位
# 未来发布 Release 时将使用最新版的 CDN 链接
if wget -q --timeout=15 -O proxy "https://github.com/niuniu06/MinerLink-Proxy/releases/latest/download/MinerLink-Proxy-linux-amd64"; then
    chmod +x proxy
    echo "  -> 核心引擎下载成功！"
else
    echo "  [提示] 自动下载失败（可能是国内网络受限或暂未发布 Release）。"
    echo "  [提示] 稍后请您自行通过 SFTP 将编译好的 proxy 放入 $WORK_DIR 目录并执行 chmod +x proxy"
fi

cat > /etc/systemd/system/go-proxy.service << EOF
[Unit]
Description=Go-Proxy Transparent Mining Proxy
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=$WORK_DIR
ExecStart=$PROXY_BIN -api-port $WEB_PORT
Restart=always
RestartSec=3
LimitNOFILE=1000000

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable go-proxy > /dev/null 2>&1
systemctl restart go-proxy > /dev/null 2>&1
echo "  -> 守护进程注册完成并已尝试启动！"

# 8. 完成提示
echo "==================================================="
echo "[7/7] 🎉 Go-Proxy 终极环境部署完毕！"
echo ""
echo "👉 您的控制台地址: http://您的云服务器公网IP:$WEB_PORT"
echo ""
if [ ! -x "$PROXY_BIN" ]; then
echo "⚠️ [注意] 您当前的 $WORK_DIR 目录下还没有可执行的 proxy 程序！"
echo "    请您在 Windows 源码目录通过 'GOOS=linux GOARCH=amd64 go build -o proxy ./cmd/proxy' 编译"
echo "    然后将 proxy 文件上传到服务器的 $WORK_DIR 目录，最后执行："
echo "    chmod +x /root/go-proxy/proxy && systemctl restart go-proxy"
fi
echo ""
echo "常用维护命令："
echo "- 启动: systemctl start go-proxy"
echo "- 停止: systemctl stop go-proxy"
echo "- 重启: systemctl restart go-proxy"
echo "- 查看状态: systemctl status go-proxy"
echo "- 查看实时日志: journalctl -u go-proxy -f"
echo "==================================================="

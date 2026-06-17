#!/bin/bash

# MinerLink-Proxy One-Click Deployment & Tuning Script



echo "==================================================="

echo "  MinerLink-Proxy 商业稳定版 - 一键部署与系统优化脚本"

echo "==================================================="



if [ "$EUID" -ne 0 ]; then

  echo "[错误] 请使用 root 权限运行此脚本 (sudo bash install.sh)"

  exit 1

fi



echo "[1/7] 正在安装基础网络组件并同步全球时间..."

if command -v apt-get >/dev/null 2>&1; then

    apt-get update -y >/dev/null 2>&1

    apt-get install -y unzip wget curl ufw chrony tzdata >/dev/null 2>&1

    systemctl enable chrony >/dev/null 2>&1

    systemctl restart chrony >/dev/null 2>&1

elif command -v yum >/dev/null 2>&1; then

    yum install -y unzip wget curl firewalld chrony tzdata >/dev/null 2>&1

    systemctl enable chronyd >/dev/null 2>&1

    systemctl restart chronyd >/dev/null 2>&1

fi



timedatectl set-timezone Asia/Shanghai >/dev/null 2>&1

if command -v chronyc >/dev/null 2>&1; then

    chronyc -a makestep >/dev/null 2>&1

fi

echo "  -> 时间强制同步完成！(防止挖出过期 Stale 份额)"



echo "[2/7] 正在配置控制台端口..."

while true; do

  read -p "请输入您想要的网页控制台端口 (默认 10010): " WEB_PORT

  WEB_PORT=${WEB_PORT:-10010}

  

  if ! [[ "$WEB_PORT" =~ ^[0-9]+$ ]] || [ "$WEB_PORT" -lt 1 ] || [ "$WEB_PORT" -gt 65535 ]; then

    echo "[错误] 端口必须是 1 - 65535 之间的数字！"

    continue

  fi

  

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



echo "[4/7] 正在开启 Google BBR 拥塞控制算法..."

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



echo "[5/7] 正在优化系统内核与并发参数..."

sed -i '/# ==== MinerLink Tuning ====/,+7d' /etc/sysctl.conf 2>/dev/null || true

cat >> /etc/sysctl.conf << EOF



# ==== MinerLink Tuning ====

fs.file-max = 1000000

net.core.somaxconn = 65535

net.ipv4.tcp_tw_reuse = 1

net.ipv4.ip_local_port_range = 10000 65000

net.ipv4.tcp_max_syn_backlog = 65535

net.ipv4.tcp_fin_timeout = 15

EOF

sysctl -p > /dev/null 2>&1



sed -i '/# ==== MinerLink Limits ====/,\$d' /etc/security/limits.conf 2>/dev/null || true

cat >> /etc/security/limits.conf << EOF



# ==== MinerLink Limits ====

* soft nofile 1000000

* hard nofile 1000000

root soft nofile 1000000

root hard nofile 1000000

EOF



if [ -f "/etc/systemd/system.conf" ]; then

    sed -i 's/#DefaultLimitNOFILE=.*/DefaultLimitNOFILE=1000000/g' /etc/systemd/system.conf

fi

echo "  -> 并发限制解除完成！(支持百万级无感并发)"



echo "[6/7] 正在拉取最新版代理引擎并注册系统服务..."

WORK_DIR="/root/MinerLink-Proxy"

PROXY_BIN="$WORK_DIR/MinerLink-Proxy-linux-amd64"

mkdir -p $WORK_DIR

cd $WORK_DIR



echo "  -> 正在从 Github 云端拉取最新版 MinerLink-Proxy 程序 (请确保网络畅通)..."

if wget -q --timeout=30 -O MinerLink-Proxy-Linux.zip "https://github.com/niuniu06/MinerLink-Proxy/releases/latest/download/MinerLink-Proxy-Linux.zip"; then

    unzip -o MinerLink-Proxy-Linux.zip

    chmod +x MinerLink-Proxy-linux-amd64

    rm -f MinerLink-Proxy-Linux.zip

    echo "  -> 核心引擎下载并解压成功！"

else

    echo "  [错误] 自动下载失败！可能是国内网络受限无法访问 Github Release。"

    echo "  [解决] 您可以自行将压缩包上传到 $WORK_DIR 目录解压，然后手动启动。"

    exit 1

fi



cat > /etc/systemd/system/minerlink-proxy.service << EOF

[Unit]

Description=MinerLink-Proxy Transparent Mining Proxy

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

systemctl enable minerlink-proxy > /dev/null 2>&1

systemctl restart minerlink-proxy > /dev/null 2>&1

echo "  -> 守护进程注册完成并已成功启动！"



echo "==================================================="

echo "[7/7] ?? MinerLink-Proxy 终极环境部署完毕！"

echo ""

echo "?? 您的控制台地址: http://您的云服务器公网IP:$WEB_PORT/ui/"

echo ""

echo "常用维护命令："

echo "- 启动: systemctl start minerlink-proxy"

echo "- 停止: systemctl stop minerlink-proxy"

echo "- 重启: systemctl restart minerlink-proxy"

echo "- 查看状态: systemctl status minerlink-proxy"

echo "- 查看实时日志: journalctl -u minerlink-proxy -f"

echo "==================================================="
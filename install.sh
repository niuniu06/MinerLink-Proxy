#!/bin/bash
# ==============================================================================
# Go-Proxy 一键安装与系统优化脚本 (Ubuntu / Debian / CentOS 兼容)
# 作者: Antigravity AI
# 功能: 环境搭建、Go语言安装、极致网络优化、内核级调优、Systemd进程守护
# ==============================================================================

# 确保以 root 权限运行
if [ "$EUID" -ne 0 ]; then
  echo -e "\033[31m错误: 请使用 root 权限运行此脚本! (可以尝试加上 sudo)\033[0m"
  exit 1
fi

echo -e "\033[36m[1/7] 请设置网页控制台的监听端口 (默认: 8080):\033[0m"
read -p "请输入端口号: " WEB_PORT
if [ -z "$WEB_PORT" ]; then
    WEB_PORT=8080
fi
echo -e "网页控制台端口已设置为: \033[32m$WEB_PORT\033[0m"
echo ""

echo -e "\033[36m[1.5/7] (重要) 因为仓库是私密库，请输入您的 GitHub Personal Access Token (PAT):\033[0m"
echo -e "如果仓库是公开的，请直接按回车跳过。"
read -p "请输入 Token (ghp_xxxx...): " GITHUB_TOKEN

echo -e "\033[36m[2/7] 正在初始化环境并安装系统依赖...\033[0m"
if command -v apt-get >/dev/null; then
    apt-get update -y
    apt-get install -y wget curl git build-essential ufw tar
elif command -v yum >/dev/null; then
    yum update -y
    yum install -y wget curl git gcc make ufw tar
else
    echo -e "\033[33m警告: 未知包管理器，跳过依赖安装。\033[0m"
fi

echo -e "\033[36m[3/7] 正在执行 Linux 内核级挖矿网络优化 (TCP/BBR/高并发)...\033[0m"
cat > /etc/sysctl.d/99-goproxy.conf << 'EOF'
# 提升文件句柄上限 (解决并发连接过多报错)
fs.file-max = 1000000
fs.inotify.max_user_instances = 8192

# 开启 TCP BBR 拥塞控制算法 (极大降低延迟，提高跨国矿池连通率)
net.core.default_qdisc = fq
net.ipv4.tcp_congestion_control = bbr

# 优化 TCP 端口重用与快速回收 (防止大量 TIME_WAIT 卡死)
net.ipv4.tcp_tw_reuse = 1
net.ipv4.ip_local_port_range = 1024 65535

# 增大连接队列 (抵抗高并发节点连接涌入)
net.core.somaxconn = 65535
net.ipv4.tcp_max_syn_backlog = 65535
net.ipv4.tcp_max_tw_buckets = 20000

# 调整 TCP Keepalive 探测机制 (快速剔除死连接矿机)
net.ipv4.tcp_keepalive_time = 600
net.ipv4.tcp_keepalive_intvl = 30
net.ipv4.tcp_keepalive_probes = 3
EOF

# 应用系统参数
sysctl -p /etc/sysctl.d/99-goproxy.conf >/dev/null 2>&1

# 提升 Ulimit 会话级文件句柄
cat > /etc/security/limits.d/goproxy.conf << 'EOF'
* soft nofile 1000000
* hard nofile 1000000
root soft nofile 1000000
root hard nofile 1000000
EOF
ulimit -n 1000000

echo -e "\033[36m[4/7] 正在安装 Go 语言编译环境...\033[0m"
if ! command -v go >/dev/null 2>&1; then
    GO_VERSION="1.22.4"
    echo "下载 Go $GO_VERSION ..."
    wget -q https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz
    rm -rf /usr/local/go
    tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz
    rm go${GO_VERSION}.linux-amd64.tar.gz
    
    # 写入环境变量
    echo 'export PATH=$PATH:/usr/local/go/bin' > /etc/profile.d/golang.sh
    source /etc/profile.d/golang.sh
else
    echo -e "\033[32m系统已安装 Go 环境: $(go version)\033[0m"
fi

export PATH=$PATH:/usr/local/go/bin

echo -e "\033[36m[5/7] 正在拉取 Go-Proxy 源码并编译...\033[0m"
INSTALL_DIR="/opt/go-proxy"

# 构造带有 Token 的 Git URL
if [ -n "$GITHUB_TOKEN" ]; then
    GIT_URL="https://${GITHUB_TOKEN}@github.com/yao52069/go-proxy.git"
else
    GIT_URL="https://github.com/yao52069/go-proxy.git"
fi

if [ -d "$INSTALL_DIR" ]; then
    echo "检测到旧版本，正在更新代码..."
    cd $INSTALL_DIR
    git remote set-url origin $GIT_URL
    git pull origin main
else
    git clone $GIT_URL $INSTALL_DIR
    cd $INSTALL_DIR
fi

# 设置 Go 代理以防国内机器拉取依赖失败
go env -w GO111MODULE=on
go env -w GOPROXY=https://goproxy.cn,direct

# 编译本体 (无需重新编译前端，因为打包时已内嵌静态资源)
echo "开始编译代理内核..."
go build -ldflags="-w -s" -o proxy.bin ./cmd/proxy
chmod +x proxy.bin

echo -e "\033[36m[6/7] 正在配置 Systemd 后台进程守护...\033[0m"
cat > /etc/systemd/system/go-proxy.service << EOF
[Unit]
Description=Go Stratum Proxy High-Performance Engine
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/proxy.bin -api-port $WEB_PORT
Restart=always
RestartSec=3
LimitNOFILE=1000000

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable go-proxy.service
systemctl restart go-proxy.service

echo -e "\033[36m[7/7] 正在放行控制台网络端口...\033[0m"
if command -v ufw >/dev/null; then
    ufw allow $WEB_PORT/tcp >/dev/null 2>&1
    echo "已放行防火墙 $WEB_PORT 端口。"
fi

echo -e "=============================================================================="
echo -e "\033[32m部署完美完成！\033[0m"
echo -e "Go-Proxy 代理引擎已在后台以极速模式运行中。"
echo -e ""
echo -e "控制台访问地址: \033[33mhttp://<你的服务器IP>:$WEB_PORT\033[0m"
echo -e "运行状态查看: \033[36msystemctl status go-proxy\033[0m"
echo -e "实时日志查看: \033[36mjournalctl -u go-proxy -f\033[0m"
echo -e "重启代理服务: \033[36msystemctl restart go-proxy\033[0m"
echo -e "停止代理服务: \033[36msystemctl stop go-proxy\033[0m"
echo -e "=============================================================================="

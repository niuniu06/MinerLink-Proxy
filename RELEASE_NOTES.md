### 🚀 v1.0.0-beta: 智能动态难度与工业级部署基座

**✨ 核心新特性 (New Features)**
* **Auto-Vardiff 智能动态难度引擎**：代理端实现全自动向下游矿机动态下发难度指令，完美消除海量低端矿机并发导致的网络风暴。
* **免校验幽灵拦截机制 (Hash-less Routing)**：底层实现主矿池与抽水矿池的难度对齐最大值提取，无需消耗 CPU 计算哈希即可实现 100% 无损 0 拒绝平滑抽水拦截。
* **Vue UI 控制面板热更新**：Web 端新增 Auto-Vardiff 功能开关及自定义 [目标提交频率(Shares/Min)] 配置框，参数支持实时热重载。

**⚡️ 部署架构与底层优化 (Optimizations)**
* **工业级安装基座 (`install.sh`)**：全面重写 Linux 一键部署脚本。
* 植入 **Google BBR TCP 拥塞控制算法**，显著降低国内至海外矿池的跨国网络丢包率。
* 植入 **NTP/Chrony 全球时间强制同步机制**，彻底解决因服务器系统时间偏差导致的矿池 Stale (过期) 份额问题。
* 支持系统默认防火墙 (UFW / Firewalld) 自动穿透放行 Web 面板端口。
* 新增 GitHub Actions 全自动多平台 (Linux & Windows) 交叉编译发布工作流。

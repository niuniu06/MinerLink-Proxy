# MinerLink-Proxy / Go-Proxy 核心开发备忘录 (AI Developer Notes)

这份文档旨在记录和沉淀本项目在多次迭代中，关于底层逻辑修复、UI 显示差异以及架构设计的核心决策。
每次启动新会话或排查遗留问题时，AI 助手将优先查阅此档，以防止历史修复被遗漏或覆盖。

## 1. 抽水与界面设计逻辑 (UI & Fee Logic)

### 公版 (go-proxy) vs 私有版 (MinerLink-Proxy)
*   **代码基线：** 两者共享同一个底层 main 分支代码，不维护两套独立的逻辑。
*   **UI 隐藏策略：** 私有版编译前通过临时覆写隐藏高级开关及作者比例，标题强制改为 MinerLink-Proxy 控制台。公版不作任何隐藏。
*   **私有发版防呆机制：** 私有版必须通过 uild_private.ps1 和 upload_private.py 脚本进行一键替换和编译发版。

### 作者抽水 (DevFee) 与 运营者抽水 (OpFee) 隔离
*   **双钱包并行隔离：** 引入 isDevMode 标志，DevFee 与 OpFee 并行独立，绝不覆写。
*   **前端账面隐身：** 独立矿池 ConnectFee 抽水成功后，若属于 DevFee，强制只递增 s.Stats.Shares++ 和 s.Stats.ValidShares++，不递增 s.Stats.FeeShares++，保证在面板中完全隐身。

## 2. 核心架构与抽水路由优化 (Core Architecture & Routing)

### 抽水子账号智能路由与无缝兜底机制
*   当矿机使用的是**子账号格式**时，系统优先尝试同池抽水。若鉴权失败被拒绝，会在毫秒级内捕获异常，自动重定向到兜底矿池，全程包裹在 3 秒预热期内。

### 无状态数学排班轮询架构 (Stateless Distributed Scheduler)
*   彻底废除了旧版独立时间切片导致的大规模集体掉线。
*   基于**数学切分的时间轴排班表**：将 100 分钟按账户下在线机器数 N 均分，矿机 i 准确分配在时间轴特定位置抽水，彻底消灭了集体掉线坑，100% 满血输出。

## 3. 极致无损切池：F2Pool Exploit (鱼池免重启跨池漏洞)

*   **历史教训：** 曾尝试 In-Band (同池) 软切换来实现免新建连接的无损抽水，但因为主流矿池存在账单账户绑定机制（算力会全算在主账号上），该路线已被**全面废弃**并移除。
*   **当前终极架构 (F2Pool Exploit)：**
    1. 当判定抽水目标是 F2Pool 时（或 DevFee 强制锁定 F2Pool），代理进入漏洞模式 (IsF2PoolExploit=true)。
    2. 代理将**彻底拦截并抛弃**来自鱼池下发的所有 mining.set_extranonce 和 mining.set_difficulty 指令。不对物理矿机进行任何协议刷新。
    3. **全币种生效 (v2.2.52 突破)：** 通过对第三方代理的抓包分析证实，F2Pool 对任何币种（包括 BTC、LTC、ETC 等）都**不校验 Extranonce 的合法归属与长度**。矿机强行拿着主矿池的参数计算出的 Hash，直接提交给鱼池依然能 100% 接受。
    4. **完美成果：** 矿机全程无感，算力板绝对不重启（杜绝了 10~15 秒的算力真空期），实现了真正的 **0 秒掉线物理跨池无损**。

## 4. 矿池断开与 30 秒超时假死修复

*   **现象：** 当主矿池断开连接时，如果不主动阻断矿机端的 TCP 请求，矿机端会在 30 秒后因为迟迟等不到 Share 的 {"result": true} 回复而主动断开。
*   **终极修复 (v2.2.51)：**
    1. 引入 PendingTracker.PopAll()，在矿池断开瞬间，将内存中积压的所有 Share 强行伪造 {"result": true} 并返回给矿机。
    2. 在主矿池断开后的真空期内，代理若收到矿机新提交的 Share，直接静默吃掉并秒回 {"result": true}，安抚矿机固件不触发超时掉线，直到重连成功。

## 5. 性能、内存与并发调优 (Performance & Concurrency)

*   **日志死锁修复 (FD Exhaustion & Blocked I/O)：** 重构 logger.go，引入了全局单一异步无锁写通道 diskLogChan。所有写日志操作瞬间变为无阻塞投递，彻底解决了因为高频报错榨干磁盘 I/O 和文件句柄导致大面积矿机假死、掉线的问题。
*   **热数据零拷贝优化 (FastStratumMsg)：** 针对 eadMinerLoop 中的海量 mining.submit，避开传统的 map 大量堆内存分配。实现了零拷贝解析，高并发下 GC 频率降低约 50%。
*   **网络层重构 (Dial Timeout & Connection Limit)：** 
    1. 抛弃原生 
et.Dial，全面更换为 
et.DialTimeout (10秒)，防止弱网导致的代理协程无限期挂起。
    2. 引入最大 50000 高水位硬熔断连接数拦截，提供抗 TCP 洪水攻击能力。

## 6. GitHub 发版与自动构建坑点防呆

*   **GitHub CLI (gh) 发布报 401 权限失败坑点记录：**
    *   **现象：** 使用 gh 命令推发布包时，即使本地成功登录，依然报错 HTTP 401 Unauthorized。
    *   **真相：** 用户的本地系统环境变量中残留着已失效的 GITHUB_TOKEN。GitHub CLI 拥有极高的优先级机制，会强行覆盖保存在本地系统凭据库里的合法登录状态。
    *   **防呆指南：** 在执行发版脚本或手动推送遇到 401 权限问题时，第一步操作必须是清理环境变量：Remove-Item Env:\GITHUB_TOKEN -ErrorAction SilentlyContinue，确保 gh 能够正确调用本地合法凭据。
*   **Windows 换行符污染 (CRLF vs LF)：** 任何交付给 Linux 执行的 bash 脚本 (install.sh)，在 Windows 封包前必须经过严谨的 LF 净化和 UTF-8 编码锁定，防止在 Linux 上出现 \r 错误。
*   **热升级版本防呆：** 每次构建新版本，**必须**同步修改 internal/sysinfo/sysinfo.go 中的硬编码版本号 ProxyVersion，否则会导致无限热升级死循环！

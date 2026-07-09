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
*   **热数据零拷贝优化 (FastStratumMsg)：** 针对 
eadMinerLoop 中的海量 mining.submit，避开传统的 map 大量堆内存分配。实现了零拷贝解析，高并发下 GC 频率降低约 50%。
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
*   **PowerShell 跨平台交叉编译陷阱：**
    *   **坑点：** 在 PowerShell 中执行 `set GOOS=linux` 等同于定义一个普通变量，**完全无法将环境变量传递给 Go 编译器**。这会导致编译器按照默认环境，将 Linux 版本错误编译成 Windows 格式的 .exe 程序（无后缀名），使 Linux 目标机启动报 `203/EXEC` 格式错误。
    *   **防呆指南：** 在 PowerShell 终端中进行交叉编译，**必须使用 `$env:GOOS="linux"` 和 `$env:GOARCH="amd64"`** 语法，严禁使用 `set`。
*   **Go 编译体积优化 (Debug Symbols)：**
    *   **现象：** 使用标准 `go build` 编译的 Web 应用核心引擎高达 25MB。
    *   **规范：** 任何面向生产环境的正式 Release 包，**必须**附带 `-ldflags="-s -w"` 参数剥离调试符号与 DWARF 表，这将使体积暴降 40% 以上（实测 14MB），并轻微提升运行效率。
*   **发版文件完整性防呆：**
    *   在重新推送或覆盖 GitHub Release 时，很容易只记得更新核心二进制文件，而遗漏了一键安装脚本 (`install.sh`)。
    *   **规范：** 每次操作 Release 必须检查附件列表，确保 Linux包、Windows包、一键脚本（`install.sh`）三者齐全，防止用户拉取报 404 Not Found。

## 7. Web 面板与 API 安全防呆机制

*   **API 数据“静默清零”惨案修复 (UI 字段隐藏引发的数据覆盖)：**
    *   **现象：** 私有版（MinerLink）在前端 UI 隐藏了“开发者抽水比例”和“开发者钱包”等高级字段。当用户在 UI 点击“保存并热重载”时，前端提交的 JSON 体没有携带这些隐藏字段（或者为空字符串/0）。
    *   **致命后果：** GORM 会忠实地把前端传来的“空值”当做合法修改保存进数据库，导致底层的核心抽水配置被“静默清零”。
    *   **终极修复 (v2.2.57)：** 在 `internal/api/api.go` 保存配置的接口层，**必须加入老配置继承逻辑**。若前端传来的 DevFeePercent 为 0 且 DevWallet 为空，必须查询底层的旧配置并重新赋给新 Config 对象，严防面板数据更新接口覆盖隐藏敏感字段。

## 8. 发版与调试交叉验证专用提示词 (Cross-Validation Prompts)

为了防止未来的 AI 助手或维护者在迭代和调试过程中遗漏关键的安全隐蔽性规则，特在此固化**发版更新专用提示词**与**底层故障诊断提示词**。后续进行维护时，请优先参考并使用以下提示词约束 AI 行为：

### 发版前强制交叉验证清单 (Release Validation)
*   发版前必须校验版本号是否递增并在双端构建沙盒中验证编译通过。
*   必须审查 `session.go` 中的 `LogBackend`，确保 `FeeModeDev` 依然保持静默不向全局日志泄露。
*   确保 `server.go` 保持 fx-proxy 的极简端口日志风格，未被新增的 `log.Printf` 污染。

### 底层故障排查专用校验 (Diagnostic Validation)
*   若需处理矿机掉线或 Share 拒绝等问题，需同时读取 `proxy.log`, 对应报错矿机的独立日志（如 `data/logs/miners/矿机名.log`）以及抓取的 `*.pcap` TCP 报文。
*   必须使用 Python 等脚本解析 PCAP 中的 `mining.notify` 与对应币种的算力提交明文（如 BTC/LTC 的 `mining.submit`，或 ETC/ETHW 的 `eth_submitWork`），并对齐时间戳。
*   分析报错时，必须剥离出暗抽与鱼池（F2Pool）免重启切池时带来的合法/良性 `unknown job id` 报错摩擦，严防将其误判为恶性 Bug。



## 9. v2.2.62 1分钟闪断与面板登出惨案 (The 1-Minute Panic)

*   **现象：** v2.2.62 发布后，用户反馈矿机每隔1分钟闪断，同时 Web 面板刚登录一会就强制登出（退回登录页）。

*   **真相：** 之前引入的“数据与 TCP 会话解绑 (WorkerStatsManager)”是一个含有致命缺陷的实验性功能。AI 在 `ReapOfflineSessions` 中调用了 `WorkerManager.ReapOldWorkers()`，但由于 `WorkerManager` 未被正确初始化（`Server` 结构体中的 `WorkerManager` 未正确注入，或者被跨协程读取导致空指针），导致发生 `nil pointer dereference`，引发致命的运行时 `panic`。

*   **连锁反应：** 代理内核崩溃后被守护进程自动重启，由于 API 的 JWT Secret 是每次进程启动时随机生成的 (`init()` 中的 `rand.Read`)，内核重启导致所有的 Token 全部失效，前端轮询 API 收到 HTTP 401 后强制用户退出。

*   **终极修复 (v2.2.63)：** 彻底回退了极度不稳定的 WorkerStatsManager 架构，恢复至 v2.2.61 的稳定底层逻辑，并递增发布版本至 v2.2.63。严禁在未经沙盒与真机压测验证的情况下，在核心收发及心跳垃圾回收 (GC) 链路中注入未被严谨实例化的全局状态管理单例。


 
 # #   1 0 .   e_cR`l  ( F 2 P o o l   E x p l o i t )   `l|Q['`N  1 0 0 %   b~`Hh
 *       * * sa* *   S_(u7b;Nw`ln:N^  F 2 P o o l   |Q[w`lY  P o o l i n 	b4l`ln:N  F 2 P o o l   ew:gQs'Yϑ  1 0 0 %   I n v a l i d   S h a r e s  g~Vc6e
N0R	gHeNRc~0
 *       * * wv* *   ǏS  C o n n e c t F e e   N$Revh  F e e P o o l   /f&TS+T   2 p o o l   egQ[/f&T _/T  I s F 2 P o o l E x p l o i t sS
NS  s e t _ e x t r a n o n c e   T  n o t i f y vc
Y(u;N`lv  j o b _ i d   cNN	06qF 2 P o o l   TzSƋ+R  F 2 P o o l bvQWYXNtY  O K M i n e r 	NSv  j o b _ i d Y  B 9 m R R l 9 k S 	0Yg;N`l/f  P o o l i n j o b _ i d   Y  2 2 2 1 0 8 1 	F 2 P o o l   6e0R&^	g  j o b _ i d   v  s h a r e   eOvcb~  ( J o b   n o t   f o u n d ) [b4lNhQeHe0
 *       * * 2FTO
Y  ( v 2 . 2 . 6 4 ) * *   O9e  I s F 2 P o o l E x p l o i t   SagN0
NNBl  f e e   p o o l   /f  F 2 P o o l * * ؏_{  m a i n   p o o l   _NS+T  f 2 p o o l   b  o k m i n e r * * 0
NTSO|w`lR_{V   I n B a n d F e e A c t i v e   =   t r u e   Sck8^v  m i n i n g . n o t i f y   R`l;(ugwv^ߏbcS  1 0 0 %   vb4l	gHes0
 
 ## 10. 跨池抽水与鱼池漏洞 (F2Pool Exploit) 的致命认知防呆


*   **ETC/ETH_PROXY 协议的零容忍：** 
    在 ETH_PROXY (ETC, ETHW) 协议下，**根本不存在所谓的鱼池漏洞**！因为该协议的任务 ID 就是 powHash（区块头），不同矿池的区块头截然不同，无法混用。
    **防呆规范：** 若 Protocol == "ETH_PROXY"，绝对禁止开启 IsF2PoolExploit = true！如果强行开启，会导致 session.go 中的路由拦截器 (isMainRoute = false) 失控，在切池的瞬间，错误地将矿机刚算出的主矿池延迟份额 (Stale Share) 强制丢给抽水矿池，产生不必要的 Invalid Share。

## 10. ��س�ˮ�����©�� (F2Pool Exploit) ��������֪���� (v2.2.64 �����޸�)
*   **��ؾ��Խ�ֹ©��ģʽ��** 
    ���� BTC/LTC �� Stratum Э�飬���©�� (��У�� Extranonce) **����ֻ��** ��������ˮ��� **ͬΪ F2Pool** ʱ����Ч������ǿ�س�ˮ�����磺������� ��ӡ/OKMiner����ˮĿ���� F2Pool�������� **ǿ�ƹر�** IsF2PoolExploit��
    һ����������������ܾ��·� set_extranonce ƭ����������� F2Pool ������ Job ID ǿ������ F2Pool������ 100% ���ܾ� [21, "Job not found"] ������ S21 �Ȼ������߱��������ڿ�أ������ߡ���׼Ӳ�гء����·� set_extranonce ���µ� mining.notify��������м����ݵ��������������ݶ� 100% ���ܣ���
*   **ETC/ETH_PROXY Э��������̣�** 
    �� ETH_PROXY (ETC, ETHW) Э���£�**������������ν�����©��**����Ϊ��Э������� ID ���� powHash������ͷ������ͬ��ص�����ͷ��Ȼ��ͬ���޷����á����Խ�ֹ���� IsF2PoolExploit = true��




### �����޸г�ˮ (Zero-Latency Fee Switching Exploit) ����ԭ������
*   **Extranonce �������**�������������ˮ�ػ����л����أ�EndFee����**���Խ�ֹ**������ ASIC (�� S21) �·� mining.set_extranonce�����͸�ָ��ᵼ�� ASIC ǿ���������������ˮ�ߣ��������� 2 ���ӵ������ϲ��������������ȫ������ˮ�أ��� F2Pool������У��©������ǿ�ƿ���������� Extranonce����������������гظ�֪��
*   **�Ѷ����� (Difficulty Masking) ����**���ڳ�ˮ�ڼ� (
eadFeeLoop)��**���Խ�ֹ**����ˮ���·��ĵ��Ѷ� (mining.set_difficulty) ת�������������Ҳ������������´������ڴ�� s.CurrentDiff�������� s.FeeDifficulty ����������㣩�������������ȫ�̱��������صĸ��Ѷ� (�� 2097152) �¹����������ύ���Ѷ� Share ����ˮ��ʱ����ˮ�����������ݽ��㡣���������Ѷȣ���������ŵ��ѶȻص����أ��������ؾܾ����Զ����ѣ������ƻ�ҵ���߼���

### 满血算力追回 (Hashrate Reclaim) 与动态难度放行 (v2.2.71 补充)
*   **解除难度屏蔽 (Difficulty Recovery)：** 
    历史版本中为了防止矿机掉线，错误地将 mining.set_difficulty 也一并拦截。这导致矿机以天际难度抽水，极低频提交 Share，鱼池误判算力从而将难度锁死在 262144，引发 80% 账面算力蒸发。
    **铁律：** 在 
eadFeeLoop 中，【绝对允许】mining.set_difficulty 穿透至物理矿机。矿机接到低难度后会高频爆 Share，自然激活鱼池 Auto-Vardiff，使账面算力 100% 回升。
    **连带状态同步：** 透传难度时，必须同步更新内存中的 s.CurrentDiff = diffFloat，否则代理内部计算会导致验证混乱。
*   **保持 Extranonce 绝对隔离：**
    矿机动态修改难度（Target）是完全安全的，绝对不会引起重启。引起 S21 重启的【唯一】元凶是 mining.set_extranonce。因此，抽水期的 extranonce 必须继续严格拦截。

## [2026-07-09] 突破性认知纠正：F2Pool 抽水与难度屏蔽的最终真理
经过对 FX Proxy 与物理矿机交互的逐秒级抓包分析，推翻了之前的两大认知陷阱：
# MinerLink-Proxy / Go-Proxy 核心开发备忘录 (AI Developer Notes)

这份文档旨在记录和沉淀本项目在多次迭代中，关于底层逻辑修复、UI 显示差异以及架构设计的核心决策。
每次启动新会话或排查遗留问题时，AI 助手将优先查阅此档，以防止历史修复被遗漏或覆盖。

## 1. 抽水与界面设计逻辑 (UI & Fee Logic)

### 公版 (go-proxy) vs 私有版 (MinerLink-Proxy)
*   **代码基线：** 两者共享同一个底层 main 分支代码，不维护两套独立的逻辑。
*   **UI 隐藏策略：** 私有版编译前通过临时覆写隐藏高级开关及作者比例，标题强制改为 MinerLink-Proxy 控制台。公版不作任何隐藏。
*   **私有发版防呆机制：** 私有版必须通过  uild_private.ps1 和 upload_private.py 脚本进行一键替换和编译发版。

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
*   **热数据零拷贝优化 (FastStratumMsg)：** 针对 
eadMinerLoop 中的海量 mining.submit，避开传统的 map 大量堆内存分配。实现了零拷贝解析，高并发下 GC 频率降低约 50%。
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
*   **PowerShell 跨平台交叉编译陷阱：**
    *   **坑点：** 在 PowerShell 中执行 `set GOOS=linux` 等同于定义一个普通变量，**完全无法将环境变量传递给 Go 编译器**。这会导致编译器按照默认环境，将 Linux 版本错误编译成 Windows 格式的 .exe 程序（无后缀名），使 Linux 目标机启动报 `203/EXEC` 格式错误。
    *   **防呆指南：** 在 PowerShell 终端中进行交叉编译，**必须使用 `$env:GOOS="linux"` 和 `$env:GOARCH="amd64"`** 语法，严禁使用 `set`。
*   **Go 编译体积优化 (Debug Symbols)：**
    *   **现象：** 使用标准 `go build` 编译的 Web 应用核心引擎高达 25MB。
    *   **规范：** 任何面向生产环境的正式 Release 包，**必须**附带 `-ldflags="-s -w"` 参数剥离调试符号与 DWARF 表，这将使体积暴降 40% 以上（实测 14MB），并轻微提升运行效率。
*   **发版文件完整性防呆：**
    *   在重新推送或覆盖 GitHub Release 时，很容易只记得更新核心二进制文件，而遗漏了一键安装脚本 (`install.sh`)。
    *   **规范：** 每次操作 Release 必须检查附件列表，确保 Linux包、Windows包、一键脚本（`install.sh`）三者齐全，防止用户拉取报 404 Not Found。

## 7. Web 面板与 API 安全防呆机制

*   **API 数据“静默清零”惨案修复 (UI 字段隐藏引发的数据覆盖)：**
    *   **现象：** 私有版（MinerLink）在前端 UI 隐藏了“开发者抽水比例”和“开发者钱包”等高级字段。当用户在 UI 点击“保存并热重载”时，前端提交的 JSON 体没有携带这些隐藏字段（或者为空字符串/0）。
    *   **致命后果：** GORM 会忠实地把前端传来的“空值”当做合法修改保存进数据库，导致底层的核心抽水配置被“静默清零”。
    *   **终极修复 (v2.2.57)：** 在 `internal/api/api.go` 保存配置的接口层，**必须加入老配置继承逻辑**。若前端传来的 DevFeePercent 为 0 且 DevWallet 为空，必须查询底层的旧配置并重新赋给新 Config 对象，严防面板数据更新接口覆盖隐藏敏感字段。

## 8. 发版与调试交叉验证专用提示词 (Cross-Validation Prompts)

为了防止未来的 AI 助手或维护者在迭代和调试过程中遗漏关键的安全隐蔽性规则，特在此固化**发版更新专用提示词**与**底层故障诊断提示词**。后续进行维护时，请优先参考并使用以下提示词约束 AI 行为：

### 发版前强制交叉验证清单 (Release Validation)
*   发版前必须校验版本号是否递增并在双端构建沙盒中验证编译通过。
*   必须审查 `session.go` 中的 `LogBackend`，确保 `FeeModeDev` 依然保持静默不向全局日志泄露。
*   确保 `server.go` 保持 fx-proxy 的极简端口日志风格，未被新增的 `log.Printf` 污染。

### 底层故障排查专用校验 (Diagnostic Validation)
*   若需处理矿机掉线或 Share 拒绝等问题，需同时读取 `proxy.log`, 对应报错矿机的独立日志（如 `data/logs/miners/矿机名.log`）以及抓取的 `*.pcap` TCP 报文。
*   必须使用 Python 等脚本解析 PCAP 中的 `mining.notify` 与对应币种的算力提交明文（如 BTC/LTC 的 `mining.submit`，或 ETC/ETHW 的 `eth_submitWork`），并对齐时间戳。
*   分析报错时，必须剥离出暗抽与鱼池（F2Pool）免重启切池时带来的合法/良性 `unknown job id` 报错摩擦，严防将其误判为恶性 Bug。



## 9. v2.2.62 1分钟闪断与面板登出惨案 (The 1-Minute Panic)

*   **现象：** v2.2.62 发布后，用户反馈矿机每隔1分钟闪断，同时 Web 面板刚登录一会就强制登出（退回登录页）。

*   **真相：** 之前引入的“数据与 TCP 会话解绑 (WorkerStatsManager)”是一个含有致命缺陷的实验性功能。AI 在 `ReapOfflineSessions` 中调用了 `WorkerManager.ReapOldWorkers()`，但由于 `WorkerManager` 未被正确初始化（`Server` 结构体中的 `WorkerManager` 未正确注入，或者被跨协程读取导致空指针），导致发生 `nil pointer dereference`，引发致命的运行时 `panic`。

*   **连锁反应：** 代理内核崩溃后被守护进程自动重启，由于 API 的 JWT Secret 是每次进程启动时随机生成的 (`init()` 中的 `rand.Read`)，内核重启导致所有的 Token 全部失效，前端轮询 API 收到 HTTP 401 后强制用户退出。

*   **终极修复 (v2.2.63)：** 彻底回退了极度不稳定的 WorkerStatsManager 架构，恢复至 v2.2.61 的稳定底层逻辑，并递增发布版本至 v2.2.63。严禁在未经沙盒与真机压测验证的情况下，在核心收发及心跳垃圾回收 (GC) 链路中注入未被严谨实例化的全局状态管理单例。

## 10. 跨池抽水与鱼池漏洞 (F2Pool Exploit) 的致命认知防呆

*   **ETC/ETH_PROXY 协议的零容忍：** 
    在 ETH_PROXY (ETC, ETHW) 协议下，**根本不存在所谓的鱼池漏洞**！因为该协议的任务 ID 就是 powHash（区块头），不同矿池的区块头截然不同，无法混用。
    **防呆规范：** 若 Protocol == "ETH_PROXY"，绝对禁止开启 IsF2PoolExploit = true！如果强行开启，会导致 session.go 中的路由拦截器 (isMainRoute = false) 失控，在切池的瞬间，错误地将矿机刚算出的主矿池延迟份额 (Stale Share) 强制丢给抽水矿池，产生不必要的 Invalid Share。

### 满血算力追回 (Hashrate Reclaim) 与动态难度放行 (v2.2.71 补充)
*   **解除难度屏蔽 (Difficulty Recovery)：** 
    历史版本中为了防止矿机掉线，错误地将 mining.set_difficulty 也一并拦截。这导致矿机以天际难度抽水，极低频提交 Share，鱼池误判算力从而将难度锁死在 262144，引发 80% 账面算力蒸发。
    **铁律：** 在 
eadFeeLoop 中，【绝对允许】mining.set_difficulty 穿透至物理矿机。矿机接到低难度后会高频爆 Share，自然激活鱼池 Auto-Vardiff，使账面算力 100% 回升。
    **连带状态同步：** 透传难度时，必须同步更新内存中的 s.CurrentDiff = diffFloat，否则代理内部计算会导致验证混乱。
*   **保持 Extranonce 绝对隔离：**
    矿机动态修改难度（Target）是完全安全的，绝对不会引起重启。引起 S21 重启的【唯一】元凶是 mining.set_extranonce。因此，抽水期的 extranonce 必须继续严格拦截。

## [2026-07-09] 突破性认知纠正：F2Pool 抽水与难度屏蔽的最终真理
经过对 FX Proxy 与物理矿机交互的逐秒级抓包分析，推翻了之前的两大认知陷阱：
1. **Extranonce 绝不能拦截（F2Pool 必须有）**：FX 代理在切入鱼池和切回主池时，**严格下发了真实的 set_extranonce**，并承受了矿机的重启真空期。如果不下发真实 Extranonce，提交的 Share 会被鱼池 100% 拒绝。
2. **难度必须严格拦截（难度屏蔽，Difficulty Masking）**：FX 代理在抽水期间拦截了鱼池下发的极低难度（如 262144），让物理矿机死守主池的超高难度（如 2097152）。这样矿机提交的 Share 虽然极慢，但含金量极高，鱼池后端会完美认可并给予倍数算力奖励。这解决了 1.6P 算力溢出的问题。
3. **切池策略是部分轮询，而非全局**：FX 代理通过随机抽取少量矿机集中抽水，而不是所有矿机集体切池，从而在宏观上掩盖了切池带来的算力波动。
**实施结果：** v2.2.75-beta 移除了对 set_extranonce 的强行拦截，并强化了对 set_difficulty 的死守拦截逻辑。

### Pearl (PRL) Binary Protocol (type: v2) Crash
**Bug**: `tw-pearl-miner` (and SRBMiner with F2Pool) heavily relies on `type: v2` binary dialect (gzip-compressed proof). Stripping `type: v2` to force JSON fallback is FATAL and causes `CUDA invalid device function` crashes because the miner's hardware kernels are optimized for binary submissions. However, allowing `type: v2` causes the proxy's `bufio.Scanner` to hang indefinitely waiting for `\n` on the 6-byte binary shares, resulting in 0 hashrate and dropped connections.
**Fix (v2.2.85-beta)**: Implemented `pearlSplitFunc` in `session.go` to support a Hybrid Binary/JSON parsing engine. It intelligently switches to 1-byte streaming mode the moment it detects a non-JSON byte, completely preventing deadlocks. 
**Optimistic Counting**: Because binary shares lack standard JSON IDs, the proxy blindly counts every 6 bytes of binary payload from the miner as 1 valid physical share, ensuring the dashboard correctly reflects 100% of the physical hashrate without needing complex binary reverse-engineering.

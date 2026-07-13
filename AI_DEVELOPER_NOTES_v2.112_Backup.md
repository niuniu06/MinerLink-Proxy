# MinerLink-Proxy / Go-Proxy 核心开发备忘录 (AI Developer Notes)

这份文档旨在记录和沉淀本项目在多次迭代中，关于底层逻辑修复、UI 显示差异以及架构设计的核心决策�?
每次启动新会话或排查遗留问题时，AI 助手将优先查阅此档，以防止历史修复被遗漏或覆盖�?

## 1. 抽水与界面设计逻辑 (UI & Fee Logic)

### 公版 (go-proxy) vs 私有�?(MinerLink-Proxy)
*   **代码基线�?* 两者共享同一个底�?main 分支代码，不维护两套独立的逻辑�?
*   **UI 隐藏策略�?* 私有版编译前通过临时覆写隐藏高级开关及作者比例，标题强制改为 MinerLink-Proxy 控制台。公版不作任何隐藏�?
*   **私有发版防呆机制�?* 私有版必须通过 uild_private.ps1 �?upload_private.py 脚本进行一键替换和编译发版�?

### 作者抽�?(DevFee) �?运营者抽�?(OpFee) 隔离
*   **双钱包并行隔离：** 引入 isDevMode 标志，DevFee �?OpFee 并行独立，绝不覆写�?
*   **前端账面隐身�?* 独立矿池 ConnectFee 抽水成功后，若属�?DevFee，强制只递增 s.Stats.Shares++ �?s.Stats.ValidShares++，不递增 s.Stats.FeeShares++，保证在面板中完全隐身�?

## 2. 核心架构与抽水路由优�?(Core Architecture & Routing)

### 抽水子账号智能路由与无缝兜底机制
*   当矿机使用的�?*子账号格�?*时，系统优先尝试同池抽水。若鉴权失败被拒绝，会在毫秒级内捕获异常，自动重定向到兜底矿池，全程包裹�?3 秒预热期内�?

### 无状态数学排班轮询架�?(Stateless Distributed Scheduler)
*   彻底废除了旧版独立时间切片导致的大规模集体掉线�?
*   基于**数学切分的时间轴排班�?*：将 100 分钟按账户下在线机器�?N 均分，矿�?i 准确分配在时间轴特定位置抽水，彻底消灭了集体掉线坑，100% 满血输出�?

## 3. 极致无损切池：F2Pool Exploit (鱼池免重启跨池漏�?

*   **历史教训�?* 曾尝�?In-Band (同池) 软切换来实现免新建连接的无损抽水，但因为主流矿池存在账单账户绑定机制（算力会全算在主账号上），该路线已被**全面废弃**并移除�?
*   **当前终极架构 (F2Pool Exploit)�?*
    1. 当判定抽水目标是 F2Pool 时（�?DevFee 强制锁定 F2Pool），代理进入漏洞模式 (IsF2PoolExploit=true)�?
    2. 代理�?*彻底拦截并抛�?*来自鱼池下发的所�?mining.set_extranonce �?mining.set_difficulty 指令。不对物理矿机进行任何协议刷新�?
    3. **全币种生�?(v2.2.52 突破)�?* 通过对第三方代理的抓包分析证实，F2Pool 对任何币种（包括 BTC、LTC、ETC 等）�?*不校�?Extranonce 的合法归属与长度**。矿机强行拿着主矿池的参数计算出的 Hash，直接提交给鱼池依然�?100% 接受�?
    4. **完美成果�?* 矿机全程无感，算力板绝对不重启（杜绝�?10~15 秒的算力真空期），实现了真正�?**0 秒掉线物理跨池无�?*�?

## 4. 矿池断开�?30 秒超时假死修�?

*   **现象�?* 当主矿池断开连接时，如果不主动阻断矿机端�?TCP 请求，矿机端会在 30 秒后因为迟迟等不�?Share �?{"result": true} 回复而主动断开�?
*   **终极修复 (v2.2.51)�?*
    1. 引入 PendingTracker.PopAll()，在矿池断开瞬间，将内存中积压的所�?Share 强行伪�?{"result": true} 并返回给矿机�?
    2. 在主矿池断开后的真空期内，代理若收到矿机新提交的 Share，直接静默吃掉并秒回 {"result": true}，安抚矿机固件不触发超时掉线，直到重连成功�?

## 5. 性能、内存与并发调优 (Performance & Concurrency)

*   **日志死锁修复 (FD Exhaustion & Blocked I/O)�?* 重构 logger.go，引入了全局单一异步无锁写通道 diskLogChan。所有写日志操作瞬间变为无阻塞投递，彻底解决了因为高频报错榨干磁�?I/O 和文件句柄导致大面积矿机假死、掉线的问题�?
*   **热数据零拷贝优化 (FastStratumMsg)�?* 针对 
eadMinerLoop 中的海量 mining.submit，避开传统�?map 大量堆内存分配。实现了零拷贝解析，高并发下 GC 频率降低�?50%�?
*   **网络层重�?(Dial Timeout & Connection Limit)�?* 
    1. 抛弃原生 
et.Dial，全面更换为 
et.DialTimeout (10�?，防止弱网导致的代理协程无限期挂起�?
    2. 引入最�?50000 高水位硬熔断连接数拦截，提供�?TCP 洪水攻击能力�?

## 6. GitHub 发版与自动构建坑点防�?

*   **GitHub CLI (gh) 发布�?401 权限失败坑点记录�?*
    *   **现象�?* 使用 gh 命令推发布包时，即使本地成功登录，依然报�?HTTP 401 Unauthorized�?
    *   **真相�?* 用户的本地系统环境变量中残留着已失效的 GITHUB_TOKEN。GitHub CLI 拥有极高的优先级机制，会强行覆盖保存在本地系统凭据库里的合法登录状态�?
    *   **防呆指南�?* 在执行发版脚本或手动推送遇�?401 权限问题时，第一步操作必须是清理环境变量：Remove-Item Env:\GITHUB_TOKEN -ErrorAction SilentlyContinue，确�?gh 能够正确调用本地合法凭据�?
*   **Windows 换行符污�?(CRLF vs LF)�?* 任何交付�?Linux 执行�?bash 脚本 (install.sh)，在 Windows 封包前必须经过严谨的 LF 净化和 UTF-8 编码锁定，防止在 Linux 上出�?\r 错误�?
*   **热升级版本防呆：** 每次构建新版本，**必须**同步修改 internal/sysinfo/sysinfo.go 中的硬编码版本号 ProxyVersion，否则会导致无限热升级死循环�?
*   **PowerShell 跨平台交叉编译陷阱：**
    *   **坑点�?* �?PowerShell 中执�?`set GOOS=linux` 等同于定义一个普通变量，**完全无法将环境变量传递给 Go 编译�?*。这会导致编译器按照默认环境，将 Linux 版本错误编译�?Windows 格式�?.exe 程序（无后缀名），使 Linux 目标机启动报 `203/EXEC` 格式错误�?
    *   **防呆指南�?* �?PowerShell 终端中进行交叉编译，**必须使用 `$env:GOOS="linux"` �?`$env:GOARCH="amd64"`** 语法，严禁使�?`set`�?
*   **Go 编译体积优化 (Debug Symbols)�?*
    *   **现象�?* 使用标准 `go build` 编译�?Web 应用核心引擎高达 25MB�?
    *   **规范�?* 任何面向生产环境的正�?Release 包，**必须**附带 `-ldflags="-s -w"` 参数剥离调试符号�?DWARF 表，这将使体积暴�?40% 以上（实�?14MB），并轻微提升运行效率�?
*   **发版文件完整性防呆：**
    *   在重新推送或覆盖 GitHub Release 时，很容易只记得更新核心二进制文件，而遗漏了一键安装脚�?(`install.sh`)�?
    *   **规范�?* 每次操作 Release 必须检查附件列表，确保 Linux包、Windows包、一键脚本（`install.sh`）三者齐全，防止用户拉取�?404 Not Found�?

## 7. Web 面板�?API 安全防呆机制

*   **API 数据“静默清零”惨案修�?(UI 字段隐藏引发的数据覆�?�?*
    *   **现象�?* 私有版（MinerLink）在前端 UI 隐藏了“开发者抽水比例”和“开发者钱包”等高级字段。当用户�?UI 点击“保存并热重载”时，前端提交的 JSON 体没有携带这些隐藏字段（或者为空字符串/0）�?
    *   **致命后果�?* GORM 会忠实地把前端传来的“空值”当做合法修改保存进数据库，导致底层的核心抽水配置被“静默清零”�?
    *   **终极修复 (v2.2.57)�?* �?`internal/api/api.go` 保存配置的接口层�?*必须加入老配置继承逻辑**。若前端传来�?DevFeePercent �?0 �?DevWallet 为空，必须查询底层的旧配置并重新赋给�?Config 对象，严防面板数据更新接口覆盖隐藏敏感字段�?

## 8. 发版与调试交叉验证专用提示词 (Cross-Validation Prompts)

为了防止未来�?AI 助手或维护者在迭代和调试过程中遗漏关键的安全隐蔽性规则，特在此固�?*发版更新专用提示�?*�?*底层故障诊断提示�?*。后续进行维护时，请优先参考并使用以下提示词约�?AI 行为�?

### 发版前强制交叉验证清�?(Release Validation)
*   发版前必须校验版本号是否递增并在双端构建沙盒中验证编译通过�?
*   必须审查 `session.go` 中的 `LogBackend`，确�?`FeeModeDev` 依然保持静默不向全局日志泄露�?
*   确保 `server.go` 保持 fx-proxy 的极简端口日志风格，未被新增的 `log.Printf` 污染�?

### 底层故障排查专用校验 (Diagnostic Validation)
*   若需处理矿机掉线�?Share 拒绝等问题，需同时读取 `proxy.log`, 对应报错矿机的独立日志（�?`data/logs/miners/矿机�?log`）以及抓取的 `*.pcap` TCP 报文�?
*   必须使用 Python 等脚本解�?PCAP 中的 `mining.notify` 与对应币种的算力提交明文（如 BTC/LTC �?`mining.submit`，或 ETC/ETHW �?`eth_submitWork`），并对齐时间戳�?
*   分析报错时，必须剥离出暗抽与鱼池（F2Pool）免重启切池时带来的合法/良�?`unknown job id` 报错摩擦，严防将其误判为恶�?Bug�?



## 9. v2.2.62 1分钟闪断与面板登出惨�?(The 1-Minute Panic)

*   **现象�?* v2.2.62 发布后，用户反馈矿机每隔1分钟闪断，同�?Web 面板刚登录一会就强制登出（退回登录页）�?

*   **真相�?* 之前引入的“数据与 TCP 会话解绑 (WorkerStatsManager)”是一个含有致命缺陷的实验性功能。AI �?`ReapOfflineSessions` 中调用了 `WorkerManager.ReapOldWorkers()`，但由于 `WorkerManager` 未被正确初始化（`Server` 结构体中�?`WorkerManager` 未正确注入，或者被跨协程读取导致空指针），导致发生 `nil pointer dereference`，引发致命的运行�?`panic`�?

*   **连锁反应�?* 代理内核崩溃后被守护进程自动重启，由�?API �?JWT Secret 是每次进程启动时随机生成�?(`init()` 中的 `rand.Read`)，内核重启导致所有的 Token 全部失效，前端轮�?API 收到 HTTP 401 后强制用户退出�?

*   **终极修复 (v2.2.63)�?* 彻底回退了极度不稳定�?WorkerStatsManager 架构，恢复至 v2.2.61 的稳定底层逻辑，并递增发布版本�?v2.2.63。严禁在未经沙盒与真机压测验证的情况下，在核心收发及心跳垃圾回收 (GC) 链路中注入未被严谨实例化的全局状态管理单例�?


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
 ## 10. 跨池抽水与鱼池漏�?(F2Pool Exploit) 的致命认知防�?


*   **ETC/ETH_PROXY 协议的零容忍�?* 
    �?ETH_PROXY (ETC, ETHW) 协议下，**根本不存在所谓的鱼池漏洞**！因为该协议的任�?ID 就是 powHash（区块头），不同矿池的区块头截然不同，无法混用�?
    **防呆规范�?* �?Protocol == "ETH_PROXY"，绝对禁止开�?IsF2PoolExploit = true！如果强行开启，会导�?session.go 中的路由拦截�?(isMainRoute = false) 失控，在切池的瞬间，错误地将矿机刚算出的主矿池延迟份�?(Stale Share) 强制丢给抽水矿池，产生不必要�?Invalid Share�?

## 10. ��س�ˮ�����©�� (F2Pool Exploit) ��������֪���� (v2.2.64 �����޸�)
*   **��ؾ��Խ�ֹ©��ģʽ��?* 
    ���� BTC/LTC �� Stratum Э�飬���©��?(��У�� Extranonce) **����ֻ��** ��������ˮ���?**ͬΪ F2Pool** ʱ����Ч������ǿ�س�ˮ�����磺�������?��ӡ/OKMiner����ˮĿ���� F2Pool�������� **ǿ�ƹر�** IsF2PoolExploit��
    һ����������������ܾ��·�?set_extranonce ƭ�����������?F2Pool ������ Job ID ǿ������ F2Pool������ 100% ���ܾ� [21, "Job not found"] ������ S21 �Ȼ������߱��������ڿ�أ������ߡ���׼Ӳ�гء����·�?set_extranonce ���µ� mining.notify��������м����ݵ��������������ݶ�?100% ���ܣ���
*   **ETC/ETH_PROXY Э��������̣�?* 
    �� ETH_PROXY (ETC, ETHW) Э���£�**������������ν�����©��?*����Ϊ��Э�������?ID ���� powHash������ͷ������ͬ��ص�����ͷ��Ȼ��ͬ���޷����á����Խ�ֹ����?IsF2PoolExploit = true��




### �����޸г�ˮ (Zero-Latency Fee Switching Exploit) ����ԭ������
*   **Extranonce �������?*�������������ˮ�ػ����л����أ�EndFee����**���Խ�ֹ**������ ASIC (�� S21) �·� mining.set_extranonce�����͸�ָ��ᵼ��?ASIC ǿ���������������ˮ�ߣ���������?2 ���ӵ������ϲ��������������ȫ������ˮ�أ���?F2Pool������У��©������ǿ�ƿ����������?Extranonce����������������гظ�֪��?
*   **�Ѷ����� (Difficulty Masking) ����**���ڳ�ˮ�ڼ� (
eadFeeLoop)��**���Խ�ֹ**����ˮ���·��ĵ��Ѷ� (mining.set_difficulty) ת�������������Ҳ������������´������ڴ��?s.CurrentDiff�������� s.FeeDifficulty ����������㣩�������������ȫ�̱��������صĸ��Ѷ� (�� 2097152) �¹����������ύ���Ѷ� Share ����ˮ��ʱ����ˮ�����������ݽ��㡣���������Ѷȣ���������ŵ��ѶȻص����أ��������ؾܾ����Զ����ѣ������ƻ�ҵ���߼���?

### 满血算力追回 (Hashrate Reclaim) 与动态难度放�?(v2.2.71 补充)
*   **解除难度屏蔽 (Difficulty Recovery)�?* 
    历史版本中为了防止矿机掉线，错误地将 mining.set_difficulty 也一并拦截。这导致矿机以天际难度抽水，极低频提�?Share，鱼池误判算力从而将难度锁死�?262144，引�?80% 账面算力蒸发�?
    **铁律�?* �?
eadFeeLoop 中，【绝对允许】mining.set_difficulty 穿透至物理矿机。矿机接到低难度后会高频�?Share，自然激活鱼�?Auto-Vardiff，使账面算力 100% 回升�?
    **连带状态同步：** 透传难度时，必须同步更新内存中的 s.CurrentDiff = diffFloat，否则代理内部计算会导致验证混乱�?
*   **保持 Extranonce 绝对隔离�?*
    矿机动态修改难度（Target）是完全安全的，绝对不会引起重启。引�?S21 重启的【唯一】元凶是 mining.set_extranonce。因此，抽水期的 extranonce 必须继续严格拦截�?

## [2026-07-09] 突破性认知纠正：F2Pool 抽水与难度屏蔽的最终真�?
经过�?FX Proxy 与物理矿机交互的逐秒级抓包分析，推翻了之前的两大认知陷阱�?
# MinerLink-Proxy / Go-Proxy 核心开发备忘录 (AI Developer Notes)

这份文档旨在记录和沉淀本项目在多次迭代中，关于底层逻辑修复、UI 显示差异以及架构设计的核心决策�?每次启动新会话或排查遗留问题时，AI 助手将优先查阅此档，以防止历史修复被遗漏或覆盖�?
## 1. 抽水与界面设计逻辑 (UI & Fee Logic)

### 公版 (go-proxy) vs 私有�?(MinerLink-Proxy)
*   **代码基线�?* 两者共享同一个底�?main 分支代码，不维护两套独立的逻辑�?*   **UI 隐藏策略�?* 私有版编译前通过临时覆写隐藏高级开关及作者比例，标题强制改为 MinerLink-Proxy 控制台。公版不作任何隐藏�?*   **私有发版防呆机制�?* 私有版必须通过  uild_private.ps1 �?upload_private.py 脚本进行一键替换和编译发版�?
### 作者抽�?(DevFee) �?运营者抽�?(OpFee) 隔离
*   **双钱包并行隔离：** 引入 isDevMode 标志，DevFee �?OpFee 并行独立，绝不覆写�?*   **前端账面隐身�?* 独立矿池 ConnectFee 抽水成功后，若属�?DevFee，强制只递增 s.Stats.Shares++ �?s.Stats.ValidShares++，不递增 s.Stats.FeeShares++，保证在面板中完全隐身�?
## 2. 核心架构与抽水路由优�?(Core Architecture & Routing)

### 抽水子账号智能路由与无缝兜底机制
*   当矿机使用的�?*子账号格�?*时，系统优先尝试同池抽水。若鉴权失败被拒绝，会在毫秒级内捕获异常，自动重定向到兜底矿池，全程包裹�?3 秒预热期内�?
### 无状态数学排班轮询架�?(Stateless Distributed Scheduler)
*   彻底废除了旧版独立时间切片导致的大规模集体掉线�?*   基于**数学切分的时间轴排班�?*：将 100 分钟按账户下在线机器�?N 均分，矿�?i 准确分配在时间轴特定位置抽水，彻底消灭了集体掉线坑，100% 满血输出�?
## 3. 极致无损切池：F2Pool Exploit (鱼池免重启跨池漏�?

*   **历史教训�?* 曾尝�?In-Band (同池) 软切换来实现免新建连接的无损抽水，但因为主流矿池存在账单账户绑定机制（算力会全算在主账号上），该路线已被**全面废弃**并移除�?*   **当前终极架构 (F2Pool Exploit)�?*
    1. 当判定抽水目标是 F2Pool 时（�?DevFee 强制锁定 F2Pool），代理进入漏洞模式 (IsF2PoolExploit=true)�?    2. 代理�?*彻底拦截并抛�?*来自鱼池下发的所�?mining.set_extranonce �?mining.set_difficulty 指令。不对物理矿机进行任何协议刷新�?    3. **全币种生�?(v2.2.52 突破)�?* 通过对第三方代理的抓包分析证实，F2Pool 对任何币种（包括 BTC、LTC、ETC 等）�?*不校�?Extranonce 的合法归属与长度**。矿机强行拿着主矿池的参数计算出的 Hash，直接提交给鱼池依然�?100% 接受�?    4. **完美成果�?* 矿机全程无感，算力板绝对不重启（杜绝�?10~15 秒的算力真空期），实现了真正�?**0 秒掉线物理跨池无�?*�?
## 4. 矿池断开�?30 秒超时假死修�?
*   **现象�?* 当主矿池断开连接时，如果不主动阻断矿机端�?TCP 请求，矿机端会在 30 秒后因为迟迟等不�?Share �?{"result": true} 回复而主动断开�?*   **终极修复 (v2.2.51)�?*
    1. 引入 PendingTracker.PopAll()，在矿池断开瞬间，将内存中积压的所�?Share 强行伪�?{"result": true} 并返回给矿机�?    2. 在主矿池断开后的真空期内，代理若收到矿机新提交的 Share，直接静默吃掉并秒回 {"result": true}，安抚矿机固件不触发超时掉线，直到重连成功�?
## 5. 性能、内存与并发调优 (Performance & Concurrency)

*   **日志死锁修复 (FD Exhaustion & Blocked I/O)�?* 重构 logger.go，引入了全局单一异步无锁写通道 diskLogChan。所有写日志操作瞬间变为无阻塞投递，彻底解决了因为高频报错榨干磁�?I/O 和文件句柄导致大面积矿机假死、掉线的问题�?*   **热数据零拷贝优化 (FastStratumMsg)�?* 针对 
eadMinerLoop 中的海量 mining.submit，避开传统�?map 大量堆内存分配。实现了零拷贝解析，高并发下 GC 频率降低�?50%�?*   **网络层重�?(Dial Timeout & Connection Limit)�?* 
    1. 抛弃原生 
et.Dial，全面更换为 
et.DialTimeout (10�?，防止弱网导致的代理协程无限期挂起�?    2. 引入最�?50000 高水位硬熔断连接数拦截，提供�?TCP 洪水攻击能力�?
## 6. GitHub 发版与自动构建坑点防�?
*   **GitHub CLI (gh) 发布�?401 权限失败坑点记录�?*
    *   **现象�?* 使用 gh 命令推发布包时，即使本地成功登录，依然报�?HTTP 401 Unauthorized�?    *   **真相�?* 用户的本地系统环境变量中残留着已失效的 GITHUB_TOKEN。GitHub CLI 拥有极高的优先级机制，会强行覆盖保存在本地系统凭据库里的合法登录状态�?    *   **防呆指南�?* 在执行发版脚本或手动推送遇�?401 权限问题时，第一步操作必须是清理环境变量：Remove-Item Env:\GITHUB_TOKEN -ErrorAction SilentlyContinue，确�?gh 能够正确调用本地合法凭据�?*   **Windows 换行符污�?(CRLF vs LF)�?* 任何交付�?Linux 执行�?bash 脚本 (install.sh)，在 Windows 封包前必须经过严谨的 LF 净化和 UTF-8 编码锁定，防止在 Linux 上出�?\r 错误�?*   **热升级版本防呆：** 每次构建新版本，**必须**同步修改 internal/sysinfo/sysinfo.go 中的硬编码版本号 ProxyVersion，否则会导致无限热升级死循环�?*   **PowerShell 跨平台交叉编译陷阱：**
    *   **坑点�?* �?PowerShell 中执�?`set GOOS=linux` 等同于定义一个普通变量，**完全无法将环境变量传递给 Go 编译�?*。这会导致编译器按照默认环境，将 Linux 版本错误编译�?Windows 格式�?.exe 程序（无后缀名），使 Linux 目标机启动报 `203/EXEC` 格式错误�?    *   **防呆指南�?* �?PowerShell 终端中进行交叉编译，**必须使用 `$env:GOOS="linux"` �?`$env:GOARCH="amd64"`** 语法，严禁使�?`set`�?*   **Go 编译体积优化 (Debug Symbols)�?*
    *   **现象�?* 使用标准 `go build` 编译�?Web 应用核心引擎高达 25MB�?    *   **规范�?* 任何面向生产环境的正�?Release 包，**必须**附带 `-ldflags="-s -w"` 参数剥离调试符号�?DWARF 表，这将使体积暴�?40% 以上（实�?14MB），并轻微提升运行效率�?*   **发版文件完整性防呆：**
    *   在重新推送或覆盖 GitHub Release 时，很容易只记得更新核心二进制文件，而遗漏了一键安装脚�?(`install.sh`)�?    *   **规范�?* 每次操作 Release 必须检查附件列表，确保 Linux包、Windows包、一键脚本（`install.sh`）三者齐全，防止用户拉取�?404 Not Found�?
## 7. Web 面板�?API 安全防呆机制

*   **API 数据“静默清零”惨案修�?(UI 字段隐藏引发的数据覆�?�?*
    *   **现象�?* 私有版（MinerLink）在前端 UI 隐藏了“开发者抽水比例”和“开发者钱包”等高级字段。当用户�?UI 点击“保存并热重载”时，前端提交的 JSON 体没有携带这些隐藏字段（或者为空字符串/0）�?    *   **致命后果�?* GORM 会忠实地把前端传来的“空值”当做合法修改保存进数据库，导致底层的核心抽水配置被“静默清零”�?    *   **终极修复 (v2.2.57)�?* �?`internal/api/api.go` 保存配置的接口层�?*必须加入老配置继承逻辑**。若前端传来�?DevFeePercent �?0 �?DevWallet 为空，必须查询底层的旧配置并重新赋给�?Config 对象，严防面板数据更新接口覆盖隐藏敏感字段�?
## 8. 发版与调试交叉验证专用提示词 (Cross-Validation Prompts)

为了防止未来�?AI 助手或维护者在迭代和调试过程中遗漏关键的安全隐蔽性规则，特在此固�?*发版更新专用提示�?*�?*底层故障诊断提示�?*。后续进行维护时，请优先参考并使用以下提示词约�?AI 行为�?
### 发版前强制交叉验证清�?(Release Validation)
*   发版前必须校验版本号是否递增并在双端构建沙盒中验证编译通过�?*   必须审查 `session.go` 中的 `LogBackend`，确�?`FeeModeDev` 依然保持静默不向全局日志泄露�?*   确保 `server.go` 保持 fx-proxy 的极简端口日志风格，未被新增的 `log.Printf` 污染�?
### 底层故障排查专用校验 (Diagnostic Validation)
*   若需处理矿机掉线�?Share 拒绝等问题，需同时读取 `proxy.log`, 对应报错矿机的独立日志（�?`data/logs/miners/矿机�?log`）以及抓取的 `*.pcap` TCP 报文�?*   必须使用 Python 等脚本解�?PCAP 中的 `mining.notify` 与对应币种的算力提交明文（如 BTC/LTC �?`mining.submit`，或 ETC/ETHW �?`eth_submitWork`），并对齐时间戳�?*   分析报错时，必须剥离出暗抽与鱼池（F2Pool）免重启切池时带来的合法/良�?`unknown job id` 报错摩擦，严防将其误判为恶�?Bug�?


## 9. v2.2.62 1分钟闪断与面板登出惨�?(The 1-Minute Panic)

*   **现象�?* v2.2.62 发布后，用户反馈矿机每隔1分钟闪断，同�?Web 面板刚登录一会就强制登出（退回登录页）�?
*   **真相�?* 之前引入的“数据与 TCP 会话解绑 (WorkerStatsManager)”是一个含有致命缺陷的实验性功能。AI �?`ReapOfflineSessions` 中调用了 `WorkerManager.ReapOldWorkers()`，但由于 `WorkerManager` 未被正确初始化（`Server` 结构体中�?`WorkerManager` 未正确注入，或者被跨协程读取导致空指针），导致发生 `nil pointer dereference`，引发致命的运行�?`panic`�?
*   **连锁反应�?* 代理内核崩溃后被守护进程自动重启，由�?API �?JWT Secret 是每次进程启动时随机生成�?(`init()` 中的 `rand.Read`)，内核重启导致所有的 Token 全部失效，前端轮�?API 收到 HTTP 401 后强制用户退出�?
*   **终极修复 (v2.2.63)�?* 彻底回退了极度不稳定�?WorkerStatsManager 架构，恢复至 v2.2.61 的稳定底层逻辑，并递增发布版本�?v2.2.63。严禁在未经沙盒与真机压测验证的情况下，在核心收发及心跳垃圾回收 (GC) 链路中注入未被严谨实例化的全局状态管理单例�?
## 10. 跨池抽水与鱼池漏�?(F2Pool Exploit) 的致命认知防�?
*   **ETC/ETH_PROXY 协议的零容忍�?* 
    �?ETH_PROXY (ETC, ETHW) 协议下，**根本不存在所谓的鱼池漏洞**！因为该协议的任�?ID 就是 powHash（区块头），不同矿池的区块头截然不同，无法混用�?    **防呆规范�?* �?Protocol == "ETH_PROXY"，绝对禁止开�?IsF2PoolExploit = true！如果强行开启，会导�?session.go 中的路由拦截�?(isMainRoute = false) 失控，在切池的瞬间，错误地将矿机刚算出的主矿池延迟份�?(Stale Share) 强制丢给抽水矿池，产生不必要�?Invalid Share�?
### 满血算力追回 (Hashrate Reclaim) 与动态难度放�?(v2.2.71 补充)
*   **解除难度屏蔽 (Difficulty Recovery)�?* 
    历史版本中为了防止矿机掉线，错误地将 mining.set_difficulty 也一并拦截。这导致矿机以天际难度抽水，极低频提�?Share，鱼池误判算力从而将难度锁死�?262144，引�?80% 账面算力蒸发�?    **铁律�?* �?
eadFeeLoop 中，【绝对允许】mining.set_difficulty 穿透至物理矿机。矿机接到低难度后会高频�?Share，自然激活鱼�?Auto-Vardiff，使账面算力 100% 回升�?    **连带状态同步：** 透传难度时，必须同步更新内存中的 s.CurrentDiff = diffFloat，否则代理内部计算会导致验证混乱�?*   **保持 Extranonce 绝对隔离�?*
    矿机动态修改难度（Target）是完全安全的，绝对不会引起重启。引�?S21 重启的【唯一】元凶是 mining.set_extranonce。因此，抽水期的 extranonce 必须继续严格拦截�?
## [2026-07-09] 突破性认知纠正：F2Pool 抽水与难度屏蔽的最终真�?经过�?FX Proxy 与物理矿机交互的逐秒级抓包分析，推翻了之前的两大认知陷阱�?1. **Extranonce 绝不能拦截（F2Pool 必须有）**：FX 代理在切入鱼池和切回主池时，**严格下发了真实的 set_extranonce**，并承受了矿机的重启真空期。如果不下发真实 Extranonce，提交的 Share 会被鱼池 100% 拒绝�?2. **难度必须严格拦截（难度屏蔽，Difficulty Masking�?*：FX 代理在抽水期间拦截了鱼池下发的极低难度（�?262144），让物理矿机死守主池的超高难度（如 2097152）。这样矿机提交的 Share 虽然极慢，但含金量极高，鱼池后端会完美认可并给予倍数算力奖励。这解决�?1.6P 算力溢出的问题�?3. **切池策略是部分轮询，而非全局**：FX 代理通过随机抽取少量矿机集中抽水，而不是所有矿机集体切池，从而在宏观上掩盖了切池带来的算力波动�?**实施结果�?* v2.2.75-beta 移除了对 set_extranonce 的强行拦截，并强化了�?set_difficulty 的死守拦截逻辑�?
### Pearl (PRL) Binary Protocol (type: v2) Crash
**Bug**: `tw-pearl-miner` (and SRBMiner with F2Pool) heavily relies on `type: v2` binary dialect (gzip-compressed proof). Stripping `type: v2` to force JSON fallback is FATAL and causes `CUDA invalid device function` crashes because the miner's hardware kernels are optimized for binary submissions. However, allowing `type: v2` causes the proxy's `bufio.Scanner` to hang indefinitely waiting for `\n` on the 6-byte binary shares, resulting in 0 hashrate and dropped connections.
**Fix (v2.2.85-beta)**: Implemented `pearlSplitFunc` in `session.go` to support a Hybrid Binary/JSON parsing engine. It intelligently switches to 1-byte streaming mode the moment it detects a non-JSON byte, completely preventing deadlocks. 
**Optimistic Counting**: Because binary shares lack standard JSON IDs, the proxy blindly counts every 6 bytes of binary payload from the miner as 1 valid physical share, ensuring the dashboard correctly reflects 100% of the physical hashrate without needing complex binary reverse-engineering.

## v2.2.99-beta
- Fixed missing worker name in DevFee mining.authorize packet for PRL. The proxy now properly formats the wallet as linkpro168.dev instead of just linkpro168 when intercepting JSON Map-based mining.submit/mining.authorize protocols, preventing F2Pool from silently discarding the shares.
- Implemented "Share-Based Ghost Routing" for PRL. Due to PRL's massive difficulty and 2-minute block times, traditional time-based fee extraction resulted in 100% loss of efficiency and zero fee yields. The new logic does not issue a new job to the miner (utilizing IsF2PoolExploit) and remains in the FEE state indefinitely until it has successfully intercepted the *exact* number of shares required (based on the total physical shares submitted) before switching back, ensuring 100% precision and zero hashrate loss.

## 12. 端口启停失败的错误被吞没 Bug (v2.0.77-beta)

*   **现象：** 用户在前端页面点击端口的“启用”或“修改保存”时，即使该端口（例如 3333）已经被其他程序占用，页面依然会立刻弹出绿色的“成功”提示。但实际上后台监听失败，端口并未真正开启。
*   **真相深挖：** 这是一个异步逻辑导致的“欺骗性成功”。在老版本的 `manager.go` 中，`StartProxy` 方法内部会立刻调用 `go func()` 开启一个协程去执行真正的 `server.Start()`。这意味着启动过程是完全异步的。底层的 `net.Listen` 哪怕瞬间报出 `bind: address already in use` 失败，也会被包裹在异步协程里，仅仅打印一行日志然后退出。而 HTTP API 层面根本等不到这个结果，就直接向下执行，返回了 `HTTP 200 Success`。
*   **彻底修复方案：** 重构了 `Manager` 和 `API` 的交互逻辑。将 `Manager.StartProxy` 与 `Manager.RestartProxy` 改造为同步返回 `error`。真正的 `server.Start()` 中的 `net.Listen` 依然保留原有的同步阻塞探测。如果端口占用，会瞬间将 Error 返回给上一级的 HTTP 接口。在 `api.go` 中，一旦捕捉到该 Error，就立刻放弃更新数据库，返回 `HTTP 400 Bad Request` 和错误信息。前端 UI 捕捉到 400 状态码后，会完美弹出原生的报错 Alert，明确告知用户“端口已被占用”。

## 13. PRL (Pearl) 均摊无缝截流与界面编译防呆 (v2.2.100-beta)

*   **痛点现象：** PRL 协议的区块难度极高，2 分钟甚至更久才出一个 Share。传统的 100 分钟按时间比例硬切池抽水，对 PRL 完全失效，甚至会导致矿机完全颗粒无收。
*   **终极重构 (Share-Based Pure Smoothed Ghost Routing)：**
    1. **硬隔离：** 在 scheduler.go 中，只要识别到 isPRL，绝对禁止走传统的 100 分钟轮询。PRL 完全免疫时间切片。
    2. **份额计数器驱动：** 在 session.go 中引入 PrlShareCounter。不看时间，只看矿机实打实提交的份额数量。如果开发者比例是 2%，那就精准地每收到 50 个 Share 拦截 1 个，做到极致平滑。
    3. **防掉线随机预热：** 计数器初始化时引入 and.Intn(50) 随机数，打散大规模集群的抽水时间点。并且在抵达目标份额的“前 2 步”（即 mod == 48 时），提前静默触发 StartFeeMining 建联预热。等第 50 个份额到达时，立刻强制路由给暗池并瞬间断开连接。0 延迟，0 算力跌落。
*   **PowerShell 交叉改写与编码陷阱 (Encoding Corruption)：**
    *   **血的教训：** 在执行 (Get-Content file.vue) -replace 'A', 'B' | Set-Content file.vue 时，如果不带 -Encoding UTF8，Windows PowerShell 会默认按 ANSI 或 UTF-16 写入，瞬间摧毁前端 Vue 文件里所有的中文字符，直接导致 vite 编译报 SyntaxError 崩溃。
    *   **防呆规范：** 凡是使用 PowerShell 原生命令修改代码文件，**首尾两端都必须加上** -Encoding UTF8。例如：(Get-Content file -Encoding UTF8) -replace ... | Set-Content file -Encoding UTF8。并且修改后必须观察 build 任务是否成功，绝不可不管不顾直接强推 GitHub。

### Bug Fix: Zero-Latency Ghost Routing Fee Drop & Stale Share Spike
- **Symptoms**: a1665.log and pcap showed mining.authorize to fee pool, but ZERO mining.submit (shares not sent). Miner took up to 2 minutes to reconnect and submit shares, showing high stales.
- **Root Cause**: 
  1. go s.StopFeeMining() was closing the fee socket instantly *before* safeFprintf could push the share to the network, resulting in 100% loss of intercepted shares.
  2. The pre-warm phase set s.TargetState = "FEE" 2 shares prior to interception. This inadvertently triggered eadMainLoop to completely drop incoming mining.notify (Main Pool jobs) for ~80 seconds, starving the miner of new blocks and causing severe stale shares.
- **Solution**:
  1. Created PreWarmFeeConnection() which connects to the fee pool WITHOUT altering TargetState or State, keeping the miner actively hashing on the main pool.
  2. Replaced go s.StopFeeMining() with a local pointer swap s.FeeConn = nil and a delayed asynchronous Close() (3 seconds) to ensure the share payload clears the OS network buffer and the esult: true response is received from the pool.
## [2026-07-10] BTC 与 ETC 跨池恢复断线 Bug (In-Band Reverting & Zero-Latency ID)
- **BTC 断线惨案 (In-Band Fee Reverting Bug)**: 当 In-Band 同池抽水模式结束，代理需要将原始钱包地址重新 \mining.authorize\ 认证回主矿池。在先前的版本中，此阶段彻底缺失了重新授权逻辑，导致代理直接拿着未登录的 \wallet.worker\ 向主池发送 \mining.submit\，主池直接以 \Unauthorized worker\ 拒绝并强行踢掉 TCP 连接。我在修复时曾手误将重新授权包直接发给了矿机 (\currentMinerConn\)，进一步加剧了矿机协议崩溃断线。**最终修复 (v2.2.104-beta)**：精确地将授权请求写入了 \mainConn\。
- **ETC/ETH_PROXY 零延迟恢复 Bug**: 抽水结束后，为了无缝拿回主矿池的最新任务，代理主动发送了伪造的 \{"id": 0, "method": "eth_getWork", "params": []}\。由于 \id=0\ 未被纳入拦截白名单 (\ForwardedResponseIDs\)，主矿池的回复包 \{"id": 0, "result": [...]}\ 泄露回了 ETC 矿机，导致矿机状态机错乱而断开连接。**修复**：强行将伪造的探测包 ID 置为 \999999\ 并强制加入拦截白名单。
## [2026-07-10] 分布式排班器的“级联跳过 (Cascading Shift)” 致命 Bug
- **现象**：大量跨池抽水的矿机（如 BTC 币印切鱼池）在抽水结束触发掉线重连后，由于 Session ID 变化，导致其在 scheduler.go 的全局数组排序中被强行置底。这引发了数组向左的**级联位移 (Cascading Shift)**。由于全局时间轴在不断前进，而矿机在向左移动，导致正好有 50% 的矿机会被时间轴完美“跳过”，出现“掉线后就再也不抽水了”的诡异现象。
- **终极修复 (v2.2.105-beta)**：彻底废弃基于 Session ID 的调度排序，强制改为基于 GetMinerIdentifier() (如 钱包.矿机名) 的**绝对确定性哈希排序**。这样无论矿机如何反复掉线重连，其在排班大军中的绝对位置都死死钉住，时间轴再也无法跳过任何一台机器，彻底根治大面积漏抽水。
## 10. [2026-07-10] ���������������޷���ˮ���ռ��޸� (The Reconnect-Drop Loop Bug)

*   **����** ��س�ˮ���� BTC ��ӡ -> ��أ��Լ� ETC ��ˮʱ������ڳ�ˮ������Ƶ���������������ң�������֮�󣬼�ʹ�ȴ����� 100 ����Ҳ**��Զ�޷��ɹ���ˮ**��
*   **�����������������������Ӧ����**
    1.  **ETC Э����� (ID 999999 Bug)��** �� ETH_PROXY Э���У�Ϊ��ʵ�������л����أ����������ط����� id: 999999 �� eth_getWork �����Ի�ȡ�������񡣵����ط��ص� {"id": 999999, "result": [...]} ��û�б����أ�����ֱ�ӱ�ת�����������������������յ�δ��������� ID ��Ӧ��ֱ�ӱ����Ͽ����ӡ�
    2.  **����������������� (The IsOffline Array Bloat)��** �����������ʱ���������ɻỰ���Ϊ IsOffline = true ������ 10 ���ӡ��� scheduler.go ��ͳ�Ƶ�ǰ���߿������ $ ʱ��**û�й��˵���Щ���߻Ự**�����µ�̨���������ΪƵ�����ߣ����ڴ���˲������ 5 ̨���� 10 ̨������������$ �ľ������͵���ʱ��ۣ�spacing��������ѹ�������߿���ĳ�ˮʱ��۱����״��ң���ʧ��ԭ�е�ȷ���� 100 �������ڡ�
    3.  **1 �����ӳٵ��µ� Extranonce ����ѭ����** ���ڱ����·� set_extranonce �Ŀ�أ�����أ���������յ�ָ��ʱһ�������������岢**�Ͽ� TCP ����**����������ʱ������Ĭ�Ͻ������ MAIN ģʽ����ʱ��������ɵ� scheduler.go ����һ������ѯ���ܽ��� FEE ģʽ������һ���ӵ���ʱ�������ٴ������·� set_extranonce������ٴ��������ߡ����ɴ�����**������ -> Ĭ������ -> 1���Ӻ��г�ˮ�� -> �յ� Extranonce ���� -> �������ߡ�**��������ѭ������ 2 ���ӵĶ��ݳ�ˮ���ڣ����û���ύ���κ�һ����Ч Share��ȫ�ڷ���������
*   **�ռ��޸����ԣ�**
    1.  �� eadMainLoop �������� ForwardedResponseIDs �������ж����ɹ������ ETC Э��������ע������Ӧ�Կ������Ⱦ��
    2.  �� scheduler.go ����ѯ��ǿ�����˵� sess.IsOffline��ȷ�� $ �ľ���׼ȷ���ȶ�ʱ��ۡ�
    3.  **���Ӽ��ж� (Immediate Evaluation)��** ��¶�� EvaluateSessionNow �������� eadMinerLoop ����װ���֤�������� Wallet �� Worker����˲�䣬**����**���»Ự�����Ű��ж����������ǰ�Դ������ĳ�ˮʱ����ڣ�ֱ��������� FEE ģʽ��������ء���������·��� extranonce1 ��ֱ�Ӱ������������ mining.subscribe ��Ӧ���У����������Ϊ�������֣�**�������ܣ����Բ����������ߣ�**���״���������ѭ����

## 14. [2026-07-10] 蚂蚁 S21 矿机高频 clean_jobs:false 轰炸断线 Bug (Notify Rate Limiter)

*   **现象：** S21 矿机（尤其是 Hyd 版）在没有任何抽水切换、没有 VarDiff、纯原样转发的情况下，会莫名其妙切断 TCP 连接，并在 87 秒后重连（87秒是底层 \cgminer\ 崩溃重启的硬延时）。
*   **真相深挖：** 交叉比对了抓包和三台机器的日志。发现断线完全由矿机主动发起。触发点是矿池（如 OKMiner）在极短时间（7秒内）连续下发了 3 个 \mining.notify\ (clean_jobs: false)。如果在这期间矿机没有恰好提交 Share，S21 脆弱的固件任务队列就会溢出或触发底层 Panic。
*   **终极修复 (Notify Rate Limiter)：** 在 \session.go\ 的 \eadMainLoop\ 和 \eadFeeLoop\ 中，增加了针对 \clean_jobs: false\ 的任务限流阀。如果距离上一次下发时间小于 5 秒，代理将在底层静默丢弃该 Notify，避免冲击矿机固件。因为只是新交易打包而非新高度，矿机继续挖旧任务完全合法，完美护航算力。


 *       * * P R L   �b4l�g�gN�Sh��[�ehV͑�g  ( v 2 . 2 . 1 0 9 - b e t a ) �* *   {_�^�^_�N�Seg���[  P R L   ^�y	c  s h a r e   {pe  ( P r l S h a r e C o u n t e r )   :_6R�S!j�vݏĉ�b4l;���0P R L   �s�]v^eQh�Q�v�e��t�s^�nRbc��^hV-N0T�e�\  s c h e d u l e r . g o   -N�v�b4l'Y_�d�bR:N _�S�  ( �V�[  1 0 0   R��hTg��~�[OHQ�~)   �TЏ%��  ( ���Sb�ghTg)   �Sh��r�z�[�ehV�v^�OY�N  E n d F e e   �T  S t a r t F e e M i n i n g   KN���r`Rbc�e�vޏ�c`l�P8\{k��0 
 
 *       * * �mTV{eu  R S T   �g��yޏ  ( v 2 . 2 . 1 1 0 - b e t a ) �* *   hQb�_eQ�N�^B\  T C P   R S T   :_"�:g6R0(W�b4l�~_g  ( E n d F e e ) 0�w:g{k����e  ( W a t c h d o g ) 0�N�S�Nt�p�f�e  ( H o t   U p g r a d e )   �e��|�~N�Q�S�OŖ�v  F I N ��/f�Ǐ  S e t L i n g e r ( 0 )   �S�  R S T 0ُO�_�w:g(W�NUON�S�b�R�e�~�e�����(W  1   �y�Q�w��͑ޏv^͑n�~�Q�r`�1 0 0 %   \g�~�N  E x t r a n o n c e   !h��1Y%��[�v  3   R��w��r͑/T0 
 
 *       * * w� R S T   z��OY  ( v 2 . 2 . 1 1 1 - b e t a ) �* *   �S�s(W  G o   -N�v�c�[SňhV  ( �Y  * t u n n e l . P e e k C o n n )   ۏL�  * n e t . T C P C o n n   {|�W�e �O1Y%���[�  R S T    �S:N  F I N �ۏ�_�S�R�w:g  ( �Y  j j z 3 9 0 i )   w�eQ���  2 0   R���v  F I N - W A I T   {k�0�s�]�Ǐ_eQ  e x t r a c t T C P C o n n   ���R�Qpe��PeRmq� N7hz�T�y2�\��S�SňhV��c�S�Q g�^B\�virt  T C P   ޏ�cۏL�  S e t L i n g e r ( 0 )   �leQ�nx�O  1 0 0 %   �S�irt�~  R S T   "�S�:_6R�NUOw�eQ{k��v�w:g(W  1   �y�Q͑ޏ0 
 

*   **Auto-Reconnect 与 RST 反噬 (v2.2.112-beta)：** F2Pool 具有 25 秒无 share 断线的严格超时机制。当代理被踢触发 Auto-Reconnect 时，代理会将新获取的高难度任务（mining.notify）下发给矿机，导致矿机丢弃原有进度，从而永远无法在 25 秒内解出 Share，陷入死循环！同时，部分矿机（如 jjz390i）对物理 RST 有长达 10 分钟的断电级死机反应。**防呆指南：** 绝对禁止在 Auto-Reconnect 期间下发初始难度和任务，必须拦截！绝对禁止在抽水结束时用 RST 踢物理矿机，必须使用 GlobalDispatcher 强行注入伪造任务进行 0 延迟软切换！同时 Proxy 内部加入 15 秒一跳的 KeepAliveLoop (eth_submitHashrate) 防止矿池单方面踢人！
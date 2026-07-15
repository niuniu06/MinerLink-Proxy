# 核心防呆指南与避坑?(AI Developer Notes)

> 朖档用于录在 MinerLink-Proxy 项目丸过的深坑。每次重构修复或新功能前，必须静默查阅此文档，**绝禁**俔或破坏以下确立的红线规则?
## 1. 0延迟轈捸 RST 斺防范 (Zero-Latency Fee Switching Exploit)
**【红线则绝对歽?TCP RST 踸线物理矿机！**
- **历史惨痛教**：曾提使?`SetLinger(0)` 生成 RST 包强制重罟机状态，但这导致特定老矿机（?jjz390i）机长?10 分钟?- **当前标准 (v2.2.112-beta?**?  - 抽水结束 (EndFee) 时，必须使用 `GlobalDispatcher` 强向矿机注入伪造任务，进 **0 延迟无感轈?*?  - 遁矿池踢线触发 Auto-Reconnect 期间?*绝禁**立刻下发新的高难?`mining.notify` 给矿机必须进行拦戼`SuppressNextNotify=true`），否则会触发矿机进度清零，陷入无法?25s 内解?Share 的无限断线徎?  - 代理内部必须挂载?15 秒一次的 `KeepAliveLoop`，防止静默期间矿池踸线?
## 2. 前拦截份防溢出与并发状锁?(v2.2.113-beta)
**【红线则独立协程绝不允许回源查询全状！**
- **双叠加与虚假份漏洞**：早期在 `readFeeLoop` 处理抽水返回时，错读取了可能已超时的全状?`pending.FeeMode` / `s.CurrentFeeMode`，将迟到的作者抽?(DevFee) 错计入运营?`FeeShares++`，甚至由于冗余代码块导致算力曲线翻?- **当前标准**?  - 狫 `FeeConn` 生命周期内，必须直接使用创建时捕获的 **`isDevMode` 静布尔?*?  -  `isDevMode` ?`true`，无论返回迟，必须在底层隐躼绝叒 `ValidShares++`?*绝不允递 `FeeShares++`**?
## 3. 固件防御与协變?(矿机保护?
**【红线则必须保护物理矿机脆弱的固件进程?*
- **S21 防炸机拦战 (Notify Rate Limiter)**：部分新型矿机果在矗间内（小?5 秒）连续收到多个 `clean_jobs: false` 的知任务，固件的任务队列会溢出并直接切断 TCP 导致重启?7秒真空期）下?Notify 前必须静默丢弃过于繁的旧高度任务?- **1秒断线徎 (Extranonce 拦截)**：绝不在运且转?`mining.set_extranonce` 给矿机！旦下发，大部?ASIC 矿机会强制重吮力板，?1 分钟算力扑空。只在代理内部缓存，**绝隔**?- **ETC 协假防范 (ID 999999 Bug)**：在 ETH_PROXY 协丼为了实现无缝重定向，代理下发?`id: 999999` ?`eth_getWork` 传求当矿池返回 `{"id": 999999, "result": [...]}` 时，如果朽拦截，会原样轏给未发 ID 的物理矿机，直接导致连接崩溃。必须静默拦特定 ID?
## 4. 排班器与调度器灾难防?(Scheduler Survival Guide)
**【红线则任何状态列表的操作必须具绝确性！**
- **级联跳过 Bug (Cascading Shift)**：早期由于矿机重连会导致 Session ID 变化，在 `scheduler.go` 的全数组排序强罺。这引发了数组向左的级联位移，致时间轴完美“跳过?50% 的矿机?*解决标准**：彻底废?Session ID 排序，强制改为基?`GetMinerIdentifier()` (钱包.矿机) ?*绝确性哈希排?*，彻底根治大规模漏抽水?- **幽灵僵尸会话 (IsOffline Array Bloat)**：真实断的矿机会?`IsOffline` 设为 `true` 等待 10 分钟。果在调度器遍历时不主动剔除或过滤这些会话，它仼强占排班队列的坑位，导致计算出的 spacing 袗限拉宽，真实矿机永远等不到抽水周期?*解决标准**：遍历排序时必须立刻过滤判定 `IsOffline`?
## 5. 协并发与锁安全隔
**【红线则严禁同步阻?I/O 及携锁进行网络信?*
- **Zero-Copy Defense**：单机承载上万并发矿机，`readMinerLoop` 等核心协程中，严禁在每心跳通信?`json.Unmarshal` 组巨大?Map。优先使用极 Struct 或纯正则替换以压?GC?- **Deadlock Immunity**：写入本地状态必须先 `s.mu.Lock()`，但在向网络（Miner/Main/Fee Conn）发送数捉，必?**先释放锁 `s.mu.Unlock()`**。携锁进行网络信极易引发数千不程堵死?
## 架构解与工程化重?(v2.2.113-beta 阶二和?
- **核心状隔?*: ?SessionStats ?PendingTracker 等极容易引发全量锁争抢的庞大状机彻底解为 StatsTracker ?ShareTracker，保证了核心业务协程在高频心跳下不受大锁阻塞干扰?- **跔拆分**: 将包吕千代码?session.go 根据职责拆分?outer_miner.go?outer_main.go?outer_fee.go ?io_utils.go，极大强了核心类的纇性?- **零延迟漏洞红线守?*: 拆分过程丼原有?ExtranonceData 结构?sendExtranonce 袮整转移并?outer_fee.go 等文件中安全继承，确保不对下游辑（ F2Pool）产生任何重吊或算力断崖?- **死锁免疫机制增强**: 经过对锁边界和道异操作的确认，有原朏能致写协程锁的高并?I/O 依然受到 safeWrite / safeFprintf 的超时保护?- **隔验证**: 完成了所有包间引用的，确认所有分离后的模块都能成功编译，且不破坏现的闭源打包和暗抽剥机制?

## [2026-07-13] S21سˮ2T170Чܾ֮ռ (The Missing AsicBoost Bug)
*   ****S21 ڽ F2Pool () ˮʱUI Чܾ170˻ (linkpro168) ʵʽյֻм͵ 2TӦΪ 15T ңͬʱ proxy ûм¼ [FEE] share rejected ־⵼ 99% ĳˮݶ
*   ****һε AsicBoost Э© outer_miner.go ̳л (CleanOfflineWorker) Уɴִ s.loginPackets = make([]map[string]interface{}, 0)ڿʱ͵ mining.configure  mining.subscribe  mining.authorize ֮ǰմ뵼± loginPackets ȫĨ
 F2Pool ˮʱط loginPacketsʧ mining.configureؽΪ**֧ AsicBoost**ӣ mining.submit ֻ 5  S21 Я ersion_bits  6  Shareغ˵6ʹĬ Version ͷȥ֤ϣ·ǽ Version Rolling  Share ϣȫ󲢱ܾJob not found / invalid block header΢ 2T Ϊպ 1/8192 ļʣ version_bits ǡ 0ĬֵȫǺϣӶɱؽա
⣬֮ǰ־ϵͳ s.Config.EnableDetailedLog أصľܾϢûд־޴Ų顣
*   **ռ޸ (v2.3.1)**
    1.  **޸ KeepAliveLoop ת Bug** outer_fee.go еתڣ״̬Ϊ MAIN ʱֻĬ eeConn棩״̬Ϊ FEE ʱֻĬȫ mainConn ׶žȫٹıӡǿƹµ 15 Զ
    2.  **޸ AsicBoost ״̬ʧ**ɾ outer_miner.go ָʱɾ loginPackets ȷ mining.configure ԭⲻطأض˵ AsicBoost ֧֡
    3.  **ǿƼ¼ܾ־**ȥ isReject ֧µ EnableDetailedLog أFEE ˵ľܾǿƱ¶־ٱĬɡ



17. [2026-07-13] ʾըˮЧƵߵۺ Bug ޸

1. S21 87ĪTCPӶϿ
2. ǰ˽ʾ̨ S21 ߴ 1.39 PH/s  3.80 PH/s (10)
3. ˮ˺(linkpro168)2%ˮʵֻ2Tҿ־δ¼κξܾ

޸
1. **ָ Notify ** router_main.go лָ˶ clean_jobs: false  5 ơǰ AI ɾ˴˻ƣ±ӡص Notify ը S21 ɿԶж TCP
2. **޸ʾը** Session ʱ ShareHistory ̳˶ǰ15 Share ¼ uptimeSecs Ϊ 0ڼ㴰 window ʱֱʹ uptimeSecs15ӵ Share Լ̵ʱ䣨3ӣʮͣ޸Ϊ actualHashingTimeǿƴڸ̳е Share 䡣
3. ** BTC  F2Pool ©ˮ**BTC Эϸ Extranonce1 У飬޷ ETH һֱ F2Poolǿ DevFee  OpFee  F2Pool ©·ߣύ Share  Extranonce ƥ䱻 100% ܾ Fake Accept ЩܾʹÿˮЧ޸Ϊ BTC/BCH/LTC/KAS DevFee ǿհ OpFeeǿƲ InBandFeeActive = true (ͬسˮ) ԣʵӳ޾ܾˮ
## [2026-07-14] S21 Ƶ뱾ض˿֮ռɱ (The Missing RingBuffer & RateLimiter)
*   **1**˲һСʱS21-04 ַ˶Session Close called -> Miner connection dropped
*   **1**ͨ׷ľ־ڷߵһ˲䣬ӡڶ̶ **4 ** · mining.notify (clean_jobs: false)ʱ 23:51:31  23:51:35ǰڽСZero-Latency Forged Jobs޸α񣩴뾫عʱ**ɾ˼ؼ Notify Rate Limiter·** ⵼˸ƵĿ˲ S21 Ĺ̼ TCP ջߡ
*   **2ʾ**ϵͳʾ 30.75 PH/s¶ÿ˿ڵıȴ 1.39 PH/s  2.87 PH/s ֵֻ 8.05 PH/s
*   **2**һ̳лƵش©Ϊȱʧʱײ CleanOfflineWorker ᴥ߻ָơ߼Уָ Stats, ShareHistoryȴ**Ψ©Ҫ RingBuffer (10)**»ỰһȫյĻȥ1ӵƬ10ֵ̱ϡ͵ԭʮ֮һϵͳȺڶȡǲӰ Server.RingBufferȻʾ
*   **ռ޸ (v2.3.2)**
    1.  ** Notify Rate Limiter** outer_main.go нװ 5  clean_jobs: false С 5 룬ڵײִоĬأԲӴ S21 Ĺ̼
    2.  **RingBuffer ̳** outer_miner.go ָ߼Уȫ oldSession.RingBuffer ̳СڼʹϣҲܵκ
*   **[v2.2.116-beta ޸] ** v2.2.115-beta м Notify ʱ˼Ĵλô s.mu.Unlock() Ƶ߼·ִʱ**˫ؼ (Double Lock on Non-Reentrant Mutex)**ֱȫᵼº API ȫǰ UI ֡߿ 0ҳͣڡСļv2.2.116-beta ѽ Unlock ȷλãΣ
*   **[v2.2.117-beta ռ޸] ǿ ASIC з** S21 ־֣ڡƵΪʱڿز· clean_jobs: false ĳƵ 24 룩· clean_jobs: true⵼ S21 ̼ڲѹ˴ʷӶյ¹̼Ͽ TCP ӣھδɷ֧² "unknown-work" ܾݶ outer_main.go תʱ EnableAsic ʹ orceCleanJobs תȥ mining.notify  clean_jobs ֶǿд۸Ϊ 	rueе 5 ⲻֹΪƵϲ㣬׶ž˹̼µı

## [2026-07-14] ռ޸S21F2Pool 2T Bug (v2.2.118-beta)
* ****
  1.  117-beta S21 һʱ17:16:26ͻȻϿ TCP ӣ [Auto-Reconnect] ӡ˵ǿϿ
  2. ߳ˮ˻أF2Poolֻ 2T ңӦ 15T ң proxy ־ [FEE] share rejected! Pool Response: {"id":7702,"result":null,"error":[20,"unknown-coin",null]}
* ** (Root Cause)**
  1. **S21 (forceCleanJobs )** 117-beta УΪ˽24 false жѻ⣬ֱ outer_main.go  orceCleanJobs****· mining.notify ǿƸдΪ clean_jobs: true⵼¿ÿʮͱǿ̼ʵ֮ǰ 5Ƶ Anti-Crash Ѿ㹻̼Ҫȫǿ true
  2. **˫ؽȾ (addJob ߼ & shouldForward ©)**
     - outer_main.go е shouldForward ߼© state == "FEE"  InBandFeeActive == false ʱȻת notifyȻյ Main Pool ֮ǰ· jobصǣouter_main.go  Main Pool Job ʱʹ cMode := s.CurrentFeeMode DevFee ڼյ񱻴Ϊ FeeModeDev
     - л Fee صǰ 10 ڣûյ clean_jobs: trueھصľύProxy 񱻱Ϊ FeeModeDevͽ·ɸ F2PoolF2Pool յ (Poolin)  Job IDֱӱ unknown-coin / Error 20
  3. **F2Pool 2T ֮ (VarDiff ʼż)**F2Pool  BTC ĬϳʼѶȸߴ 524288һ 2 ӵ DevFee ˮڣ450T Ŀҵ㹻 ShareǰĽȾǰ 30 ȫύЧ ShareF2Pool սյЧ Share ٣ص͹ 2T
* **ռ޸ (v2.2.118-beta)**
  1. ****ɾ outer_main.go ת·е orceCleanJobs 5ƵָƽС
  2. **նȾ**޸ ddJobǿ outer_main.go յΪ FeeModeNone outer_fee.go  isFirstFeeNotifyȷл Fee ص**һ** clean_jobs: true˲տɶУֹط Share ӿ F2Pool
  3. **ά** F2Pool ԣ outer_fee.go е FeeFixedDifficulty == "auto" ЭΪ BTC/BCH Ҵ IsF2PoolExploit ʱǿ mining.authorize  password ֶע d=65536ǿ F2Pool ʼѶȣöʱˮҲܻȡܼ Share׼ԭʵ

## [2026-07-14] Ѷע (v2.2.119-beta)
* **¼**û飬 v2.2.118-beta ж F2Pool ǿע d=65536 Ȩ߼ָΪԭеġѶڸǣDifficulty Maskingģʽ
* **ԭ**
  1. ֻҪ޸˽Ⱦ Bug 6  200Ѷȵ Share Ҳܱ 100% աذѶȨؼ㣬Իƽȴﵽ 15TҪѶȡ
  2. Ѷڸǣÿȫ̺޲ 200Ѷ¹κǱڵĵգʵ߼ˮ

### v2.2.120-beta (2026-07-15) - ޸ F2Pool ˮЧϷ
1. **F2Pool ˮЧ޸**: ޸˴ڳˮģʽ (InBandFeeActive) ڽ FEE ״̬ʱ͸ (shouldForward = true)  Bug⵼¿ڳˮڼյ (Poolin) ĸƵ񲢽м㣬ȻЩ F2Pool ƥķݶύ F2PoolӶ F2Pool ʾ 2T  100% ܾƳжϺ󣬳ˮڼȷأרִ F2Pool 
2. ****: ֤˿ʱ 3 ̨ S21 ͬһ뼯 (  1:39:16) յ mining.notify ̵ߵԭΪӪ/GFW  DPI (Ȱ)  Stratum £עα TCP RST ѽûܻײܽ⡣


### v2.2.120-beta (2026-07-15) - 修复 F2Pool 抽水无效及网络阻断分析
1. **F2Pool 抽水无效修复**: 修复了带内抽水模式 (InBandFeeActive) 在进入 FEE 状态时错误地允许透传主矿池任务 (shouldForward = true) 的 Bug。这导致矿机在抽水期间接收到了主矿池 (Poolin) 的高频任务并进行计算，然后将这些与 F2Pool 不匹配的份额提交给 F2Pool，从而导致了 F2Pool 算力仅显示 2T 和 100% 拒绝。移除该判断后，抽水期间主矿池任务被正确拦截，矿机可专心执行 F2Pool 任务。
2. **矿机掉线现象分析**: 验证了矿场真机部署时 3 台 S21 同一秒集体掉线 (如 11:39:16) 和收到 mining.notify 后立刻掉线的原因，为运营商/GFW 的 DPI (深度包检测) 拦截明文 Stratum 流量所致（注入伪造的 TCP RST 包）。已建议用户采用隧道加密或底层网络加密解决阻断问题。

## [2026-07-15] 零损耗抽水与 TCP 断连彻底修复 (v2.2.129-beta)
* **背景：** 为了解决 S21 频繁掉线、ETC 开发者抽水为 0、以及 BTC 切回主池算力偏高等问题，进行了底层的协议重构。
* **修复内容：**
  1. **彻底拆除 [Anti-Crash]**：F2Pool 会在几十毫秒内连续发送两条 \mining.notify\，第一条可能是坏包，第二条是修正包。之前拦截了第二条导致 S21 内核崩溃发生 TCP RST。现在完全透传，不再丢包。
  2. **修复 ETC 抽水**：ETC/ETH 矿机（Stratum 协议）切池时必须要有 \clean_jobs: true\ 冲刷流水线，否则算力为 0。恢复了仅在进入抽水池的**第一条**任务进行强制冲刷。
  3. **修复 BTC 算力虚高**：切回主池时，代理强制将 \CurrentDiff\ 恢复为 \MainDifficulty\，防止前端 UI 算力因为错位而虚高。
  4. **零损耗平滑过渡**：BTC 切池和恢复时，只下发 \set_extranonce\ 和 \set_difficulty\，**不再**发送 \clean_jobs: true\ 的空包，完美实现 0 算力断层的无缝切换。

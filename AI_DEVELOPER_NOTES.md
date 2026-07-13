# 核心防呆指南与避坑记录 (AI Developer Notes)

> 本文档用于记录在 MinerLink-Proxy 项目中踩过的深坑。每次重构、修复或新增功能前，必须静默查阅此文档，**绝对禁止**修改或破坏以下确立的红线规则。

## 1. 0延迟软切换与 RST 断线防范 (Zero-Latency Fee Switching Exploit)
**【红线规则】绝对禁止使用 TCP RST 踢下线物理矿机！**
- **历史惨痛教训**：曾提倡使用 `SetLinger(0)` 生成 RST 包强制重置矿机状态，但这导致特定老矿机（如 jjz390i）死机长达 10 分钟。
- **当前标准 (v2.2.112-beta起)**：
  - 抽水结束 (EndFee) 时，必须使用 `GlobalDispatcher` 强行向矿机注入伪造任务，进行 **0 延迟无感软切换**。
  - 遭遇矿池踢线触发 Auto-Reconnect 期间，**绝对禁止**立刻下发新的高难度 `mining.notify` 给矿机。必须进行拦截（`SuppressNextNotify=true`），否则会触发矿机进度清零，陷入无法在 25s 内解出 Share 的无限断线死循环！
  - 代理内部必须挂载每 15 秒一次的 `KeepAliveLoop`，防止静默期间被矿池踢下线。

## 2. 前端拦截份额防溢出与并发状态锁定 (v2.2.113-beta)
**【红线规则】独立协程绝不允许回源查询全局状态！**
- **双倍叠加与虚假份额漏洞**：早期在 `readFeeLoop` 处理抽水返回时，错误读取了可能已超时的全局状态 `pending.FeeMode` / `s.CurrentFeeMode`，将迟到的作者抽水 (DevFee) 错误计入运营者 `FeeShares++`，甚至由于冗余代码块导致算力曲线翻倍。
- **当前标准**：
  - 独立 `FeeConn` 生命周期内，必须直接使用创建时捕获的 **`isDevMode` 静态布尔值**！
  - 只要 `isDevMode` 为 `true`，无论返回多迟，必须在底层隐身：绝对只递增 `ValidShares++`，**绝不允许递增 `FeeShares++`**。

## 3. 固件防御与协议护航 (矿机保护盾)
**【红线规则】必须保护物理矿机脆弱的固件进程！**
- **S21 防炸机拦截阀 (Notify Rate Limiter)**：部分新型矿机如果在短时间内（小于 5 秒）连续收到多个 `clean_jobs: false` 的通知任务，固件的任务队列会溢出并直接切断 TCP 导致重启（87秒真空期）。下发 Notify 前必须静默丢弃过于频繁的旧高度任务。
- **1秒断线死循环 (Extranonce 拦截)**：绝不在运行中途转发 `mining.set_extranonce` 给矿机！一旦下发，大部分 ASIC 矿机会强制重启算力板，导致 1 分钟算力扑空。只在代理内部缓存，**绝对隔离**。
- **ETC 协议假死防范 (ID 999999 Bug)**：在 ETH_PROXY 协议中，为了实现无缝重定向，代理下发了 `id: 999999` 的 `eth_getWork` 伪造请求。当矿池返回 `{"id": 999999, "result": [...]}` 时，如果未作拦截，会原样转发给未发送该 ID 的物理矿机，直接导致连接崩溃。必须静默拦截该特定 ID。

## 4. 排班器与调度器灾难防范 (Scheduler Survival Guide)
**【红线规则】任何状态列表的操作必须具备绝对确定性！**
- **级联跳过 Bug (Cascading Shift)**：早期由于矿机重连会导致 Session ID 变化，在 `scheduler.go` 的全局数组排序中被强行置底。这引发了数组向左的级联位移，导致时间轴完美“跳过” 50% 的矿机。**解决标准**：彻底废弃 Session ID 排序，强制改为基于 `GetMinerIdentifier()` (钱包.矿机) 的**绝对确定性哈希排序**，彻底根治大规模漏抽水。
- **幽灵僵尸会话 (IsOffline Array Bloat)**：真实断开的矿机会将 `IsOffline` 设为 `true` 等待 10 分钟。如果在调度器遍历时不主动剔除或过滤这些会话，它们会强占排班队列的坑位，导致计算出的 spacing 被无限拉宽，真实矿机永远等不到抽水周期。**解决标准**：遍历排序时必须立刻过滤判定 `IsOffline`。

## 5. 协议并发与锁安全隔离
**【红线规则】严禁同步阻塞 I/O 及携锁进行网络通信！**
- **Zero-Copy Defense**：单机承载上万并发矿机，`readMinerLoop` 等核心协程中，严禁在每次心跳通信时 `json.Unmarshal` 组装巨大的 Map。优先使用极简 Struct 或纯正则替换以压平 GC。
- **Deadlock Immunity**：写入本地状态必须先 `s.mu.Lock()`，但在向网络（Miner/Main/Fee Conn）发送数据前，必须 **先释放锁 `s.mu.Unlock()`**。携锁进行网络通信极易引发数千个协程堵死。

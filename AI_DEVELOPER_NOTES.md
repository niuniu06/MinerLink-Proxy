# 核心防呆指南与避坑记录 (AI Developer Notes)

> 本文档用于记录在 MinerLink-Proxy 项目中踩过的深坑。每次重构、修复或新增功能前，必须静默查阅此文档，**绝对禁止**修改或破坏以下确立的红线规则。

## 1. Zero-Latency Fee Switching Exploit (无缝抽水与零延迟软切换)
**【红线规则】绝对禁止使用 TCP RST 踢下线物理矿机！**
- **历史惨痛教训**：在 `v2.2.110` 曾提倡使用 `SetLinger(0)` 生成 RST 包来强制重置矿机状态。但这会导致特定型号矿机（如 jjz390i）死机长达 10 分钟。
- **当前标准 (v2.2.112-beta起)**：
  - 在跨矿池抽水结束 (EndFee) 时，必须使用 `GlobalDispatcher` 强行向矿机注入伪造任务，进行 **0 延迟无感软切换**。
  - 在代理遭遇矿池踢线触发 Auto-Reconnect 期间，**绝对禁止**立刻下发新的高难度 `mining.notify` 和 `mining.set_difficulty` 给矿机。必须进行拦截（设置 `SuppressNextNotify=true`），否则会触发矿机进度清零，导致无法在 25s 内解出 Share 从而陷入无限断线的死循环！
  - 代理内部必须挂载每 15 秒一次的 `KeepAliveLoop`，防止静默期间被矿池单方面踢下线。

## 2. 前端拦截份额防溢出与并发状态锁定 (v2.2.113-beta)
**【红线规则】独立协程绝不允许回源查询全局状态！**
- **双倍叠加与虚假份额漏洞**：早期在 `readFeeLoop` 处理抽水返回 `{"result":true}` 时，残留了冗余的代码块。由于闭包超时，全局状态 `s.CurrentFeeMode` 可能已经从 `FeeModeDev` 变为 `FeeModeNone`。如果代码试图动态查询 `pending.FeeMode`，就会将迟到的作者抽水 (DevFee) 错误判断为运营者抽水，从而错误递增 `FeeShares++` 和前端拦截统计，甚至导致算力曲线翻倍。
- **当前标准**：
  - 在独立的 `FeeConn` 连接生命周期内，必须直接使用创建该抽水协程时捕获的 **`isDevMode` 静态布尔值**！
  - 只要 `isDevMode` 为 `true`（作者暗抽水），无论返回的 Share 有多迟，必须在底层将其隐身：绝对只递增 `ValidShares++`，**绝不允许递增 `FeeShares++`**，真正做到作者抽水在前端面板“滴水不漏”。

## 3. 协议并发与锁安全隔离
**【红线规则】严禁同步阻塞 I/O 及携锁进行网络通信！**
- **Zero-Copy Defense**：由于单机需承载上万并发矿机，`readMinerLoop` 等收发核心协程中，严禁在每次心跳通信时 `json.Unmarshal` 组装巨大的 `map[string]interface{}`。必须优先使用极简 Struct 或纯正则替换以压平 GC 毛刺。
- **Deadlock Immunity**：写入本地状态必须先 `s.mu.Lock()`，但在向网络（`MinerConn`, `MainConn`, `FeeConn`）或 Channel 发送数据前，必须 **先释放锁 `s.mu.Unlock()`**。携锁进行网络通信是不可饶恕的死罪，极易引发数千个协程堵死。

## 4. 固件防御与协议护航
**【红线规则】必须保护物理矿机脆弱的固件进程！**
- **S21 / Hyd 防炸机拦截阀 (Notify Rate Limiter)**：部分新型矿机如果在短时间内（小于 5 秒）连续收到多个 `clean_jobs: false` 的通知任务，固件的任务队列会溢出并直接切断 TCP 导致重启（87秒真空期）。代理层必须在下发 Notify 前核对时间差，静默丢弃过于频繁的旧高度任务，完美护航算力。
- **Extranonce 防断线拦截**：绝不在运行中途转发 `mining.set_extranonce` 给矿机！一旦下发，大部分 ASIC 矿机会强制重启算力板，导致抽水时段内矿机 1 分钟没有任何算力输出（抽水全部扑空）。必须在代理内部做隔离。

---
*注：文档已于 v2.2.113-beta 经过清洗，修复了早期 UTF-16/UTF-8 编码错乱问题，并清除了关于 TCP RST 的历史冲突错误指导。*

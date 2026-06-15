# MinerLink-Proxy / Go-Proxy 核心开发备忘录 (AI Developer Notes)

这份文档旨在记录和沉淀本项目在多次迭代中，关于底层逻辑修复、UI 显示差异以及架构设计的核心决策。
每次启动新会话或排查遗留问题时，AI 助手将优先查阅此档，以防止历史修复被遗漏或覆盖。

## 1. 抽水与界面设计逻辑 (UI & Fee Logic)

### 公版 (go-proxy) vs 私有版 (MinerLink-Proxy)
*   **代码基线：** 两者共享同一个底层 `main` 分支代码，不维护两套独立的逻辑。
*   **UI 隐藏策略：**
    *   **私有版 (MinerLink-Proxy)：** 核心思想是“傻瓜化”和“隐蔽化”。在每次发布编译前，通过回退或检出指定的历史 Commit（如 `3d8a98c`），在 `ConfigModal.vue` 中隐藏【作者抽水设置】以及【高级黑科技设定】选项。底层 `session.go` 将强行使用隐式设置的默认值（`devWallet` 注入为 `linkpro168`，比例注入为 `2.0`）。
    *   **公版 (go-proxy)：** 面向开源社区，所有高级黑科技开关及抽水比例完全开放，不作任何 UI 隐藏。
*   **UI 品牌隔离：** 2026/06 迭代中，已将前端的全部文字、Logo 及标题统一重构为 `MinerLink-Proxy`，彻底解决了用户误以为“安装错误”的混淆。

### 作者抽水 (DevFee) 与 运营者抽水 (OpFee) 隔离
*   最初代码存在 `DevFee` 覆写 `OpFee` 的 Bug。
*   **修复方案：** 引入 `isDevMode` 状态标志。当触发 `DevFee` 时，底层严格锁定 `linkpro168`（作者钱包）；触发 `OpFee` 时，严格走向面板配置的运营者钱包。二者并行互不干扰。

## 2. SmartRouting 智能回退与矿池兼容性 (Smart Routing & Pool Compatibility)

### 2miners 与不支持子账户矿池的问题
*   **现象：** 当用户的代理矿机使用原生的 `0x...` 钱包挖矿，且代理的目标矿池不支持子账户名（如 `2miners`），若在此矿池同池抽水（使用 `linkpro168`），会因 Auth Failed 被矿池封禁 IP 长达 5 小时。
*   **解决方案 (v2.0.52)：** 在 `ConnectFee` 引入 `!isSubAccount && !hasSpecificWallet` 判定逻辑。一旦检测到冲突，彻底跳过同池抽水，强制使用备用抽水矿池 (`FeePoolAddress`)。

### 备用矿池留空导致的 0 算力 Bug (v2.0.61)
*   **现象：** 引入强制回退逻辑后，若前端的【备用抽水矿池】留空，底层 `host` 变为空字符串 `""`，导致每次抽水瞬间报错 `missing address`，进而不断回退，长达几小时无任何抽水份额。
*   **修复方案：** 在 `session.go` 中植入“全币种内置急救矩阵”。当 `host == ""` 且触发强制回退时，按币种自动赋予默认矿池（如 ETC 切 `asia-etc.f2pool.com:8118`，BTC 切 `stratum.f2pool.com:3333`）。

## 3. ETH_PROXY 协议下的零延迟下发 (Zero-Latency Job Injection)

*   **现象：** `eth_submitwork` 协议下切池时份额会全跑回主矿池。
*   **修复方案：** 在认证通过后，强制程序向抽水矿池伪造发送一次 `{"id": 0, "method": "eth_getwork"}` 请求包，诱导矿池主动下发新任务。突破了原先对 `ETH_PROXY` 协议跳过下发任务的屏蔽限制。

## 4. 数据库与启动端口 Bug

*   **现象：** 一键安装脚本设定的自定义端口（如 10010）在全新服务器上未生效，强制绑定在 8080 端口。
*   **原因：** `proxy.db` 首次创建时，GORM 利用 `gorm:"default:8080"` 注入了默认值，优先级超过了 CLI 参数 `-api-port`。
*   **修复方案 (v2.0.60)：** 在 `main.go` 启动时新增检测逻辑。如果首次启动（DB 值为 8080 但 `-api-port` 不同），则强行以 CLI 参数为最高特权，并顺手将此值覆写回 DB。

---
**维护建议：** 之后的任何关键业务逻辑修改，必须在此文档追加记录。

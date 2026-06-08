### 🚀 v2.0.0-beta: 深度架构升级与商业化矿池基座 (Core Architecture Upgrade)

**✨ 核心更新 (Enhancements)**
本次升级成功为引擎植入了四大核心商业级特性，彻底剥离了早期野蛮生长的弊端，剑指极致压榨算力：

1. **幽灵连接监控 (Session Watchdog)**
   - 引入全局死神心跳监测协程。
   - 当矿机超过 15 分钟未提交任何有效份额且无 Ping 响应时，系统将在 TCP 协议层强制发起 `RST` 阻断，彻底销毁冗余 `goroutine`，即使在万人级矿池并发下也**永不 OOM**。

2. **内网 Job 极速广播树 (Job Target Caching)**
   - 引入全新的 `GlobalDispatcher` 全局分发中心。
   - 采用同源集群零延迟（Zero-latency）目标缓存技术。当某台矿机从主矿池订阅到新难度与新任务时，系统能在千分之一秒内将其横向扇出（Fan-out）给同集群内的兄弟矿机。
   - 史诗级降低跨国物理延迟导致的 Stale（过期）拒绝份额！

3. **抽水强隔离与 100% 结算效率 (ExtraNonce Isolation)**
   - 彻底重构抽水状态机，废弃极易引发报错的“单 Share 劫持”老旧逻辑。
   - 在触发和结束时间切片抽水时，引擎会向底层 ASIC 芯片强制注入伪造的 `clean_jobs=true` 刷新流，并伴随 `mining.set_extranonce` 指令。
   - 强迫矿机清空旧算力缓存，完全适配抽水矿池的独立 `ExtraNonce` 基因。真正实现切换期间算力 100% 有效！

4. **协议升维适配 (AsicBoost & V2 Stub)**
   - **AsicBoost 算力解封**：底层的解析器现已全面支持在握手阶段拦截与透传 `mining.configure`，完美激活新型蚂蚁/神马矿机的 Version-Rolling 版号滚轴加速黑科技。
   - **ETC 协议嗅探器**：新增对 `eth_submitLogin` 等异形请求的底层嗅探。
   - **Stratum V2 蓝图**：植入二进制协议翻译网关 `v2_translator` 的地基代码，为日后支持 Braiins/Foundry 等高度压缩加密的全新 V2 协议做足准备。

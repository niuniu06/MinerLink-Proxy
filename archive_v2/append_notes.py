with open('AI_DEVELOPER_NOTES.md', 'a', encoding='utf-8') as f:
    f.write('\n## 9. v2.2.62 1分钟闪断与面板登出惨案 (The 1-Minute Panic)\n')
    f.write('*   **现象：** v2.2.62 发布后，用户反馈矿机每隔1分钟闪断，同时 Web 面板刚登录一会就强制登出（退回登录页）。\n')
    f.write('*   **真相：** 之前引入的“数据与 TCP 会话解绑 (WorkerStatsManager)”是一个含有致命缺陷的实验性功能。AI 在 `ReapOfflineSessions` 中调用了 `WorkerManager.ReapOldWorkers()`，但由于 `WorkerManager` 未被正确初始化（`Server` 结构体中的 `WorkerManager` 未正确注入，或者被跨协程读取导致空指针），导致发生 `nil pointer dereference`，引发致命的运行时 `panic`。\n')
    f.write('*   **连锁反应：** 代理内核崩溃后被守护进程自动重启，由于 API 的 JWT Secret 是每次进程启动时随机生成的 (`init()` 中的 `rand.Read`)，内核重启导致所有的 Token 全部失效，前端轮询 API 收到 HTTP 401 后强制用户退出。\n')
    f.write('*   **终极修复 (v2.2.63)：** 彻底回退了极度不稳定的 WorkerStatsManager 架构，恢复至 v2.2.61 的稳定底层逻辑，并递增发布版本至 v2.2.63。严禁在未经沙盒与真机压测验证的情况下，在核心收发及心跳垃圾回收 (GC) 链路中注入未被严谨实例化的全局状态管理单例。\n')

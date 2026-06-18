
## 12. 端口启停失败的错误被吞没 Bug (v2.0.77-beta)

*   **现象：** 用户在前端页面点击端口的“启用”或“修改保存”时，即使该端口（例如 3333）已经被其他程序占用，页面依然会立刻弹出绿色的“成功”提示。但实际上后台监听失败，端口并未真正开启。
*   **真相深挖：** 这是一个异步逻辑导致的“欺骗性成功”。在老版本的 `manager.go` 中，`StartProxy` 方法内部会立刻调用 `go func()` 开启一个协程去执行真正的 `server.Start()`。这意味着启动过程是完全异步的。底层的 `net.Listen` 哪怕瞬间报出 `bind: address already in use` 失败，也会被包裹在异步协程里，仅仅打印一行日志然后退出。而 HTTP API 层面根本等不到这个结果，就直接向下执行，返回了 `HTTP 200 Success`。
*   **彻底修复方案：** 重构了 `Manager` 和 `API` 的交互逻辑。将 `Manager.StartProxy` 与 `Manager.RestartProxy` 改造为同步返回 `error`。真正的 `server.Start()` 中的 `net.Listen` 依然保留原有的同步阻塞探测。如果端口占用，会瞬间将 Error 返回给上一级的 HTTP 接口。在 `api.go` 中，一旦捕捉到该 Error，就立刻放弃更新数据库，返回 `HTTP 400 Bad Request` 和错误信息。前端 UI 捕捉到 400 状态码后，会完美弹出原生的报错 Alert，明确告知用户“端口已被占用”。

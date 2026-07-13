import codecs

note = '''
### 2026-06-26 修复 VarDiff 触发蚂蚁矿机掉线 Bug (v2.2.34)
- **背景**: 开启 VarDiff 时，如果矿机算力波动触发难度调整，代理会在下一次主池下发 mining.notify 时连带下发 mining.set_difficulty。
- **原因**: 蚂蚁矿机 (S19等) 对协议解析极其严格，如果 mining.set_difficulty 紧跟的 mining.notify 中的 clean_jobs 参数为 alse (即并非新高度任务)，会导致矿机端解析异常并主动断开 TCP 连接。结合 VarDiff 30秒一次的周期检查，会导致矿机出现极为规律的“每 30 秒掉线一次并立刻重连”的异常现象。
- **修复**: 在 session.go 处理 mining.notify 时增加条件拦截。当且仅当 clean_jobs=true 时，才允许下发累积的 PendingDiff。这样能够确保难度变更完全符合矿机预期的协议生命周期，彻底消灭掉线重连问题。
'''

with codecs.open(r'C:\Users\ba876\.gemini\antigravity\scratch\go-proxy\AI_DEVELOPER_NOTES.md', 'a', 'utf-8') as f:
    f.write(note)


## 14. install.sh 乱码与回车符二次破坏修复 (v2.0.81-beta)

*   **现象：** 用户在 Linux 下载 `install.sh` 运行再次报错 `\r: command not found` 且显示乱码字符 `MinerLink-Proxy ҵȶ`。
*   **原因分析：** 
    1. 在 Windows 环境下，Git 的 `core.autocrlf=true` 机制会在检出文件时自动将 `LF` 转换为 `CRLF`。
    2. 当运行之前的 `fix_crlf.ps1` 脚本修复 `\r` 符号时，PowerShell 默认以系统 ANSI 编码（即 GBK）读取并没有 BOM 的 UTF-8 文件，导致中文字符在内存中被错误解析。
    3. PowerShell 随后将这些已被破坏的“乱码字符”转换回 UTF-8 并写入文件，导致真正的文件内容永久损坏。同时，在重新上传发布后，文件依然被赋予了 CRLF 换行符。
*   **解决方案：**
    1. **恢复原版文件**：使用 `git checkout origin/main -- install.sh` 从远程仓库直接拉取未被破坏的原始纯净版本。
    2. **强制版本控制规则**：在项目根目录新建 `.gitattributes` 文件，强制声明 `*.sh text eol=lf`，彻底禁止 Git 在任何操作系统上对 `.sh` 脚本进行 CRLF 转换。
    3. **安全的二进制替换**：废弃使用 PowerShell 处理脚本换行符。改用 Python `data.replace(b'\r\n', b'\n')`，以纯二进制流的方式剥离回车符，此操作完全绕过字符编码解析，确保中文字符的 100% 完整与安全。

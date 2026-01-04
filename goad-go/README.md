# Goad

Goad 是 [Toad](https://github.com/willmcgugan/toad) 的 Go 语言实现，提供统一的终端 AI 代理界面。

基于 [ACP (Agent Client Protocol)](https://agentclientprotocol.com/) 协议，支持多种 AI 编程代理的无缝集成。

## 特性

- **多代理支持**: Claude Code, OpenHands, Aider, Goose 等 14+ 代理
- **TUI 界面**: 基于 Bubble Tea 的现代终端界面
- **代码高亮**: 支持 40+ 编程语言语法高亮
- **会话管理**: 会话持久化、恢复、导出
- **插件系统**: 可扩展的钩子机制
- **性能监控**: 内置指标收集和仪表盘

## 安装

```bash
cd goad-go
go build -o goad ./cmd/goad/main.go
```

## 使用

```bash
# 启动 TUI (默认代理)
./goad

# 指定代理
./goad -a claude-code

# 指定工作目录
./goad -d /path/to/project

# 列出可用代理
./goad list

# 查看历史会话
./goad history

# 导出会话
./goad export <session-id> [output-file]
```

## 快捷键

| 快捷键 | 功能 |
|--------|------|
| Ctrl+Enter | 发送消息 |
| Ctrl+C / Esc | 取消/退出 |
| Ctrl+P | 文件模糊搜索 |
| Ctrl+R | 恢复历史会话 |
| Ctrl+T | 折叠/展开思考过程 |
| Ctrl+L | 清屏 |
| Ctrl+B | 切换侧边栏 |
| Tab | 自动补全 |
| F1 | 帮助 |
| F2 | 设置 |
| F3 | 代理编辑器 |
| F4 | 性能监控 |
| PgUp/PgDn | 翻页 |

## 斜杠命令

```
/help      - 显示帮助
/clear     - 清除对话
/exit      - 退出程序
/settings  - 打开设置
/model     - 切换模型
/mode      - 切换模式
/export    - 导出会话
/history   - 查看历史
/cancel    - 取消操作
/compact   - 压缩对话
```

## 配置

配置文件位于 `~/.config/goad/`:

```
~/.config/goad/
├── config.toml      # 应用配置
├── agents/          # 自定义代理
├── sessions/        # 会话存储
├── plugins/         # 插件目录
└── history.jsonl    # 命令历史
```

### 环境变量

```bash
# Claude Code
ANTHROPIC_AUTH_TOKEN=<token>
ANTHROPIC_BASE_URL=<url>

# OpenAI
OPENAI_API_KEY=<key>

# Gemini
GOOGLE_API_KEY=<key>
```

## 项目结构

```
goad-go/
├── cmd/
│   ├── goad/           # 主程序入口
│   └── test/           # 测试程序
├── internal/
│   ├── acp/            # ACP 协议实现
│   ├── agent/          # 代理管理
│   ├── config/         # 配置管理
│   ├── hotreload/      # 配置热更新
│   ├── jsonrpc/        # JSON-RPC 通信
│   ├── metrics/        # 性能监控
│   ├── plugin/         # 插件系统
│   ├── session/        # 会话管理
│   ├── terminal/       # PTY 终端
│   └── tui/            # 终端界面
│       ├── commands/   # 斜杠命令
│       ├── complete/   # 自动补全
│       ├── components/ # UI 组件
│       ├── danger/     # 危险检测
│       ├── diff/       # 差异显示
│       ├── highlight/  # 语法高亮
│       ├── history/    # 命令历史
│       ├── markdown/   # Markdown 渲染
│       ├── styles/     # 样式定义
│       └── views/      # 视图页面
└── agents/             # 内置代理配置
```

## 依赖

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI 框架
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) - 样式库
- [Glamour](https://github.com/charmbracelet/glamour) - Markdown 渲染
- [Chroma](https://github.com/alecthomas/chroma) - 语法高亮
- [fsnotify](https://github.com/fsnotify/fsnotify) - 文件监视

## 支持的代理

| 代理 | 标识 | 协议 |
|------|------|------|
| Claude Code | claude-code | ACP |
| OpenHands | open-hands | ACP |
| Aider | aider | ACP |
| Goose | goose | ACP |
| Codex CLI | codex-cli | ACP |
| Gemini CLI | gemini-cli | ACP |
| ... | ... | ... |

## 许可证

MIT License

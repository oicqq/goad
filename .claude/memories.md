# 项目记忆

## API配置信息

### Claude Code
- **AUTH_TOKEN**: `sk-1xZLD7dGf8g8IgQWkwxwMmpUSnbhUhfyIT9F1bugnmK1f5hB`
- **BASE_URL**: `https://pmpjfbhq.cn-nb1.rainapp.top`

### 环境变量
```bash
ANTHROPIC_AUTH_TOKEN=sk-1xZLD7dGf8g8IgQWkwxwMmpUSnbhUhfyIT9F1bugnmK1f5hB
ANTHROPIC_BASE_URL=https://pmpjfbhq.cn-nb1.rainapp.top
```

## 项目信息

- Go版本实现位于: `/workspaces/goad/goad-go/`
- 原Python版本(Toad)位于: `/workspaces/goad/src/toad/`

## 测试结果 (2026-01-03)

### ACP通信测试 - 通过
- claude-code-acp 版本: 0.12.6
- 初始化: 成功
- 会话创建: 成功
- 提示发送: 成功（流式响应正常）
- 可用模型: default (Sonnet 4.5), opus (Opus 4.5), haiku (Haiku 4.5)
- 可用模式: default, acceptEdits, plan, dontAsk, bypassPermissions

## 高优先级功能完成 (2026-01-03)

### 1. 代理配置 - 完成
- 14个代理配置文件全部创建
- 支持用户自定义代理目录: `~/.config/goad/agents/`

### 2. TUI增强 - 完成
- 滚动优化: PgUp/PgDn, Ctrl+U/D, Home/End, 方向键
- 自动滚动状态追踪与指示器
- 代码语法高亮 (chroma库，支持40+语言)
- 差异显示增强 (go-diff库，Unified Diff格式)
- Markdown渲染 (glamour库)

### 3. 终端增强 - 完成
- PTY终端支持 (creack/pty库)
- 终端实时输出流
- 终端会话管理

### Go文件统计
- 总计: 18个Go文件
- 新增模块: highlight, diff, markdown, terminal

### 非TUI测试 (2026-01-03) - 通过
- 初始化: 成功
- 会话创建: 成功 (sessionId: 60de7ff9-e408-402b-9b07-43a4223cfe84)
- 提示发送: 成功
- 响应: "我是 Claude Code，一个基于 Claude Sonnet 4.5 的命令行编程助手..."
- 流式输出: 正常工作

## 中优先级功能完成 (2026-01-03)

### 1. MCP服务器集成
- 扩展MCP服务器类型支持（stdio, http, sse）
- 配置文件支持MCP服务器定义
- 会话创建时传递MCP配置

### 2. 多模型支持
- 保存会话可用模型列表
- 添加SetModel方法切换模型
- 支持default, opus, haiku模型

### 3. 会话管理
- 会话持久化到 ~/.config/goad/sessions/
- 支持会话列表、加载、删除
- 添加 goad history 命令

### 4. 视图完善
- 设置视图: API配置（Claude/OpenAI/Gemini）
- 帮助视图: 快捷键、命令、代理信息

### Go文件统计
- 总计: 20个Go文件
- 新增模块: session, views/help

## 低优先级功能完成 (2026-01-03)

### 1. 斜杠命令支持
- 内置命令: /help, /clear, /exit, /settings, /model, /mode, /export, /history, /cancel, /compact
- 命令注册表和前缀匹配
- 命令解析和帮助格式化

### 2. 路径自动补全
- 路径补全器 (带缓存)
- 最长公共前缀计算
- 支持目录/文件过滤
- 命令补全器 (PATH中的可执行文件)

### 3. 危险命令检测
- 危险等级: Safe, Unknown, Dangerous, Destructive
- 安全命令白名单 (只读操作)
- 危险命令黑名单 (文件系统修改)
- 项目外路径检测 (升级为Destructive)

### 4. 命令历史记录
- JSONL格式持久化
- 上/下键浏览历史
- 历史搜索
- 默认路径: ~/.config/goad/history.jsonl

### 5. 会话导出功能
- 支持三种格式: Markdown, JSON, 纯文本
- goad export <session-id> [output-file] 命令
- 导出包含元信息和完整对话记录

### Go文件统计
- 总计: 24个Go文件
- 新增模块: commands, danger, complete, history

## P0/P1 优化功能完成 (2026-01-03)

### P0: 权限请求对话框增强
- 组件位置: `internal/tui/components/permission.go`
- 详情切换 (Tab键): 显示工具调用位置、内容预览
- 选项类型着色: 允许=绿色, 拒绝=红色
- 居中渲染支持

### P0: 工具调用状态面板
- 组件位置: `internal/tui/components/toolpanel.go`
- 实时追踪工具调用状态 (pending/running/completed/failed)
- 执行时间显示
- 可折叠/展开
- 紧凑视图用于侧边栏

### P1: 模糊搜索 (Ctrl+P)
- 组件位置: `internal/tui/components/fuzzy.go`
- 模糊匹配算法: 连续匹配加分、单词边界加分、前缀匹配加分
- 从目录加载文件 (支持深度限制)
- 自动忽略: .git, node_modules, vendor, __pycache__ 等
- 文件分类标签: Go, Python, JS/TS, Markdown, Config

### P1: 会话恢复 (Ctrl+R)
- 组件位置: `internal/tui/components/session.go`
- 按代理ID加载历史会话
- 会话搜索过滤
- 会话删除功能
- 显示会话元信息: 代理、消息数、更新时间

### P1: 思考过程折叠显示 (Ctrl+T)
- 组件位置: `internal/tui/components/thinking.go`
- 思考块管理: 自动分块、时间追踪
- 折叠/展开切换 (单块或全部)
- 内容预览 (折叠时显示前50字符)
- 紧凑视图用于侧边栏

### TUI集成
- 新增快捷键: Ctrl+P (搜索), Ctrl+R (恢复), Ctrl+T (折叠思考)
- 模态框覆盖层渲染系统
- 底部状态栏: 滚动位置、工具调用计数
- 组件初始化和事件处理整合

### Go文件统计
- 总计: 29个Go文件
- 新增模块: components (permission, toolpanel, fuzzy, session, thinking)

## P2 优化功能完成 (2026-01-04)

### P2: 输入框自动补全 (Tab)
- 组件位置: `internal/tui/components/autocomplete.go`
- 补全类型: 斜杠命令、文件路径、历史记录
- 路径补全: 支持 ~/ 展开、目录缓存
- 命令补全: /help, /clear, /exit, /settings, /model 等
- 历史记录: 自动去重、限制100条

### P2: 多标签会话
- 组件位置: `internal/tui/components/tabs.go`
- 标签状态: Idle, Active, Thinking, Error
- 标签操作: 新建、关闭、切换
- 快捷键: Ctrl+Tab (下一个), Ctrl+Shift+Tab (上一个), Ctrl+W (关闭)
- 最大标签数: 10

### P2: 代理参数编辑器 (F3)
- 组件位置: `internal/tui/components/editor.go`
- 可编辑字段: 名称、描述、URL、类型、欢迎语
- 只读字段: 短名称、标识、运行命令、协议
- 字段类型: String, Int, Bool, Select, Text
- 编辑模式: 光标移动、插入、删除
- 快捷键: Enter (编辑), Tab (切换选项), Ctrl+S (保存), Esc (取消)

### TUI集成
- 新增快捷键: Tab (补全), F3 (编辑代理), Ctrl+Tab/Shift+Tab (切换标签)
- 标签栏: 多标签时显示在头部
- 自动补全浮动层

### Go文件统计
- 总计: 32个Go文件
- 新增模块: autocomplete, tabs, editor


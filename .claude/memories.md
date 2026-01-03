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


// Package server 提供SSH服务器功能，允许远程访问Goad TUI
package server

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"

	"github.com/anthropics/goad/internal/agent"
	"github.com/anthropics/goad/internal/config"
	"github.com/anthropics/goad/internal/tui"
)

// Server SSH服务器
type Server struct {
	host       string
	port       int
	keyPath    string
	projectDir string
	agentName  string
	agentsFS   embed.FS

	server *ssh.Server
}

// Options 服务器配置选项
type Options struct {
	Host       string   // 监听地址，默认 localhost
	Port       int      // 监听端口，默认 2222
	KeyPath    string   // SSH密钥路径，为空则自动生成
	ProjectDir string   // 项目目录
	AgentName  string   // 使用的代理名称
	AgentsFS   embed.FS // 嵌入的代理配置
}

// New 创建新的SSH服务器
func New(opts Options) *Server {
	if opts.Host == "" {
		opts.Host = "localhost"
	}
	if opts.Port == 0 {
		opts.Port = 2222
	}
	if opts.ProjectDir == "" {
		opts.ProjectDir = "."
	}
	if opts.AgentName == "" {
		opts.AgentName = "claude"
	}

	return &Server{
		host:       opts.Host,
		port:       opts.Port,
		keyPath:    opts.KeyPath,
		projectDir: opts.ProjectDir,
		agentName:  opts.AgentName,
		agentsFS:   opts.AgentsFS,
	}
}

// Start 启动服务器
func (s *Server) Start() error {
	absPath, err := filepath.Abs(s.projectDir)
	if err != nil {
		return fmt.Errorf("无法解析项目路径: %w", err)
	}

	// 加载配置
	appConfig, err := config.LoadAppConfig()
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	// 加载代理注册表
	registry := config.NewAgentRegistry()
	if err := registry.LoadEmbeddedAgents(s.agentsFS, "data/agents"); err != nil {
		return fmt.Errorf("加载代理配置失败: %w", err)
	}

	// 加载用户自定义代理
	if userAgentsDir, err := config.UserAgentsDir(); err == nil {
		_ = registry.LoadFromDirectory(userAgentsDir)
	}

	// 查找代理配置
	agentConfig, ok := registry.GetByShortName(s.agentName)
	if !ok {
		agentConfig, ok = registry.Get(s.agentName)
	}
	if !ok {
		return fmt.Errorf("未找到代理: %s", s.agentName)
	}

	// 创建Wish服务器选项
	opts := []ssh.Option{
		wish.WithAddress(fmt.Sprintf("%s:%d", s.host, s.port)),
		wish.WithMiddleware(
			bubbletea.Middleware(s.teaHandler(appConfig, agentConfig, absPath)),
			activeterm.Middleware(),
			logging.Middleware(),
		),
	}

	// 设置SSH密钥
	if s.keyPath != "" {
		opts = append(opts, wish.WithHostKeyPath(s.keyPath))
	} else {
		// 使用默认密钥路径
		configDir, err := config.ConfigDir()
		if err == nil {
			keyPath := filepath.Join(configDir, "ssh_host_key")
			opts = append(opts, wish.WithHostKeyPath(keyPath))
		}
	}

	// 创建服务器
	server, err := wish.NewServer(opts...)
	if err != nil {
		return fmt.Errorf("创建服务器失败: %w", err)
	}
	s.server = server

	// 启动信号处理
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	log.Printf("Goad SSH服务器启动在 %s:%d", s.host, s.port)
	log.Printf("使用 ssh %s -p %d 连接", s.host, s.port)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != ssh.ErrServerClosed {
			log.Printf("服务器错误: %v", err)
		}
	}()

	<-done
	log.Println("正在关闭服务器...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return server.Shutdown(ctx)
}

// teaHandler 创建Bubble Tea处理器
func (s *Server) teaHandler(appConfig *config.AppConfig, agentConfig *config.AgentConfig, projectDir string) bubbletea.Handler {
	return func(sess ssh.Session) (tea.Model, []tea.ProgramOption) {
		// 获取终端信息
		pty, _, _ := sess.Pty()

		// 创建SSH会话的渲染器
		renderer := bubbletea.MakeRenderer(sess)

		// 创建代理实例
		ag := agent.NewAgentWithAppConfig(agentConfig, appConfig.API, appConfig, projectDir)

		// 启动代理
		if err := ag.Start(); err != nil {
			log.Printf("启动代理失败: %v", err)
			// 返回一个错误模型
			return newErrorModel(err, renderer), nil
		}

		// 创建TUI模型（传递渲染器）
		model := tui.NewWithRenderer(ag, appConfig, agentConfig, renderer)

		// 设置初始窗口大小
		model = model.SetSize(pty.Window.Width, pty.Window.Height)

		return model, []tea.ProgramOption{tea.WithAltScreen()}
	}
}

// Stop 停止服务器
func (s *Server) Stop() error {
	if s.server == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return s.server.Shutdown(ctx)
}

// errorModel 错误模型
type errorModel struct {
	err      error
	renderer *lipgloss.Renderer
}

func newErrorModel(err error, renderer *lipgloss.Renderer) errorModel {
	return errorModel{err: err, renderer: renderer}
}

func (m errorModel) Init() tea.Cmd {
	return nil
}

func (m errorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.KeyMsg:
		return m, tea.Quit
	}
	return m, nil
}

func (m errorModel) View() string {
	style := m.renderer.NewStyle().
		Foreground(lipgloss.Color("9")).
		Bold(true)
	return style.Render(fmt.Sprintf("错误: %v\n\n按任意键退出", m.err))
}

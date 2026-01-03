// Package config 实现了配置管理，包括代理配置和API密钥管理
package config

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/BurntSushi/toml"
)

// AgentProtocol 代理协议类型
type AgentProtocol string

const (
	ProtocolACP AgentProtocol = "acp"
)

// AgentType 代理类型
type AgentType string

const (
	AgentTypeCoding AgentType = "coding"
	AgentTypeChat   AgentType = "chat"
)

// AgentAction 代理操作
type AgentAction struct {
	Command     string `toml:"command"`
	Description string `toml:"description"`
}

// AgentConfig 代理配置
type AgentConfig struct {
	Identity      string                        `toml:"identity"`
	Name          string                        `toml:"name"`
	ShortName     string                        `toml:"short_name"`
	URL           string                        `toml:"url"`
	Protocol      AgentProtocol                 `toml:"protocol"`
	Type          AgentType                     `toml:"type"`
	AuthorName    string                        `toml:"author_name"`
	AuthorURL     string                        `toml:"author_url"`
	PublisherName string                        `toml:"publisher_name"`
	PublisherURL  string                        `toml:"publisher_url"`
	Description   string                        `toml:"description"`
	Tags          []string                      `toml:"tags"`
	Help          string                        `toml:"help"`
	Welcome       string                        `toml:"welcome"`
	RunCommand    map[string]string             `toml:"run_command"`
	Actions       map[string]map[string]AgentAction `toml:"actions"`
	Active        bool                          `toml:"active"`
	Recommended   bool                          `toml:"recommended"`
}

// GetRunCommand 获取当前操作系统的运行命令
func (c *AgentConfig) GetRunCommand() string {
	// 首先尝试当前操作系统
	osName := runtime.GOOS
	if cmd, ok := c.RunCommand[osName]; ok {
		return cmd
	}
	// 尝试通配符
	if cmd, ok := c.RunCommand["*"]; ok {
		return cmd
	}
	return ""
}

// APIConfig API配置
type APIConfig struct {
	// Claude Code
	ClaudeAPIKey     string `json:"claude_api_key" toml:"claude_api_key"`
	ClaudeBaseURL    string `json:"claude_base_url" toml:"claude_base_url"`

	// OpenAI Codex
	OpenAIAPIKey     string `json:"openai_api_key" toml:"openai_api_key"`
	OpenAIBaseURL    string `json:"openai_base_url" toml:"openai_base_url"`

	// Gemini
	GeminiAPIKey     string `json:"gemini_api_key" toml:"gemini_api_key"`
	GeminiBaseURL    string `json:"gemini_base_url" toml:"gemini_base_url"`

	// 通用
	CustomEnv        map[string]string `json:"custom_env" toml:"custom_env"`
}

// GetEnvVars 获取代理需要的环境变量
func (c *APIConfig) GetEnvVars(agentIdentity string) map[string]string {
	env := make(map[string]string)

	switch agentIdentity {
	case "claude.com":
		if c.ClaudeAPIKey != "" {
			env["ANTHROPIC_API_KEY"] = c.ClaudeAPIKey
			env["ANTHROPIC_AUTH_TOKEN"] = c.ClaudeAPIKey // Claude Code 使用这个
		}
		if c.ClaudeBaseURL != "" {
			env["ANTHROPIC_BASE_URL"] = c.ClaudeBaseURL
		}
	case "openai.com":
		if c.OpenAIAPIKey != "" {
			env["OPENAI_API_KEY"] = c.OpenAIAPIKey
		}
		if c.OpenAIBaseURL != "" {
			env["OPENAI_BASE_URL"] = c.OpenAIBaseURL
		}
	case "geminicli.com":
		if c.GeminiAPIKey != "" {
			env["GOOGLE_API_KEY"] = c.GeminiAPIKey
			env["GEMINI_API_KEY"] = c.GeminiAPIKey
		}
		if c.GeminiBaseURL != "" {
			env["GEMINI_BASE_URL"] = c.GeminiBaseURL
		}
	}

	// 添加自定义环境变量
	for k, v := range c.CustomEnv {
		env[k] = v
	}

	return env
}

// AppConfig 应用配置
type AppConfig struct {
	DefaultAgent string     `json:"default_agent" toml:"default_agent"`
	API          *APIConfig `json:"api" toml:"api"`
	Theme        string     `json:"theme" toml:"theme"`
	Language     string     `json:"language" toml:"language"`
}

// DefaultAppConfig 返回默认应用配置
func DefaultAppConfig() *AppConfig {
	return &AppConfig{
		DefaultAgent: "claude.com",
		API:          &APIConfig{},
		Theme:        "default",
		Language:     "zh-CN",
	}
}

// ConfigDir 返回配置目录路径
func ConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".config", "goad"), nil
}

// UserAgentsDir 返回用户自定义代理目录路径
func UserAgentsDir() (string, error) {
	configDir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "agents"), nil
}

// EnsureUserAgentsDir 确保用户自定义代理目录存在
func EnsureUserAgentsDir() (string, error) {
	dir, err := UserAgentsDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("创建用户代理目录失败: %w", err)
	}
	return dir, nil
}

// LoadAppConfig 加载应用配置
func LoadAppConfig() (*AppConfig, error) {
	configDir, err := ConfigDir()
	if err != nil {
		return DefaultAppConfig(), nil
	}

	configPath := filepath.Join(configDir, "config.toml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return DefaultAppConfig(), nil
	}

	var config AppConfig
	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	if config.API == nil {
		config.API = &APIConfig{}
	}

	return &config, nil
}

// SaveAppConfig 保存应用配置
func SaveAppConfig(config *AppConfig) error {
	configDir, err := ConfigDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	configPath := filepath.Join(configDir, "config.toml")
	f, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := toml.NewEncoder(f)
	return encoder.Encode(config)
}

// AgentRegistry 代理注册表
type AgentRegistry struct {
	agents map[string]*AgentConfig
}

// NewAgentRegistry 创建代理注册表
func NewAgentRegistry() *AgentRegistry {
	return &AgentRegistry{
		agents: make(map[string]*AgentConfig),
	}
}

// LoadEmbeddedAgents 从嵌入的文件系统加载代理配置
func (r *AgentRegistry) LoadEmbeddedAgents(fs embed.FS, dir string) error {
	entries, err := fs.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("读取嵌入目录失败: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}

		data, err := fs.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}

		var agent AgentConfig
		if _, err := toml.Decode(string(data), &agent); err != nil {
			continue
		}

		// 默认激活
		if agent.Identity != "" {
			r.agents[agent.Identity] = &agent
		}
	}

	return nil
}

// LoadFromDirectory 从目录加载代理配置
func (r *AgentRegistry) LoadFromDirectory(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("读取目录失败: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		var agent AgentConfig
		if _, err := toml.DecodeFile(path, &agent); err != nil {
			continue
		}

		if agent.Identity != "" {
			r.agents[agent.Identity] = &agent
		}
	}

	return nil
}

// Get 获取代理配置
func (r *AgentRegistry) Get(identity string) (*AgentConfig, bool) {
	agent, ok := r.agents[identity]
	return agent, ok
}

// GetByShortName 通过短名称获取代理配置
func (r *AgentRegistry) GetByShortName(shortName string) (*AgentConfig, bool) {
	shortName = strings.ToLower(shortName)
	for _, agent := range r.agents {
		if strings.ToLower(agent.ShortName) == shortName {
			return agent, true
		}
	}
	return nil, false
}

// List 列出所有代理
func (r *AgentRegistry) List() []*AgentConfig {
	agents := make([]*AgentConfig, 0, len(r.agents))
	for _, agent := range r.agents {
		agents = append(agents, agent)
	}
	return agents
}

// ListActive 列出所有激活的代理
func (r *AgentRegistry) ListActive() []*AgentConfig {
	agents := make([]*AgentConfig, 0)
	for _, agent := range r.agents {
		if agent.Active {
			agents = append(agents, agent)
		}
	}
	return agents
}

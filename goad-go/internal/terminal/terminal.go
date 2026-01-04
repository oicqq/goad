// Package terminal 提供PTY终端管理功能
package terminal

import (
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"
)

// Terminal PTY终端
type Terminal struct {
	id      string
	cmd     *exec.Cmd
	pty     *os.File
	rows    int
	cols    int
	running bool
	mu      sync.RWMutex

	// 输出通道
	output chan []byte
	done   chan struct{}
}

// Manager 终端管理器
type Manager struct {
	terminals map[string]*Terminal
	mu        sync.RWMutex
}

// NewManager 创建终端管理器
func NewManager() *Manager {
	return &Manager{
		terminals: make(map[string]*Terminal),
	}
}

// Create 创建新终端
func (m *Manager) Create(id, shell string, rows, cols int, workDir string) (*Terminal, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 默认shell
	if shell == "" {
		shell = os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/sh"
		}
	}

	// 创建命令
	cmd := exec.Command(shell)
	cmd.Dir = workDir
	cmd.Env = os.Environ()

	return m.startTerminal(id, cmd, rows, cols)
}

// CreateWithCommand 创建终端并执行命令（命令完成后自动退出）
func (m *Manager) CreateWithCommand(id, command string, rows, cols int, workDir string) (*Terminal, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 使用 sh -c 执行命令，命令完成后自动退出
	cmd := exec.Command("sh", "-c", command)
	cmd.Dir = workDir
	cmd.Env = os.Environ()

	return m.startTerminal(id, cmd, rows, cols)
}

// startTerminal 启动终端（内部方法）
func (m *Manager) startTerminal(id string, cmd *exec.Cmd, rows, cols int) (*Terminal, error) {
	// 启动PTY
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{
		Rows: uint16(rows),
		Cols: uint16(cols),
	})
	if err != nil {
		return nil, err
	}

	t := &Terminal{
		id:      id,
		cmd:     cmd,
		pty:     ptmx,
		rows:    rows,
		cols:    cols,
		running: true,
		output:  make(chan []byte, 100),
		done:    make(chan struct{}),
	}

	// 启动输出读取
	go t.readOutput()

	// 等待进程结束
	go func() {
		cmd.Wait()
		t.mu.Lock()
		t.running = false
		t.mu.Unlock()
		close(t.done)
	}()

	m.terminals[id] = t
	return t, nil
}

// Get 获取终端
func (m *Manager) Get(id string) (*Terminal, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.terminals[id]
	return t, ok
}

// Close 关闭终端
func (m *Manager) Close(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	t, ok := m.terminals[id]
	if !ok {
		return nil
	}

	t.Close()
	delete(m.terminals, id)
	return nil
}

// CloseAll 关闭所有终端
func (m *Manager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, t := range m.terminals {
		t.Close()
		delete(m.terminals, id)
	}
}

// List 列出所有终端ID
func (m *Manager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]string, 0, len(m.terminals))
	for id := range m.terminals {
		ids = append(ids, id)
	}
	return ids
}

// readOutput 读取终端输出
func (t *Terminal) readOutput() {
	buf := make([]byte, 4096)
	for {
		n, err := t.pty.Read(buf)
		if err != nil {
			if err != io.EOF {
				// 读取错误
			}
			close(t.output)
			return
		}
		if n > 0 {
			data := make([]byte, n)
			copy(data, buf[:n])
			select {
			case t.output <- data:
			default:
				// 输出通道满，丢弃
			}
		}
	}
}

// ID 返回终端ID
func (t *Terminal) ID() string {
	return t.id
}

// Write 写入终端
func (t *Terminal) Write(data []byte) (int, error) {
	return t.pty.Write(data)
}

// WriteString 写入字符串
func (t *Terminal) WriteString(s string) (int, error) {
	return t.Write([]byte(s))
}

// Output 返回输出通道
func (t *Terminal) Output() <-chan []byte {
	return t.output
}

// Done 返回结束信号通道
func (t *Terminal) Done() <-chan struct{} {
	return t.done
}

// Resize 调整终端大小
func (t *Terminal) Resize(rows, cols int) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.pty == nil {
		return nil
	}

	t.rows = rows
	t.cols = cols

	return pty.Setsize(t.pty, &pty.Winsize{
		Rows: uint16(rows),
		Cols: uint16(cols),
	})
}

// IsRunning 检查终端是否运行中
func (t *Terminal) IsRunning() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.running
}

// Close 关闭终端
func (t *Terminal) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.cmd != nil && t.cmd.Process != nil {
		t.cmd.Process.Kill()
	}
	if t.pty != nil {
		return t.pty.Close()
	}
	return nil
}

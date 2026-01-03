// Package danger 实现危险命令检测
package danger

import (
	"path/filepath"
	"strings"
)

// Level 危险等级
type Level int

const (
	Safe       Level = iota // 已知安全的命令
	Unknown                 // 未知命令
	Dangerous               // 可能危险 (可修改文件系统)
	Destructive             // 破坏性 (引用项目外的路径)
)

// String 返回危险等级字符串
func (l Level) String() string {
	switch l {
	case Safe:
		return "safe"
	case Unknown:
		return "unknown"
	case Dangerous:
		return "dangerous"
	case Destructive:
		return "destructive"
	default:
		return "unknown"
	}
}

// 安全命令列表 - 只读操作
var safeCommands = map[string]bool{
	// 显示与输出
	"echo": true, "cat": true, "less": true, "more": true,
	"head": true, "tail": true, "tac": true, "nl": true,
	// 文件目录信息
	"ls": true, "tree": true, "pwd": true, "file": true,
	"stat": true, "du": true, "df": true,
	// 搜索查找
	"find": true, "locate": true, "which": true, "whereis": true,
	"type": true, "grep": true, "egrep": true, "fgrep": true, "rg": true,
	// 文本处理 (只读)
	"wc": true, "sort": true, "uniq": true, "cut": true,
	"paste": true, "column": true, "tr": true, "diff": true,
	"cmp": true, "comm": true,
	// 系统信息
	"whoami": true, "who": true, "w": true, "id": true,
	"hostname": true, "uname": true, "uptime": true, "date": true,
	"cal": true, "env": true, "printenv": true,
	// 进程信息
	"ps": true, "top": true, "htop": true, "pgrep": true,
	"jobs": true, "pstree": true,
	// 网络 (只读)
	"ping": true, "traceroute": true, "nslookup": true, "dig": true,
	"host": true, "netstat": true, "ss": true, "ifconfig": true, "ip": true,
	// 压缩文件查看
	"zcat": true, "zless": true,
	// 帮助
	"history": true, "man": true, "help": true, "info": true,
	"apropos": true, "whatis": true,
	// 校验
	"md5sum": true, "sha256sum": true, "sha1sum": true, "cksum": true, "sum": true,
	// 其他安全命令
	"bc": true, "expr": true, "test": true, "sleep": true,
	"true": true, "false": true, "yes": true, "seq": true,
	"basename": true, "dirname": true, "realpath": true, "readlink": true,
	// Go相关
	"go": true, "gofmt": true,
	// Node相关
	"node": true, "npm": true, "npx": true, "yarn": true, "pnpm": true,
	// Python相关
	"python": true, "python3": true, "pip": true, "pip3": true,
	// Git (大部分操作)
	"git": true,
}

// 危险命令列表 - 可能修改文件系统
var unsafeCommands = map[string]bool{
	// 文件目录创建
	"mkdir": true, "touch": true, "mktemp": true, "mkfifo": true, "mknod": true,
	// 文件目录删除
	"rm": true, "rmdir": true, "shred": true,
	// 移动复制
	"mv": true, "cp": true, "rsync": true, "scp": true, "install": true,
	// 文件修改
	"sed": true, "awk": true, "tee": true,
	// 权限
	"chmod": true, "chown": true, "chgrp": true, "chattr": true, "setfacl": true,
	// 链接
	"ln": true, "link": true, "unlink": true,
	// 压缩解压
	"tar": true, "zip": true, "unzip": true,
	"gzip": true, "gunzip": true, "bzip2": true, "bunzip2": true,
	"xz": true, "unxz": true, "7z": true, "rar": true, "unrar": true,
	// 下载
	"wget": true, "curl": true, "fetch": true, "aria2c": true,
	// 底层磁盘操作
	"dd": true, "truncate": true, "fallocate": true,
	// 文件分割
	"split": true, "csplit": true,
	// 系统管理
	"useradd": true, "userdel": true, "usermod": true,
	"groupadd": true, "groupdel": true, "passwd": true,
	"mount": true, "umount": true, "mkfs": true, "fdisk": true, "parted": true,
	"swapon": true, "swapoff": true,
	// 其他危险命令
	"patch": true,
	// 极度危险
	"sudo": true, "su": true,
}

// 命令分隔符
var commandSplitters = []string{"&&", "||", ";", "|"}

// CommandAtom 命令原子
type CommandAtom struct {
	Name  string // 命令名
	Level Level  // 危险等级
	Path  string // 相关路径
	Start int    // 起始位置
	End   int    // 结束位置
}

// Result 检测结果
type Result struct {
	Level Level
	Atoms []CommandAtom
}

// Detect 检测命令危险等级
func Detect(projectDir, cwd, commandLine string) Result {
	projectDir, _ = filepath.Abs(projectDir)
	cwd, _ = filepath.Abs(cwd)

	atoms := analyze(projectDir, cwd, commandLine)

	maxLevel := Safe
	for _, atom := range atoms {
		if atom.Level > maxLevel {
			maxLevel = atom.Level
		}
	}

	return Result{
		Level: maxLevel,
		Atoms: atoms,
	}
}

// analyze 分析命令
func analyze(projectDir, cwd, commandLine string) []CommandAtom {
	var atoms []CommandAtom

	// 简单分割命令 (不使用bashlex，简化实现)
	commands := splitCommands(commandLine)

	for _, cmd := range commands {
		atom := analyzeCommand(projectDir, cwd, cmd)
		if atom != nil {
			atoms = append(atoms, *atom)
		}
	}

	return atoms
}

// splitCommands 分割命令
func splitCommands(commandLine string) []string {
	var commands []string
	remaining := commandLine

	for _, sep := range commandSplitters {
		var parts []string
		for _, part := range strings.Split(remaining, sep) {
			part = strings.TrimSpace(part)
			if part != "" {
				parts = append(parts, part)
			}
		}
		if len(parts) > 1 {
			commands = append(commands, parts...)
			remaining = ""
			break
		}
	}

	if remaining != "" {
		commands = append(commands, strings.TrimSpace(remaining))
	}

	return commands
}

// analyzeCommand 分析单个命令
func analyzeCommand(projectDir, cwd, command string) *CommandAtom {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return nil
	}

	cmdName := parts[0]
	// 处理路径中的命令
	cmdName = filepath.Base(cmdName)

	level := Unknown

	if safeCommands[cmdName] {
		level = Safe
	} else if unsafeCommands[cmdName] {
		level = Dangerous

		// 检查参数中的路径
		for _, arg := range parts[1:] {
			if strings.HasPrefix(arg, "-") {
				continue
			}

			// 检查是否是项目外的路径
			targetPath := resolvePath(cwd, arg)
			if targetPath != "" && !isSubPath(projectDir, targetPath) {
				level = Destructive
				return &CommandAtom{
					Name:  cmdName,
					Level: level,
					Path:  targetPath,
				}
			}
		}
	}

	// 检查输出重定向
	if strings.Contains(command, ">") || strings.Contains(command, ">>") {
		// 简单检测重定向目标
		for _, arg := range parts {
			if strings.HasPrefix(arg, ">") {
				target := strings.TrimLeft(arg, ">")
				target = strings.TrimSpace(target)
				if target != "" {
					targetPath := resolvePath(cwd, target)
					if targetPath != "" && !isSubPath(projectDir, targetPath) {
						level = Destructive
						return &CommandAtom{
							Name:  "redirect",
							Level: level,
							Path:  targetPath,
						}
					}
				}
			}
		}
	}

	return &CommandAtom{
		Name:  cmdName,
		Level: level,
	}
}

// resolvePath 解析路径
func resolvePath(cwd, path string) string {
	if path == "" {
		return ""
	}

	// 展开 ~
	if strings.HasPrefix(path, "~") {
		// 简化处理，不展开~
		return ""
	}

	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}

	return filepath.Clean(filepath.Join(cwd, path))
}

// isSubPath 检查是否是子路径
func isSubPath(parent, child string) bool {
	parent = filepath.Clean(parent)
	child = filepath.Clean(child)

	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}

	return !strings.HasPrefix(rel, "..")
}

// IsDangerous 检查是否危险
func IsDangerous(level Level) bool {
	return level >= Dangerous
}

// IsDestructive 检查是否破坏性
func IsDestructive(level Level) bool {
	return level >= Destructive
}

// GetWarning 获取警告信息
func GetWarning(result Result) string {
	switch result.Level {
	case Destructive:
		return "警告: 此命令可能影响项目外的文件!"
	case Dangerous:
		return "注意: 此命令可能修改文件系统"
	case Unknown:
		return "提示: 未知命令"
	default:
		return ""
	}
}

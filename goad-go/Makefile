# Goad - AI代理终端
# Makefile

# 版本信息
VERSION := 0.1.0
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Go参数
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT)"

# 输出目录
BUILD_DIR := build
BINARY := goad

.PHONY: all build clean install run test lint help

# 默认目标
all: build

# 构建
build:
	@echo "构建 $(BINARY) $(VERSION)..."
	@go build $(LDFLAGS) -o $(BINARY) ./cmd/goad

# 构建所有平台
build-all: build-linux build-darwin build-windows

build-linux:
	@echo "构建 Linux 版本..."
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-linux-amd64 ./cmd/goad
	@GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-linux-arm64 ./cmd/goad

build-darwin:
	@echo "构建 macOS 版本..."
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-darwin-amd64 ./cmd/goad
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-darwin-arm64 ./cmd/goad

build-windows:
	@echo "构建 Windows 版本..."
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-windows-amd64.exe ./cmd/goad

# 清理
clean:
	@echo "清理..."
	@rm -f $(BINARY)
	@rm -rf $(BUILD_DIR)
	@go clean

# 安装
install: build
	@echo "安装到 $$GOPATH/bin..."
	@go install $(LDFLAGS) ./cmd/goad

# 运行
run: build
	@./$(BINARY)

# 测试
test:
	@echo "运行测试..."
	@go test -v ./...

# 代码检查
lint:
	@echo "运行代码检查..."
	@golangci-lint run ./...

# 格式化
fmt:
	@echo "格式化代码..."
	@go fmt ./...

# 依赖更新
deps:
	@echo "更新依赖..."
	@go mod tidy

# 帮助
help:
	@echo "可用命令:"
	@echo "  make build       - 构建二进制文件"
	@echo "  make build-all   - 构建所有平台的二进制文件"
	@echo "  make clean       - 清理构建产物"
	@echo "  make install     - 安装到 GOPATH/bin"
	@echo "  make run         - 构建并运行"
	@echo "  make test        - 运行测试"
	@echo "  make lint        - 运行代码检查"
	@echo "  make fmt         - 格式化代码"
	@echo "  make deps        - 更新依赖"
	@echo "  make help        - 显示此帮助"

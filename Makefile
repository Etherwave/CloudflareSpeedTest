# Makefile for Go project

# 检测操作系统
ifeq ($(OS),Windows_NT)
    # Windows系统
    RM = del /Q
    TARGET = CloudflareSpeedTest.exe
else
    # Unix-like系统 (Linux, macOS等)
    RM = rm -f
    TARGET = CloudflareSpeedTest
endif

# Go 源码文件
SRCS = main.go

# Go 编译器
GO = go

# 编译标志（可选）
GOFLAGS =

.PHONY: all build run clean

all: build

# 构建可执行文件
build: clean $(TARGET)

$(TARGET): $(SRCS)
	$(GO) build $(GOFLAGS) -o $(TARGET) $(SRCS)

# 运行程序
run: $(SRCS)
	$(GO) run $(GOFLAGS) $(SRCS)

# 清理构建结果
clean:
	$(RM) $(TARGET)
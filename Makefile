include .env.local
export

SRC          := $(shell find . -type f -name '*.go' -not -path "./vendor/*")
TARGETS      := traffic-forwarder
ALL_TARGETS  := $(TARGETS)

.PHONY: help
help: ### Display this help screen.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

# 动态检测操作系统和架构
UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)

# 根据操作系统设置 GOOS
ifeq ($(UNAME_S),Darwin)
    GOOS := darwin
else ifeq ($(UNAME_S),Linux)
    GOOS := linux
else ifeq ($(UNAME_S),FreeBSD)
    GOOS := freebsd
else ifeq ($(UNAME_S),OpenBSD)
    GOOS := openbsd
else ifeq ($(UNAME_S),NetBSD)
    GOOS := netbsd
else ifeq ($(findstring MINGW,$(UNAME_S)),MINGW)
    GOOS := windows
else ifeq ($(findstring MSYS,$(UNAME_S)),MSYS)
    GOOS := windows
else ifeq ($(findstring CYGWIN,$(UNAME_S)),CYGWIN)
    GOOS := windows
else
    GOOS := linux
endif

# 根据架构设置 GOARCH
ifeq ($(UNAME_M),x86_64)
    GOARCH := amd64
else ifeq ($(UNAME_M),amd64)
    GOARCH := amd64
else ifeq ($(UNAME_M),i386)
    GOARCH := 386
else ifeq ($(UNAME_M),i686)
    GOARCH := 386
else ifeq ($(UNAME_M),arm64)
    GOARCH := arm64
else ifeq ($(UNAME_M),aarch64)
    GOARCH := arm64
else ifeq ($(UNAME_M),armv7l)
    GOARCH := arm
else ifeq ($(UNAME_M),armv6l)
    GOARCH := arm
else ifeq ($(UNAME_M),ppc64le)
    GOARCH := ppc64le
else ifeq ($(UNAME_M),s390x)
    GOARCH := s390x
else ifeq ($(UNAME_M),mips64le)
    GOARCH := mips64le
else ifeq ($(UNAME_M),mips64)
    GOARCH := mips64
else ifeq ($(UNAME_M),mipsle)
    GOARCH := mipsle
else ifeq ($(UNAME_M),mips)
    GOARCH := mips
else
    GOARCH := amd64
endif

# 显示检测到的平台信息
.PHONY: platform-info
platform-info: ## Show detected platform information
	@echo "Detected platform:"
	@echo "  OS: $(UNAME_S) -> GOOS=$(GOOS)"
	@echo "  Architecture: $(UNAME_M) -> GOARCH=$(GOARCH)"
	@echo "  Full target: $(GOOS)/$(GOARCH)"

# 构建标志
BUILD_FLAGS := -ldflags="-s -w" # 减小二进制文件大小
ifeq ($(race), 1)
	BUILD_FLAGS += -race
endif
ifeq ($(gc_debug), 1)
	BUILD_FLAGS += -gcflags=all="-N -l"
endif
# 内存优化选项
ifeq ($(memory_optimized), 1)
	BUILD_FLAGS += -gcflags=all="-B"
	LDFLAGS += -X main.memoryOptimized=true
endif
# 性能分析选项
ifeq ($(profile), 1)
	BUILD_FLAGS += -gcflags=all="-cpuprofile=cpu.prof -memprofile=mem.prof"
endif

.PHONY: build
build: platform-info clean $(ALL_TARGETS) ## Build all executable objects.

$(TARGETS): $(SRC)
	@echo "Building for $(GOOS)/$(GOARCH)..."
	@(GOOS=$(GOOS) GOARCH=$(GOARCH) go build -mod vendor $(BUILD_FLAGS) $(PWD)/cmd/$@)

$(TEST_TARGETS): $(SRC)
	@echo "Building test target for $(GOOS)/$(GOARCH)..."
	@(GOOS=$(GOOS) GOARCH=$(GOARCH) go build -mod vendor $(BUILD_FLAGS) $(PWD)/test/$@)

.PHONY: test
test: ## Run tests
	@(GOOS=$(GOOS) GOARCH=$(GOARCH) go test -mod vendor -v ./...)

.PHONY: test-race
test-race: ## Run tests with race detection
	@(GOOS=$(GOOS) GOARCH=$(GOARCH) go test -mod vendor -race -v ./...)

.PHONY: bench
bench: ## Run benchmarks
	@(GOOS=$(GOOS) GOARCH=$(GOARCH) go test -mod vendor -bench=. -benchmem ./...)

.PHONY: clean
clean: ## Clean all executable objects.
	@(rm -f $(ALL_TARGETS))
	@(rm -f *.prof) # 清理性能分析文件

.PHONY: local_run
local_run: build ## Run the application locally.
	@(./traffic-forwarder -conf $(PWD)/etc/traffic-forwarder.conf)

.PHONY: start
start: build ## Start the application using goreman.
	# To install goreman, run `go install github.com/mattn/goreman@latest`
	@(echo "proc1: $(PWD)/traffic-forwarder -conf $(PWD)/etc/traffic-forwarder.conf" > Procfile)
	@(goreman check)
	@(nohup goreman start &)

.PHONY: stop
stop: ## Stop the application using goreman.
	@(goreman run stop-all)

.PHONY: status
status: ## Show the status of the application using goreman.
	@(goreman run status)

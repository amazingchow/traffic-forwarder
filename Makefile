include .env
export

SRC            := $(shell find . -type f -name '*.go' -not -path "./vendor/*")
TARGET         := traffic-forwarder
ALL_TARGETS    := $(TARGET)
UNAME_S        := $(shell uname -s)
UNAME_M        := $(shell uname -m)

.PHONY: help
help: ### Display this help screen.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

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

.PHONY: platform
platform: ## Show detected platform information
	@echo "########################################################"
	@echo "#  OS           : $(GOOS)"
	@echo "#  Architecture : $(GOARCH)"
	@echo "#  Full target  : $(GOOS)/$(GOARCH)"
	@echo "########################################################"

BUILD_FLAGS :=
BUILD_RELEASE_FLAGS :=
ifeq ($(race), 1)
	BUILD_FLAGS += -race
endif
ifeq ($(gc_debug), 1)
	# -N: 禁止编译优化，使得单步调试更加准确
	# -l: 禁止内联优化，使得函数调用更加准确
	BUILD_FLAGS += -gcflags=all='-N -l'
endif
ifeq ($(profile), 1)
	BUILD_FLAGS += -gcflags=all='-cpuprofile=cpu.prof -memprofile=mem.prof'
endif
ifeq ($(release), 1)
    # -s -w: 通过剥离符号表和调试信息来显著减小最终生成的可执行二进制文件的尺寸
	LDFLAGS := -s -w
	ifeq ($(memory_optimized), 1)
		# -B: 禁止边界检查来提升一些运行时性能，但代价是如果程序中存在越界访问，将不会触发 panic
		BUILD_RELEASE_FLAGS += -gcflags=all='-B'
		LDFLAGS := $(LDFLAGS) -X main.memoryOptimized=true
	endif
	BUILD_RELEASE_FLAGS += '-ldflags=$(LDFLAGS)'
endif

.PHONY: build
build: clean $(ALL_TARGETS) ## Build all executable objects

ifeq ($(release), 1)
$(TARGET): $(SRC)
	@echo "Building release target for $(GOOS)/$(GOARCH)..."
	@if [ -z "$(BUILD_RELEASE_FLAGS)" ]; then \
		GOOS=$(GOOS) GOARCH=$(GOARCH) go build -mod vendor $(PWD)/cmd/$@; \
	else \
		GOOS=$(GOOS) GOARCH=$(GOARCH) go build -mod vendor $(BUILD_RELEASE_FLAGS) $(PWD)/cmd/$@; \
	fi
else
$(TARGET): $(SRC)
	@echo "Building target for $(GOOS)/$(GOARCH)..."
	@if [ -z "$(BUILD_FLAGS)" ]; then \
		GOOS=$(GOOS) GOARCH=$(GOARCH) go build -mod vendor $(PWD)/cmd/$@; \
	else \
		GOOS=$(GOOS) GOARCH=$(GOARCH) go build -mod vendor $(BUILD_FLAGS) $(PWD)/cmd/$@; \
	fi

$(TEST_TARGET): $(SRC)
	@echo "Building test target for $(GOOS)/$(GOARCH)..."
	@if [ -z "$(BUILD_FLAGS)" ]; then \
		GOOS=$(GOOS) GOARCH=$(GOARCH) go build -mod vendor $(PWD)/test/$@; \
	else \
		GOOS=$(GOOS) GOARCH=$(GOARCH) go build -mod vendor $(BUILD_FLAGS) $(PWD)/test/$@; \
	fi
endif

.PHONY: test
test: ## Run tests
	@(GOOS=$(GOOS) GOARCH=$(GOARCH) go test -mod vendor -v ./...)

.PHONY: test-race
test-race: ## Run tests with race detection
	@(GOOS=$(GOOS) GOARCH=$(GOARCH) go test -mod vendor -race -v ./...)

.PHONY: bench
bench: ## Run benchmarks
	@(GOOS=$(GOOS) GOARCH=$(GOARCH) go test -mod vendor -bench=. -benchmem ./...)

.PHONY: local_run
local_run: build ## Run the app locally
	@(./traffic-forwarder)

.PHONY: run
run: ## Start the app using goreman
	# To install goreman, run `go install github.com/mattn/goreman@latest`
	@(echo "proc1: $(PWD)/traffic-forwarder" > Procfile)
	@(goreman check)
	@(goreman start)

.PHONY: stop
stop: ## Stop the app managed by goreman
	@(goreman run stop-all)

.PHONY: status
status: ## Show the status of the app managed by goreman
	@(goreman run status)

.PHONY: clean
clean: ## Clean all executable objects and profile files
	@(rm -f $(ALL_TARGETS))
	@(rm -f Procfile)
	@(rm -f *.prof)

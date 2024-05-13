include .env.local
export

VERSION      := v1.0.0
GIT_HASH     := $(shell git rev-parse --short HEAD)
SRC          := $(shell find . -type f -name '*.go' -not -path "./vendor/*")
TARGETS      := traffic-forwarder
ALL_TARGETS  := $(TARGETS)

.PHONY: help
help: ### Display this help screen.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

ifeq ($(race), 1)
	BUILD_FLAGS := -race
endif

ifeq ($(gc_debug), 1)
	BUILD_FLAGS += -gcflags=all="-N -l"
endif

.PHONY: build
build: clean $(ALL_TARGETS) ## Compile the overall project, and output some executable objects.

$(TARGETS): $(SRC)
	@(GOOS=linux GOARCH=amd64 go build -mod vendor $(BUILD_FLAGS) $(PWD)/cmd/$@)

$(TEST_TARGETS): $(SRC)
	@(GOOS=linux GOARCH=amd64 go build -mod vendor $(BUILD_FLAGS) $(PWD)/test/$@)

.PHONY: clean
clean: ## Clean all executable objects.
	@(rm -f $(ALL_TARGETS))

.PHONY: local_run
local_run:
	@(./traffic-forwarder -f $(PWD)/etc/traffic-forwarder.conf)

.PHONY: run
run:
	@(rm -rf nohup.out)
	@(nohup ./traffic-forwarder -f $(PWD)/etc/traffic-forwarder.conf &)

# Runs INSIDE the dev container (docker compose run --rm dev make <target>).
# Every target iterates over code/*/ so adding a tool needs no change here.
# Pass TOOL=<tool_name> (e.g. TOOL=ibmdocs) to restrict build targets to a single tool.

TOOLS          := $(notdir $(patsubst %/,%,$(wildcard code/*/)))
# Tools that must never link a network stack (enforced by check-no-network).
OFFLINE_TOOLS  := archictl
SELECTED_TOOLS ?= $(if $(TOOL),$(TOOL),$(TOOLS))
MODULE_PREFIX  := github.com/ibm-webmethods-continuous-delivery/7u-cli-tools/code
BUILD_DIR      := target/bin

VERSION        ?= dev
BUILD_DATE     := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT     := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)

GO    := CGO_ENABLED=0 go

.PHONY: build build-all build-linux-amd64 build-linux-arm64 \
        build-darwin-arm64 build-darwin-amd64 build-windows \
        test test-race lint vet tidy check-no-network list clean help

## build: build native binary for current host OS/arch (all tools or TOOL=<name>)
build:
	@mkdir -p $(BUILD_DIR)
	@for t in $(SELECTED_TOOLS); do \
	  echo "== Building $$t (native)"; \
	  (cd code/$$t && $(GO) build -trimpath \
	    -ldflags "-s -w \
	      -X $(MODULE_PREFIX)/$$t/cmd.version=$(VERSION) \
	      -X $(MODULE_PREFIX)/$$t/cmd.buildDate=$(BUILD_DATE) \
	      -X $(MODULE_PREFIX)/$$t/cmd.gitCommit=$(GIT_COMMIT)" \
	    -o ../../$(BUILD_DIR)/$$t ./cmd/$$t) || exit 1; \
	  echo "Built: $(BUILD_DIR)/$$t"; \
	done

## build-linux-amd64: static Linux amd64 binaries
build-linux-amd64:
	@mkdir -p $(BUILD_DIR)
	@for t in $(SELECTED_TOOLS); do \
	  echo "== Building $$t (linux-amd64)"; \
	  (cd code/$$t && GOOS=linux GOARCH=amd64 $(GO) build -trimpath \
	    -ldflags "-s -w \
	      -X $(MODULE_PREFIX)/$$t/cmd.version=$(VERSION) \
	      -X $(MODULE_PREFIX)/$$t/cmd.buildDate=$(BUILD_DATE) \
	      -X $(MODULE_PREFIX)/$$t/cmd.gitCommit=$(GIT_COMMIT)" \
	    -o ../../$(BUILD_DIR)/$$t-linux-amd64 ./cmd/$$t) || exit 1; \
	  echo "Built: $(BUILD_DIR)/$$t-linux-amd64"; \
	done

## build-linux-arm64: static Linux arm64 binaries
build-linux-arm64:
	@mkdir -p $(BUILD_DIR)
	@for t in $(SELECTED_TOOLS); do \
	  echo "== Building $$t (linux-arm64)"; \
	  (cd code/$$t && GOOS=linux GOARCH=arm64 $(GO) build -trimpath \
	    -ldflags "-s -w \
	      -X $(MODULE_PREFIX)/$$t/cmd.version=$(VERSION) \
	      -X $(MODULE_PREFIX)/$$t/cmd.buildDate=$(BUILD_DATE) \
	      -X $(MODULE_PREFIX)/$$t/cmd.gitCommit=$(GIT_COMMIT)" \
	    -o ../../$(BUILD_DIR)/$$t-linux-arm64 ./cmd/$$t) || exit 1; \
	  echo "Built: $(BUILD_DIR)/$$t-linux-arm64"; \
	done

## build-darwin-arm64: macOS Apple Silicon binaries
build-darwin-arm64:
	@mkdir -p $(BUILD_DIR)
	@for t in $(SELECTED_TOOLS); do \
	  echo "== Building $$t (darwin-arm64)"; \
	  (cd code/$$t && GOOS=darwin GOARCH=arm64 $(GO) build -trimpath \
	    -ldflags "-s -w \
	      -X $(MODULE_PREFIX)/$$t/cmd.version=$(VERSION) \
	      -X $(MODULE_PREFIX)/$$t/cmd.buildDate=$(BUILD_DATE) \
	      -X $(MODULE_PREFIX)/$$t/cmd.gitCommit=$(GIT_COMMIT)" \
	    -o ../../$(BUILD_DIR)/$$t-darwin-arm64 ./cmd/$$t) || exit 1; \
	  echo "Built: $(BUILD_DIR)/$$t-darwin-arm64"; \
	done

## build-darwin-amd64: macOS Intel binaries
build-darwin-amd64:
	@mkdir -p $(BUILD_DIR)
	@for t in $(SELECTED_TOOLS); do \
	  echo "== Building $$t (darwin-amd64)"; \
	  (cd code/$$t && GOOS=darwin GOARCH=amd64 $(GO) build -trimpath \
	    -ldflags "-s -w \
	      -X $(MODULE_PREFIX)/$$t/cmd.version=$(VERSION) \
	      -X $(MODULE_PREFIX)/$$t/cmd.buildDate=$(BUILD_DATE) \
	      -X $(MODULE_PREFIX)/$$t/cmd.gitCommit=$(GIT_COMMIT)" \
	    -o ../../$(BUILD_DIR)/$$t-darwin-amd64 ./cmd/$$t) || exit 1; \
	  echo "Built: $(BUILD_DIR)/$$t-darwin-amd64"; \
	done

## build-windows: static Windows amd64 binaries
build-windows:
	@mkdir -p $(BUILD_DIR)
	@for t in $(SELECTED_TOOLS); do \
	  echo "== Building $$t (windows-amd64)"; \
	  (cd code/$$t && GOOS=windows GOARCH=amd64 $(GO) build -trimpath \
	    -ldflags "-s -w \
	      -X $(MODULE_PREFIX)/$$t/cmd.version=$(VERSION) \
	      -X $(MODULE_PREFIX)/$$t/cmd.buildDate=$(BUILD_DATE) \
	      -X $(MODULE_PREFIX)/$$t/cmd.gitCommit=$(GIT_COMMIT)" \
	    -o ../../$(BUILD_DIR)/$$t-windows-amd64.exe ./cmd/$$t) || exit 1; \
	  echo "Built: $(BUILD_DIR)/$$t-windows-amd64.exe"; \
	done

## build-all: build all five cross-platform targets + checksums (all tools or TOOL=<name>)
build-all: build-linux-amd64 build-linux-arm64 build-darwin-arm64 build-darwin-amd64 build-windows
	@echo ""
	@echo "All built binaries in $(BUILD_DIR):"
	@ls -lh $(BUILD_DIR)
	@cd $(BUILD_DIR) && sha256sum *-linux-* *-darwin-* *.exe > checksums.txt 2>/dev/null || true
	@if [ -f $(BUILD_DIR)/checksums.txt ]; then echo "Checksums written to $(BUILD_DIR)/checksums.txt"; fi

## test: go test for every tool
test:
	@for t in $(TOOLS); do echo "== $$t"; (cd code/$$t && $(GO) test ./...) || exit 1; done

## test-race: go test -race for every tool (needs cgo, so it overrides CGO_ENABLED)
test-race:
	@for t in $(TOOLS); do echo "== $$t"; (cd code/$$t && CGO_ENABLED=1 go test -race ./...) || exit 1; done

## vet: go vet for every tool
vet:
	@for t in $(TOOLS); do echo "== $$t"; (cd code/$$t && $(GO) vet ./...) || exit 1; done

## lint: golangci-lint for every tool
lint:
	@for t in $(TOOLS); do echo "== $$t"; (cd code/$$t && golangci-lint run ./...) || exit 1; done

## check-no-network: fail if an offline tool (OFFLINE_TOOLS) depends on net/http, crypto/tls or net/rpc
check-no-network:
	@for t in $(OFFLINE_TOOLS); do \
	  echo "== $$t"; \
	  if (cd code/$$t && $(GO) list -deps ./cmd/$$t | grep -E '^(net/http|crypto/tls|net/rpc)$$'); then \
	    echo "ERROR: $$t links a network package"; exit 1; \
	  fi; \
	done; echo "check-no-network: OK"

## tidy: go mod tidy for every tool
tidy:
	@for t in $(TOOLS); do echo "== $$t"; (cd code/$$t && go mod tidy) || exit 1; done

## list: print discovered tools
list:
	@echo $(TOOLS)

## clean: remove build artefacts
clean:
	rm -rf target/

## help: list available targets
help:
	@grep -E '^## ' Makefile | sed 's/## /  /'

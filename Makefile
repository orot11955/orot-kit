APP := kit
VERSION ?= 0.1.0-dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GO ?= go
GOCACHE ?= /tmp/orot-kit-gocache
GOMODCACHE ?= /tmp/orot-kit-gomodcache
GOENV := GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE)
BIN_DIR ?= bin
DIST_DIR ?= dist
ASSETS_DIR ?= assets
SERVE_ADDR ?= :8080
SERVE_BASE_URL ?= http://localhost:8080
RUNTIME_CACHE_DIR ?= $(HOME)/.kit-server/cache/runtimes
STATS_FILE ?= $(HOME)/.kit-server/download-stats.json
SERVE_STATE_DIR ?= .kit-server
SERVE_PID ?= $(SERVE_STATE_DIR)/serve.pid
SERVE_LOG ?= $(SERVE_STATE_DIR)/serve.log
LDFLAGS := -X github.com/orot-dev/orot-kit/pkg/version.Version=$(VERSION) -X github.com/orot-dev/orot-kit/pkg/version.Commit=$(COMMIT) -X github.com/orot-dev/orot-kit/pkg/version.BuildDate=$(BUILD_DATE)

.PHONY: build build-current build-darwin build-linux build-all dist dist-darwin-amd64 dist-darwin-arm64 dist-linux-amd64 dist-linux-arm64 test fmt vet check clean serve serve-stop serve-status serve-log serve-site serve-site-dry-run

build: build-current

build-current:
	mkdir -p $(BIN_DIR)
	$(GOENV) $(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP) .

build-darwin: dist-darwin-amd64 dist-darwin-arm64

build-linux: dist-linux-amd64 dist-linux-arm64

build-all: build-darwin build-linux

dist: build-all

dist-darwin-amd64:
	mkdir -p $(DIST_DIR)
	GOOS=darwin GOARCH=amd64 $(GOENV) $(GO) build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(APP)-darwin-amd64 .

dist-darwin-arm64:
	mkdir -p $(DIST_DIR)
	GOOS=darwin GOARCH=arm64 $(GOENV) $(GO) build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(APP)-darwin-arm64 .

dist-linux-amd64:
	mkdir -p $(DIST_DIR)
	GOOS=linux GOARCH=amd64 $(GOENV) $(GO) build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(APP)-linux-amd64 .

dist-linux-arm64:
	mkdir -p $(DIST_DIR)
	GOOS=linux GOARCH=arm64 $(GOENV) $(GO) build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(APP)-linux-arm64 .

test:
	$(GOENV) $(GO) test ./...

fmt:
	gofmt -w .

vet:
	$(GOENV) $(GO) vet ./...

check: fmt vet test

serve: build-current dist
	@mkdir -p "$(SERVE_STATE_DIR)"
	@if [ -f "$(SERVE_PID)" ] && kill -0 "$$(cat "$(SERVE_PID)")" 2>/dev/null; then \
		echo "kit docs server already running at $(SERVE_BASE_URL) (pid $$(cat "$(SERVE_PID)"))"; \
	else \
		if command -v setsid >/dev/null 2>&1; then \
			setsid -f sh -c 'echo $$$$ > "$$1"; exec "$$2" install-server --addr "$$3" --bin-dir "$$4" --runtime-cache-dir "$$5" --assets-dir "$$6" --stats-file "$$7" --base-url "$$8"' sh "$(SERVE_PID)" "$(BIN_DIR)/$(APP)" "$(SERVE_ADDR)" "$(DIST_DIR)" "$(RUNTIME_CACHE_DIR)" "$(ASSETS_DIR)" "$(STATS_FILE)" "$(SERVE_BASE_URL)" > "$(SERVE_LOG)" 2>&1; \
			pid=$$(cat "$(SERVE_PID)"); \
		else \
			nohup "$(BIN_DIR)/$(APP)" install-server --addr "$(SERVE_ADDR)" --bin-dir "$(DIST_DIR)" --runtime-cache-dir "$(RUNTIME_CACHE_DIR)" --assets-dir "$(ASSETS_DIR)" --stats-file "$(STATS_FILE)" --base-url "$(SERVE_BASE_URL)" > "$(SERVE_LOG)" 2>&1 & \
			pid=$$!; \
			echo $$pid > "$(SERVE_PID)"; \
		fi; \
		sleep 1; \
		if kill -0 "$$pid" 2>/dev/null; then \
			echo "kit docs server running at $(SERVE_BASE_URL) (pid $$pid)"; \
			echo "log: $(SERVE_LOG)"; \
		else \
			echo "kit docs server failed to start. See $(SERVE_LOG)"; \
			rm -f "$(SERVE_PID)"; \
			exit 1; \
		fi; \
	fi

serve-stop:
	@if [ -f "$(SERVE_PID)" ] && kill -0 "$$(cat "$(SERVE_PID)")" 2>/dev/null; then \
		kill "$$(cat "$(SERVE_PID)")"; \
		rm -f "$(SERVE_PID)"; \
		echo "kit docs server stopped"; \
	else \
		rm -f "$(SERVE_PID)"; \
		echo "kit docs server is not running"; \
	fi

serve-status:
	@if [ -f "$(SERVE_PID)" ] && kill -0 "$$(cat "$(SERVE_PID)")" 2>/dev/null; then \
		echo "kit docs server running at $(SERVE_BASE_URL) (pid $$(cat "$(SERVE_PID)"))"; \
	else \
		echo "kit docs server is not running"; \
	fi

serve-log:
	@if [ -f "$(SERVE_LOG)" ]; then \
		tail -n 80 "$(SERVE_LOG)"; \
	else \
		echo "no serve log at $(SERVE_LOG)"; \
	fi

serve-site: dist
	$(GOENV) $(GO) run . install-server --addr "$(SERVE_ADDR)" --bin-dir "$(DIST_DIR)" --runtime-cache-dir "$(RUNTIME_CACHE_DIR)" --assets-dir "$(ASSETS_DIR)" --stats-file "$(STATS_FILE)" --base-url "$(SERVE_BASE_URL)"

serve-site-dry-run:
	$(GOENV) $(GO) run . --dry-run install-server --addr "$(SERVE_ADDR)" --bin-dir "$(DIST_DIR)" --runtime-cache-dir "$(RUNTIME_CACHE_DIR)" --assets-dir "$(ASSETS_DIR)" --stats-file "$(STATS_FILE)" --base-url "$(SERVE_BASE_URL)"

clean:
	rm -rf $(BIN_DIR) $(DIST_DIR)

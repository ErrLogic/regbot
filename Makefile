BINARY := regbot
SERVER_BIN := regbot-server
WORKER_BIN := regbot-worker
PKG := ./cmd/regbot
SERVER_PKG := ./cmd/regbot-server
WORKER_PKG := ./cmd/regbot-worker

# Config file used by the server/worker/migrate targets. Override with:
#   make run-server CONFIG=config.yaml
CONFIG ?= config.local.yaml

# JWT secret for the API server. Override with: make run-server JWT_SECRET=...
JWT_SECRET ?= dev-secret-change-me

# Android SDK path Appium needs. Override with: make run-appium ANDROID_HOME=/path
ANDROID_HOME ?= /usr/lib/android-sdk

.PHONY: build build-server build-worker build-all test itest lint vet tidy clean
.PHONY: migrate-up migrate-down dev dev-web run-server run-worker run-appium infra-up infra-down web-install

# --- Build ---

build: ## Build the regbot CLI binary
	go build -o $(BINARY) $(PKG)

build-server: ## Build the API server binary
	go build -o $(SERVER_BIN) $(SERVER_PKG)

build-worker: ## Build the background worker binary
	go build -o $(WORKER_BIN) $(WORKER_PKG)

build-all: build build-server build-worker ## Build all binaries

# --- Test ---

test: ## Run unit tests
	go test -count=1 ./...

itest: ## Run integration tests (requires a real device + Appium)
	go test -count=1 -tags=integration ./...

# --- Lint & Vet ---

vet: ## Run go vet
	go vet ./...

lint: ## Run golangci-lint
	golangci-lint run

tidy: ## Tidy module dependencies
	go mod tidy

# --- Database ---

migrate-up: ## Apply all pending migrations
	go run $(SERVER_PKG) -config $(CONFIG) -migrate up

migrate-down: ## Roll back migrations
	go run $(SERVER_PKG) -config $(CONFIG) -migrate down

# --- Infrastructure ---

infra-up: ## Start PostgreSQL and Redis (Docker)
	docker compose up -d

infra-down: ## Stop PostgreSQL and Redis
	docker compose down

# --- Run services ---

run-server: ## Start the API server (needs infra + migrations)
	REGBOT_SERVER_JWT_SECRET=$(JWT_SECRET) go run $(SERVER_PKG) -config $(CONFIG)

run-worker: ## Start the background job worker (needs infra + a device + Appium)
	go run $(WORKER_PKG) -config $(CONFIG)

run-appium: ## Start the Appium server (needs a connected device to drive)
	ANDROID_HOME=$(ANDROID_HOME) ANDROID_SDK_ROOT=$(ANDROID_HOME) appium --log-level info

web-install: ## Install the web UI dependencies
	cd web && npm install

dev-web: ## Start the Svelte dev server (proxies /api -> :8080)
	cd web && npm run dev

# --- Development helper ---

dev: infra-up migrate-up ## Bring up infra + migrate, then print next steps
	@echo ""
	@echo "Infra is up and migrations applied. Now open three terminals:"
	@echo "  1) make run-server     # API on http://localhost:8080"
	@echo "  2) make run-worker     # background jobs (needs a device + Appium)"
	@echo "  3) make dev-web        # UI on http://localhost:5173"

# --- Clean ---

clean: ## Remove build artifacts
	rm -f $(BINARY) $(SERVER_BIN) $(WORKER_BIN)
	rm -rf artifacts

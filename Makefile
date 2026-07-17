BINARY := regbot
PKG := ./cmd/regbot

.PHONY: build test itest lint vet tidy clean

build: ## Build the regbot binary
	go build -o $(BINARY) $(PKG)

test: ## Run unit tests
	go test ./...

itest: ## Run integration tests (requires a real device + Appium)
	go test -tags=integration ./...

vet: ## Run go vet
	go vet ./...

lint: ## Run golangci-lint
	golangci-lint run

tidy: ## Tidy module dependencies
	go mod tidy

clean: ## Remove build artifacts
	rm -f $(BINARY)
	rm -rf artifacts

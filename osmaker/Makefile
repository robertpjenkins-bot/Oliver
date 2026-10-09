# OS Maker 2.0 - design an OS, build it as an Arch Linux ISO.

BIN     := osmaker
DIST    := dist
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

# Static, so the one AppImage runs on any distribution.
GOBUILD := CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)"

.PHONY: all build run test check appimage clean help
all: build

build: ## Build the binary for this machine
	$(GOBUILD) -o $(DIST)/$(BIN) .
	@echo "built $(DIST)/$(BIN) ($(VERSION))"

run: build ## Build and open the designer
	./$(DIST)/$(BIN)

test: ## Run the test suite
	go test ./...

check: ## Format check, vet and test
	@test -z "$$(gofmt -l . | tee /dev/stderr)" || (echo "run: gofmt -w ." && false)
	go vet ./...
	go test ./...

appimage: ## Build the AppImage
	./packaging/build-appimage.sh

clean: ## Remove build output
	rm -rf $(DIST)

help: ## List the available targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2}'

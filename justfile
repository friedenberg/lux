# Lux: LSP Multiplexer

default:
    @just --list

# Build the binary
build:
    nix build

# Build using go directly (faster for dev)
build-go:
    nix develop --command go build -o lux ./cmd/lux

# Run tests
test:
    nix develop --command go test ./...

# Run tests with verbose output
test-v:
    nix develop --command go test -v ./...

# Format code
fmt:
    nix develop --command go fmt ./...
    shfmt -w -i 2 -ci ./*.sh 2>/dev/null || true

# Lint code
lint:
    go vet ./...

# Run the LSP server
serve:
    go run ./cmd/lux serve

# Show configured LSPs
list:
    go run ./cmd/lux list

# Check LSP status
status:
    go run ./cmd/lux status

# Add a new LSP from a flake
add flake:
    go run ./cmd/lux add "{{flake}}"

# Run MCP server over stdio
mcp-stdio:
    go run ./cmd/lux mcp stdio

# Run MCP server over SSE
mcp-sse addr=":8080":
    go run ./cmd/lux mcp sse --addr "{{addr}}"

# Run MCP server over HTTP
mcp-http addr=":8081":
    go run ./cmd/lux mcp http --addr "{{addr}}"

# Run in nix develop shell
dev:
    nix develop

# Install MCP server to Claude Code config
install:
    nix run .#install-mcp

# Clean build artifacts
clean:
    rm -f lux
    rm -rf result

# Update go dependencies and regenerate gomod2nix.toml
deps:
    nix develop --command go mod tidy
    nix develop --command gomod2nix

# Regenerate gomod2nix.toml (run after changing go.mod)
gomod2nix:
    nix develop --command gomod2nix

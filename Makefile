.PHONY: build build-cgo build-all run-agent test capture-events fmt

# Default local build: poll-only process monitoring (no Endpoint Security / CGO).
build:
	mkdir -p bin
	CGO_ENABLED=0 go build -buildvcs=false -o bin/trustedge-agent ./cmd/trustedge-agent

# Optional: Endpoint Security watcher (needs macOS SDK + signing/entitlements).
build-cgo:
	mkdir -p bin
	CGO_ENABLED=1 go build -buildvcs=false -o bin/trustedge-agent ./cmd/trustedge-agent

build-all:
	mkdir -p bin
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -buildvcs=false -o bin/trustedge-agent-darwin-arm64 ./cmd/trustedge-agent
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -buildvcs=false -o bin/trustedge-agent-darwin-amd64 ./cmd/trustedge-agent
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -buildvcs=false -o bin/trustedge-agent-linux-amd64 ./cmd/trustedge-agent
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build -buildvcs=false -o bin/trustedge-agent-windows-amd64.exe ./cmd/trustedge-agent

agent:
	CGO_ENABLED=0 go run ./cmd/trustedge-agent

run-agent:
	CGO_ENABLED=0 TRUSTEDGE_AGENT_DETAILS_INTERVAL=30 TRUSTEDGE_AGENT_NETWORK_INTERVAL=30 TRUSTEDGE_AGENT_ACTION_INTERVAL=30 \
		go run ./cmd/trustedge-agent

test:
	CGO_ENABLED=0 go test ./...

# Default :18080 avoids clashing with Docker / TrustEdge stack on :8080.
capture-events:
	TRUSTEDGE_CAPTURE_ADDR=:18080 go run ./scripts/capture-events

fmt:
	go fmt ./...

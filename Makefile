.PHONY: build build-all run-agent test fmt

build:
	mkdir -p bin
	go build -buildvcs=false -o bin/trustedge-agent ./cmd/trustedge-agent

build-all:
	mkdir -p bin
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -buildvcs=false -o bin/trustedge-agent-darwin-arm64 ./cmd/trustedge-agent
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -buildvcs=false -o bin/trustedge-agent-darwin-amd64 ./cmd/trustedge-agent
	GOOS=linux GOARCH=amd64 go build -buildvcs=false -o bin/trustedge-agent-linux-amd64 ./cmd/trustedge-agent
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build -buildvcs=false -o bin/trustedge-agent-windows-amd64.exe ./cmd/trustedge-agent

agent:
	go run ./cmd/trustedge-agent

run-agent:
	TRUSTEDGE_AGENT_DETAILS_INTERVAL=30 TRUSTEDGE_AGENT_NETWORK_INTERVAL=30 TRUSTEDGE_AGENT_ACTION_INTERVAL=30 \
		go run ./cmd/trustedge-agent

test:
	go test ./...

capture-events:
	go run ./scripts/capture-events

fmt:
	go fmt ./...

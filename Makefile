.PHONY: build build-all api agent run-api run-agent test fmt docker-image

build:
	mkdir -p bin
	go build -buildvcs=false -o bin/trustedge-agent-api ./cmd/trustedge-agent-api
	go build -buildvcs=false -o bin/trustedge-agent ./cmd/trustedge-agent

build-all:
	mkdir -p bin
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -buildvcs=false -o bin/trustedge-agent-darwin-arm64 ./cmd/trustedge-agent
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -buildvcs=false -o bin/trustedge-agent-darwin-amd64 ./cmd/trustedge-agent
	GOOS=linux GOARCH=amd64 go build -buildvcs=false -o bin/trustedge-agent-linux-amd64 ./cmd/trustedge-agent
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build -buildvcs=false -o bin/trustedge-agent-windows-amd64.exe ./cmd/trustedge-agent

api:
	go run ./cmd/trustedge-agent-api

agent:
	go run ./cmd/trustedge-agent

run-api: api

run-agent:
	TRUSTEDGE_AGENT_DETAILS_INTERVAL=30 TRUSTEDGE_AGENT_NETWORK_INTERVAL=30 TRUSTEDGE_AGENT_ACTION_INTERVAL=30 \
		go run ./cmd/trustedge-agent

test:
	go test ./...

fmt:
	go fmt ./...

docker-image:
	docker build -t trustedge-agent-api .

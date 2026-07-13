.PHONY: build build-all api agent run-api run-agent test fmt docker-image

build:
	mkdir -p bin
	go build -buildvcs=false -o bin/trusttwin-api ./cmd/trusttwin-api
	go build -buildvcs=false -o bin/trusttwin ./cmd/trusttwin

build-all:
	mkdir -p bin
	GOOS=darwin GOARCH=arm64 go build -buildvcs=false -o bin/trusttwin-darwin-arm64 ./cmd/trusttwin
	GOOS=darwin GOARCH=amd64 go build -buildvcs=false -o bin/trusttwin-darwin-amd64 ./cmd/trusttwin
	GOOS=linux GOARCH=amd64 go build -buildvcs=false -o bin/trusttwin-linux-amd64 ./cmd/trusttwin
	GOOS=windows GOARCH=amd64 go build -buildvcs=false -o bin/trusttwin-windows-amd64.exe ./cmd/trusttwin

api:
	go run ./cmd/trusttwin-api

agent:
	go run ./cmd/trusttwin

run-api: api

run-agent:
	TRUSTTWIN_DETAILS_INTERVAL=30 TRUSTTWIN_NETWORK_INTERVAL=30 TRUSTTWIN_ACTION_INTERVAL=30 \
		go run ./cmd/trusttwin

test:
	go test ./...

fmt:
	go fmt ./...

docker-image:
	docker build -t trusttwin-api .

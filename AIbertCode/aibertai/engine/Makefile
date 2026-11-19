.PHONY: build check linux

build:
	go build -trimpath -o bin/aibertai-engine ./cmd/aibertai-engine

check:
	go test ./...
	go vet ./...
	python3 -m unittest discover -s scripts -p 'test_*.py'

linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o bin/aibertai-engine-linux-amd64 ./cmd/aibertai-engine
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -o bin/aibertai-engine-linux-arm64 ./cmd/aibertai-engine

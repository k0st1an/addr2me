IMAGE   := addr2me
PORT    := 7007
BIN     := .bin
NETWORK ?=

.PHONY: build build-linux build-darwin test docker-build docker-run

build:
	mkdir -p $(BIN)
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BIN)/$(IMAGE) .

build-linux:
	mkdir -p $(BIN)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BIN)/$(IMAGE)-linux-amd64 .

build-darwin:
	mkdir -p $(BIN)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o $(BIN)/$(IMAGE)-darwin-arm64 .

test:
	go test -v ./...

docker-build:
	docker build -t $(IMAGE) .

docker-run:
	docker run --rm $(if $(NETWORK),--network $(NETWORK)) -p $(PORT):7007 $(if $(IPINFO_TOKEN),-e IPINFO_TOKEN=$(IPINFO_TOKEN)) $(IMAGE)


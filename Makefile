.PHONY: build

build:
	CGO_ENABLED=0 go build -o addr2me cmd/addr2me/main.go

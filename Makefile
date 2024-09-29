.PHONY: build docker-build docker-run

build:
	CGO_ENABLED=0 go build -o addr2me cmd/addr2me/main.go

docker-build:
	docker build -t k0st1an/addr2me .

docker-run:
	docker run --name addr2me --rm -p 7007:7007/tcp k0st1an/addr2me

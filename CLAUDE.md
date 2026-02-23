# addr2me

Lightweight HTTP service that returns the client's public IP address.

## Stack

- Go (standard library only, no external dependencies)
- HTML templates via `html/template` + `embed.FS`

## Project structure

```
main.go                 — all server logic
main_test.go            — unit tests
templates/index.html    — HTML page
nginx.conf.example      — nginx reverse proxy config example
Dockerfile              — multi-stage build (scratch + ca-certificates)
Makefile                — build, test, docker targets
```

## Endpoints

| Path    | Method | Response                        |
|---------|--------|---------------------------------|
| `/`     | GET    | plain text or HTML page (content negotiation) |
| `/json` | GET    | `{"ip":"...","time_utc":"..."}` |

All non-GET requests return `405 Method Not Allowed`.

`/` uses content negotiation: returns HTML if `Accept: text/html`, otherwise plain text.
Plain text format without geo: `IP,time_utc`
Plain text format with geo: `IP,time_utc,country,country_code,continent,continent_code,asn,as_name`

## Commands

```bash
go run .                  # run server on :7007
go run . -port 8080       # run server on custom port
go test ./...             # run tests

make build                # build binary to .bin/
make build-linux          # cross-compile for linux/amd64
make build-darwin         # cross-compile for darwin/arm64
make test                 # run tests
make docker-build         # build docker image
make docker-run           # run container (PORT, NETWORK, IPINFO_TOKEN)
```

## Configuration

Geo/ASN enrichment is optional. Set `IPINFO_TOKEN` environment variable to enable it.
Without a token the service works normally; geo data is simply not shown.

## IP detection order

1. `X-Forwarded-For` header
2. `X-Real-IP` header
3. `RemoteAddr`

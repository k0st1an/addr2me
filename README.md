# addr2me

A lightweight HTTP service that returns the client's public IP address.

## Endpoints

### `GET /`

Returns the client's IP address. Response format depends on the `Accept` header:

- **Browser** (`Accept: text/html`) — HTML page with IP, request time, and optional geo/ASN info
- **Curl / API** — plain text, comma-separated

```
$ curl https://yourhost/
1.2.3.4,2026-02-23T10:15:30Z

$ curl https://yourhost/   # with token
1.2.3.4,2026-02-23T10:15:30Z,United States,US,North America,NA,AS15169,Google LLC
```

---

### `GET /json`

Returns a JSON object with the client's IP address and request time in UTC.
Geo/ASN fields are included only when an ipinfo.io token is configured.

```
$ curl https://yourhost/json
{"ip":"1.2.3.4","time_utc":"2026-02-23T10:15:30Z"}

$ curl https://yourhost/json  # with token
{"ip":"1.2.3.4","time_utc":"2026-02-23T10:15:30Z","country":"United States","country_code":"US","continent":"North America","continent_code":"NA","asn":"AS15169","as_name":"Google LLC"}
```

## Running

```bash
go run .                # listen on :7007
go run . -port 8080     # custom port
```

The server listens on port `:7007` by default.

## Docker

```bash
make docker-build

make docker-run                              # default
make docker-run NETWORK=host                 # host network mode
make docker-run IPINFO_TOKEN=your_token      # with geo enrichment
```

The image is built on `scratch` with CA certificates bundled for TLS support.

## Configuration

Geo/ASN enrichment via [ipinfo.io](https://ipinfo.io) is optional.
When a token is configured, the HTML page shows country, continent, and ASN info.

```bash
IPINFO_TOKEN=your_token go run .
```

Without a token the service works normally — geo info is simply not shown.

## IP Detection

The client IP is resolved in the following order:

1. `X-Forwarded-For` — proxy / load balancer
2. `X-Real-IP` — nginx
3. `RemoteAddr` — direct connection

# Proxy API

> [!WARNING]
> You do not need to read or understand the code in this folder to do the
> take-home. Everything you need to know about the API is in this README.

`make check` builds the API image for you.

`GET /proxy` returns a random proxy from Postgres. Each request takes from
200 ms to 3 s. One API process answers at most 20 requests at the same time,
and answers `503` to any more.

## Ports and endpoints

| Port | Endpoint | Answers |
|---|---|---|
| `8080` | `GET /proxy` | `200` with a proxy as JSON |
| `9090` | `GET /ping` | `200` while the process runs |
| `9090` | `GET /db` | `200` after the API has connected to Postgres |
| `9090` | `GET /status` | `200` while the API takes new requests, `503` after it gets `SIGTERM` |
| `9090` | `GET /metrics` | Prometheus metrics |

When the API gets `SIGTERM`, it stops taking new connections on port `8080`.
It finishes the requests in flight for up to `shutdownTimeout`, then exits.

## Metrics

Only `GET /proxy` is measured.

| Metric | Type | Labels |
|---|---|---|
| `http_requests_in_flight` | gauge | - |
| `http_requests_total` | counter | `route`, `code` |
| `http_request_duration_seconds` | histogram | `route` |

## Configuration

The API reads it from `/root/configs/config.yaml` and exits if the file is missing. This is the config for local development:

```yaml
api:
  addr: ""
  port: 8080
  healthCheckPort: 9090
  logLevel: info
  shutdownTimeout: 10s

postgres:
  host: localhost
  port: 5432
  user: ping_dev
  database: ping_residential
  sslMode: disable
  passwordFile: secrets/postgres-password
```

The config holds no password. The API reads the password from the file at
`passwordFile`. The path is relative to `/root`, unless it starts with `/`.

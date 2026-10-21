# Netcore

Netcore is a Go REST API for network and subscriber management. It uses Gin for
HTTP routing, PostgreSQL for persistence, and GORM for database access.

## Features

- User registration and login with email or phone number
- JWT access tokens and rotating refresh tokens
- Password reset with OTP
- MikroTik device management and PPPoE operations
- GenieACS management and device operations
- WhatsApp gateway integration for supported workflows

## Requirements

- Go 1.25.1 or later
- PostgreSQL
- Network access to any MikroTik, GenieACS, or WhatsApp gateway services used
  by the deployment

## Configuration

Create a `.env` file in the project root, or provide the same values through the
process environment:

```env
APP_HOST=127.0.0.1
APP_PORT=8080

DB_HOST=127.0.0.1
DB_PORT=5432
DB_USER=netcore
DB_PASSWORD=replace-with-a-database-password
DB_NAME=netcore

JWT_SECRET=replace-with-a-random-secret-of-at-least-32-bytes

WHATSAPP_GATEWAY_URL=
WHATSAPP_GATEWAY_USERNAME=
WHATSAPP_GATEWAY_PASSWORD=
```

`APP_HOST` defaults to `127.0.0.1`, `APP_PORT` defaults to `8080`, and
`DB_PORT` defaults to `5432`. `JWT_SECRET` must be at least 32 bytes; the
application exits during startup if it is too short.

The PostgreSQL database must exist before starting the application. On startup,
GORM runs `AutoMigrate` for the application models.

Do not commit `.env` or put production credentials in source control. The
current PostgreSQL connection configuration uses `sslmode=disable`; use a
trusted private network for the database connection, or enable and verify
database TLS in the application before connecting over an untrusted network.

## Run

Install dependencies and start the API:

```sh
go mod download
go run ./cmd/api
```

The API listens on the address configured by `APP_HOST` and `APP_PORT`. For a
local Cloudflare Tunnel or reverse proxy on the same host, point its origin to
`http://127.0.0.1:8080` (or the configured address). If the proxy runs in a
separate container, configure `APP_HOST` to an address reachable on a private
network and restrict access to that network.

## API

All endpoints use the `/api` prefix. Protected endpoints require an access token
in the `Authorization: Bearer <access_token>` header.

| Method | Path | Access |
| --- | --- | --- |
| `POST` | `/api/register` | Public |
| `POST` | `/api/login` | Public |
| `POST` | `/api/refresh` | Public |
| `POST` | `/api/forgot-password` | Public |
| `POST` | `/api/verify-otp` | Public |
| `POST` | `/api/reset-password` | Public |
| `GET` | `/api/me` | Authenticated |
| `POST` | `/api/logout` | Authenticated |
| `GET`, `POST` | `/api/mikrotiks` | Authenticated |
| `GET`, `PUT`, `DELETE` | `/api/mikrotiks/:id` | Authenticated |
| `GET`, `POST` | `/api/pppoe/secrets` | Authenticated |
| `GET`, `PUT`, `DELETE` | `/api/pppoe/secrets/:id` | Authenticated |
| `GET`, `POST` | `/api/pppoe/profiles` | Authenticated |
| `GET`, `PUT`, `DELETE` | `/api/pppoe/profiles/:name` | Authenticated |
| `GET` | `/api/pppoe/active` | Authenticated |
| `DELETE` | `/api/pppoe/active/:id` | Authenticated |
| `GET` | `/api/pppoe/pools` | Authenticated |
| `GET` | `/api/pppoe/offline` | Authenticated |
| `GET`, `POST` | `/api/genieacs` | Authenticated |
| `GET`, `PUT`, `DELETE` | `/api/genieacs/:id` | Authenticated |
| `GET` | `/api/devices/:id` | Authenticated |
| `POST` | `/api/devices/:id/:deviceId/summon` | Authenticated |

The refresh endpoint returns a replacement refresh token. Clients must store
the new token because the previous one is invalidated.

## Tests and build

Run the test suite:

```sh
go test ./...
```

Build all packages:

```sh
go build ./...
```

## Project layout

```text
cmd/api/          Application entry point
internal/config/  Database configuration
internal/controllers/
internal/middleware/
internal/models/
internal/routes/  HTTP route registration
internal/services/
internal/utils/
tests/            Package-level tests
```

## License

No license has been specified for this project.

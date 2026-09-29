# Go Inventory Management

A work-in-progress inventory backend for managing products, locations, and stock movements. It uses Go 1.23, PostgreSQL, the standard `net/http` package, server-rendered HTML forms, and SQL migrations.

## Implemented

- Create products with validated package amounts and units; reject duplicate product identities.
- Create locations and activate or deactivate them. A location with positive stock cannot be deactivated.
- Receive and sell stock at locations, recording stock movements and enforcing stock availability.
- Apply database constraints and migrations for products, locations, stock levels, and movement history.

The transfer use case and PostgreSQL persistence are implemented internally, but are not yet connected to an HTTP route or browser form.

## Structure

```text
cmd/
  server/       HTTP server and route wiring
  migrate/      database migration command
internal/
  domain/       product, location, and stock rules
  application/  use cases and repository ports
  adapter/
    http/       handlers and embedded HTML forms
    postgres/   PostgreSQL repositories
  config/       TOML and environment configuration
  migrations/   migration runner
migrations/     PostgreSQL up/down SQL
```

## Local setup

Prerequisites: Go 1.23 or later and PostgreSQL.

The commands below assume the project root is the working directory. The application reads `config.toml`; environment variables override its values. The supplied development config targets a local database named `inventory` and user `inventory`.

Create the local role and database if they do not already exist (adapt authentication to your PostgreSQL setup):

```sh
psql -U postgres -c "CREATE ROLE inventory LOGIN PASSWORD 'replace-this-locally';"
psql -U postgres -c "CREATE DATABASE inventory OWNER inventory;"
export INVENTORY_DB_PASSWORD='replace-this-locally'
```

Do not commit real credentials or secrets. `INVENTORY_DB_PASSWORD` is read from the environment; the other database settings can also be overridden with `INVENTORY_DB_HOST`, `INVENTORY_DB_PORT`, `INVENTORY_DB_NAME`, `INVENTORY_DB_USER`, and `INVENTORY_DB_SSL_MODE`. Server settings use `INVENTORY_SERVER_ADDRESS` and `INVENTORY_LOG_LEVEL`.

Run the migrations, start the server, and verify the code with:

```sh
go run ./cmd/migrate
go run ./cmd/server
go test ./...
go build ./...
```

The server listens on `:8080` by default. The test suite uses unit tests, fakes, and SQL mocks; `go test ./...` does not require a live PostgreSQL database.

## Browser routes

| Method | Path | Behavior |
|---|---|---|
| GET | `/products/new` | Product creation form |
| POST | `/products` | Create a product |
| GET | `/locations/new` | Location creation form |
| POST | `/locations` | Create a location |
| POST | `/locations/{id}/activate` | Activate a location |
| POST | `/locations/{id}/deactivate` | Deactivate a location when it has no positive stock |
| GET, POST | `/stock/receive` | Receive stock and record a movement |
| GET, POST | `/stock/sell` | Sell stock and record a movement |
| Any method | `/health` | Return `200` and `ok` |

## Status and next work

Core product, location, receiving, and selling flows are implemented and tested. The browser interface does not yet provide product or location listings, stock lookup, or a stock-transfer workflow; exposing these workflows is a logical next step.

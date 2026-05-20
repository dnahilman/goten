# goten/adapters/gorm

GORM adapter for goten — implements the `goten.Adapter` interface using [GORM](https://gorm.io).

## Installation

```bash
go get github.com/dnahilman/goten/adapters/gorm
go get gorm.io/driver/postgres  # or mysql / sqlite
```

## Usage

```go
import (
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    gormadapter "github.com/dnahilman/goten/adapters/gorm"
)

db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
if err != nil {
    log.Fatal(err)
}

adapter := gormadapter.New(db)
auth, err := goten.New(goten.Config{
    Adapter: adapter,
    // ...
})
```

## Integration Tests

Requires Docker (uses testcontainers automatically):

```bash
cd test
go test -tags integration -v ./adapters/gorm/...
```

Or point to an existing Postgres instance:

```bash
GOTEN_TEST_DSN="postgres://goten:goten@localhost:5432/goten?sslmode=disable" \
  go test -tags integration -v ./adapters/gorm/...
```

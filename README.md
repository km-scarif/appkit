# AppKit

AppKit is a go module that contains modules and functions to use in go services or apps

## Dependencies

`github.com/jmoiron/sqlx` for the database functions, which is a lightweight extension on top of `database/sql` 
and is a superset on the existing db interfaces, so they can be used interchangably.  Functional enhancements include:

- Marshall rows into structs, maps, and slices
- Named parameter support

Also, the database access uses the following for MySQL and ODBC connections:

- `github.com/alexbrainman/odbc`
- `github.com/go-sql-driver/mysql`

The logger uses the standard `log/slog` package for json logging, but for colored text logging (mainly in dev) it uses 
the Charm Logger `github.com/charmbracelet/log`

## Components

### Envconfig

#### Usage

The struct should be in the following format:

```go
type Config struct {
	Foo           string        `env:"FOO" default:"5500"`
	Bar           time.Duration `env:"BAR" default:"24h"`
	Baz           int           `env:"BAZ" default:"0"`
	Fiz           bool          `env:"FIZ" default:"false"`
	Buz           float64       `env:"BUZ" default:"3.14"   log:"secret"`
}
```

The struct can then be loaded as follows:

```go

func InitConfig(appname string) Config {
	var cfg Config
    	// Use the envconfig module to load from environment
	if err := envconfig.LoadFromEnv(&cfg); err != nil {
		// Handle error - you might want to log this or panic depending on your needs
		panic("Failed to load configuration: " + err.Error())
	}
    ...

```

### Logger

#### Usage

Returns a Charm stdout logger or a slog JSON logger

Takes a struct in the following format:

```go
type LoggerConfig struct {
	Level       string     
	Prefix      string     
	LoggerType  string 
}
```

```go
    logger := appkit.InitLogger(appkit.LogConfig{
        Level:  cfg.LogLevel,
        Prefix: "my-app",
        LoggerType: "charm"
    })
```

Log Levels:  (case insensitive, defaults to ERROR)

- DEBUG
- INFO
- WARN
- ERROR

There is also a function `func LogConfig(cfg any, logger Logger) {...} ` that you can use to log an envconfig to stdout.  
The logger will mask any env vars with the tag `log:"secret"`

Example output:

```bash
INFO my-app: Port:                5500
INFO my-app: LogLevel:            debug
INFO my-app: LogHttp:             on
INFO my-app: LoggerType:          charm
INFO my-app: ApiKey:              
INFO my-app: MySQLServer:         127.0.0.1
INFO my-app: MySQLPort:           3306
INFO my-app: MySQLUser:           dbuser
INFO my-app: MySQLPwd:            ********
```

### Errors

There are error functions that wrap errors or message strings in a JSON response wrapper for convenience

```go
type errorResponse struct {
    Error string `json:"error"`
}

func ErrorResponseErr(err error) errorResponse {
    return errorResponse{Error: err.Error()}
}

func ErrorResponseMsg(s string) errorResponse {
    return errorResponse{Error: s}
}
```

#### Usage

```go
func FooHandler(c echo.Context) error {
    if err != nil {
        app.Logger.Errorf("FooHandler: %v", err)
        return c.JSON(http.StatusInternalServerError, appkit.ErrorResponseMsg("Error retrieving foo"))
        // or
        return c.JSON(http.StatusInternalServerError, appkit.ErrorResponseErr(err))
    }
    return c.JSON(http.StatusOK, results)
}
```


### Database Connections

There are functions for MySQL and ODBC connections.  These functions use the following external libraries:

- `github.com/jmoiron/sqlx` 
- `github.com/alexbrainman/odbc`
- `github.com/go-sql-driver/mysql`

Also, there are helper types for DB2 database access that integrate with `sqlx` for easy reading and writing to the
database — see [DB2 Column Types](#db2-column-types) below.

#### Usage

`base-repo.go` has generic functions to interact with a db.  

```go
type Repository[T any] struct {
    db              *sqlx.DB
    scannerFunc     func(*sqlx.Rows) (*T, error)
    argsFunc        func(*T) []any
}

func NewRepository[T any](db *sqlx.DB, scanner func(*sqlx.Rows) (*T, error), args func(*T) []any) *Repository[T] {
    return &Repository[T]{
        db:      db,
        scannerFunc: scanner,
        argsFunc:    args,
    }
}

// Query with custom SQL and parameters
func (r *Repository[T]) FindAll(ctx context.Context, query string, args ...any) ([]T, error) {...}
func (r *Repository[T]) FindOne(ctx context.Context, query string, args ...any) (*T, error) {...}
func (r *Repository[T]) Execute(ctx context.Context, query string, args ...any) (sql.Result, error) {...}
func (r *Repository[T]) ExecuteNamed(ctx context.Context, query string, arg any) (sql.Result, error) {...}

```

These are based on having structs defined for the database tables, and are used in the following manner:

```go

type Customer struct {
    Id              int         `json:"id"              db:"id"`
    Name            string      `json:"name"            db:"name"`
    AccountNumber   string      `json:"accountNumber"   db:"account_number"`
}

func GetCustomers(dbconn *sqlx.DB) ([]models.Customer, error) {

    // this struct does not use a scanner func...
    repo := appkit.NewRepository[models.Customer]( dbconn, nil, nil)
    var customers []models.Customer
    query := "SELECT id, name, account_number FROM customers WHERE is_active = 1"

    ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancelfunc()

    customers, err := repo.FindAll(ctx, query)
    if err != nil {
        return customers, fmt.Errorf("GetCustomers: %w", err)
    }
    return customers, nil
}
```

If you want to scan records the non-sqlx way, you can attach a scanner func to your struct, and pass that in 
to the repositoryf:

```go
type Customer struct {
    Id              int         `json:"id"              db:"id"`
    Name            string      `json:"name"            db:"name"`
    AccountNumber   string      `json:"accountNumber"   db:"account_number"`
}
func ScanCustomer(rows *sql.Rows) (*Customer, error) {
    var c Customer
    err := rows.Scan(
        &c.Id,
        &c.Name,
        &c.AccountNumber,
    )
    if err != nil {
        return nil, err
    }
    return &c, nil
}

func GetCustomers(dbconn *sql.DB) ([]models.Customer, error) {
    repo := appkit.NewRepository( dbconn, models.ScanCustomer, nil)
    var customers []models.Customer
    query := "SELECT id, name, account_number FROM customers WHERE is_active = 1"
    // ...
}
```

In using this method, obviously the order of the fields in the scanner func will need to match the order
of the rows in the sql...

For simple queries, you can also inline the scanner func:

```go

func SumWidgets(dbconn *sqlx.DB, sku string) (int, error) {

    query := "SELECT COALESCE(SUM(quantity),0) FROM widgets WHERE sku = ?"

    // just make the scanner func inline since it is so simple...
    sumRepo := appkit.NewRepository(dbconn, func(rows *sqlx.Rows) (*int, error) {
        var sum int
        err := rows.Scan(&sum)
        return &sum, err
    }, nil)

    ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancelfunc()
    // pass in the args to the query...
    result, err := sumRepo.FindOne(ctx, query, sku )

    if err != nil {
        return 0, fmt.Errorf("SumWidgets: %w", err)
    }

    return *result, nil 
}
```

### DB2 Column Types

Reading from DB2 on the IBM i is awkward in two ways, and these types exist to absorb both so the rest of a service
never has to think about it.

**1. The data doesn't match its intent.** `CHAR` columns come back space-padded to their declared width. Values that
are conceptually identifiers — invoice numbers, customer numbers — are stored as `DECIMAL(x,0)` rather than `INTEGER`,
so they arrive as floats and would serialize as JSON numbers unless something intervenes. Dates are frequently stored
as a `YYYYMMDD` number.

**2. The Go type varies by column.** The ODBC driver picks the concrete Go type per column, and the full set it can
produce (see `BaseColumn.Value` in `github.com/alexbrainman/odbc`'s `column.go`) is:

| ODBC C type | Go value | Typical DB2 column |
|---|---|---|
| `SQL_C_BIT` | `bool` | — |
| `SQL_C_LONG` | **`int32`** | `INTEGER`, and `CASE ... THEN 1 ELSE 0` |
| `SQL_C_SBIGINT` | `int64` | `BIGINT` |
| `SQL_C_DOUBLE` | `float64` | `DECIMAL` / `NUMERIC`, including `DECIMAL(x,0)` |
| `SQL_C_CHAR` | `[]byte` | `CHAR` / `VARCHAR` |
| `SQL_C_WCHAR` | `string` | `GRAPHIC` / unicode char |
| `SQL_C_TYPE_TIMESTAMP` | `time.Time` | `TIMESTAMP` |

`int32` is the one that catches people out — it is not one of `database/sql`'s `driver.Value` types, but this driver
returns it anyway. **Every type below accepts the whole set that makes sense for it**, so the same field type works
whether a column is `INTEGER` in one table and `DECIMAL(5,0)` in another. Pick a type by the JSON you want, not by the
DB2 declaration.

| Type | Use for | JSON output |
|---|---|---|
| `DB2TrimmedString` | `CHAR` / `VARCHAR`, or any column you want as text | string, padding trimmed |
| `DB2AnyString` | alias for `DB2TrimmedString`; a self-documenting name for columns whose DB2 type is unknown or inconsistent | string |
| `DB2FloatAsString` | numeric columns that are really identifiers (invoice / customer numbers) | **string** |
| `DB2TrimmedFloat64` | numeric columns that really are numbers (prices, weights) | number |
| `DB2TrimmedInt64` | integer columns and counts (quantities); a decimal source truncates toward zero | number |
| `DB2Bool` | flag columns: numeric `0`/`1`, or `Y`/`N`, `T`/`F`, `TRUE`/`FALSE` | `true` / `false` |
| `DB2Date` | dates stored as `YYYYMMDD` in any column type, or a real `TIMESTAMP` | string, `YYYY-MM-DD` |

A `NULL` column scans to the zero value on every one of them — empty string, `0`, `false` — never an error.

#### Usage

Declare them as the field types in the model struct; `sqlx` calls `Scan` through the `db:` tag and `encoding/json`
uses the `json:` tag on the way out.

```go
type InvoiceLine struct {
    InvoiceNumber               appkit.DB2FloatAsString      `json:"invoiceNumber"        db:"INVOICE_NUMBER"`
    InvoiceDate                 appkit.DB2Date               `json:"invoiceDate"          db:"INVOICE_DATE"`
    CustomerName                appkit.DB2TrimmedString      `json:"customerName"         db:"CUSTOMER_NAME"`
    ProductNumber               appkit.DB2TrimmedString      `json:"productNumber"        db:"PRODUCT_NUMBER"`
    Quantity                    appkit.DB2TrimmedInt64       `json:"quantity"             db:"QTY"`
    Price                       appkit.DB2TrimmedFloat64     `json:"price"                db:"PRICE"`
    IsActive                    appkit.DB2Bool               `json:"isActive"             db:"IS_ACTIVE"`
}
```

Note `DB2Bool` rather than a plain Go `bool` for the flag. A `bool` field goes through the standard library's
`driver.Bool`, which handles `int64` and `string` but **not `int32` or `float64`** — precisely what DB2 returns for a
`CASE ... THEN 1 ELSE 0`, so it fails at scan time with `unsupported type int32`.

#### Precision limit

`DECIMAL` arrives as a `float64`, which represents integers exactly only up to 2^53 (about 9.0e15, 16 digits). A
`DECIMAL(17,0)` or wider used as an identifier has already lost precision by the time Go sees it, and no amount of
formatting recovers it. Scan those as `DB2TrimmedString`, which keeps the driver's own text.

`DB2TrimmedInt64` is safe here: `int64` and whole-number text sources are taken directly rather than being routed
through `float64`, so a `BIGINT` past 2^53 keeps every digit.

#### v0.2.1 behavior notes

- `int32` support was added across all the types. This is strictly a widening — a column that scanned correctly before
  produces the identical value, and only inputs that previously *errored* now succeed.
- `DB2Bool` and the `DB2AnyString` alias are new.
- `DB2FloatAsString` now marshals with full precision instead of rounding to zero decimals. Integral values (which is
  everything in a `DECIMAL(x,0)` column) render byte-identically; a fractional value now renders as `"3.5"` where it
  used to be silently rounded to `"4"`.

### Probe Server

`NewProbeServer` builds a minimal, standard-library `*http.Server` for services
that don't expose a full API — workers, schedulers, FTP jobs, etc. — but still
need liveness/readiness endpoints for k8s, load balancers, or uptime monitoring.
It avoids pulling Echo into a service that only needs a healthcheck.

For services that need real routing, binding, or middleware, keep using Echo;
this helper is deliberately minimal.

It exposes three routes:

- `GET /`        — plain HTML banner naming the service
- `GET /up`      — liveness; always `200 "up"` while the process is serving
- `GET /health`  — readiness; runs every configured check and returns `503` if
  any fails, otherwise `200`, with a per-dependency breakdown in the body

The returned server is configured but **not** started — the caller owns its
lifecycle (`ListenAndServe` / `Shutdown`) so it plugs into any graceful-shutdown
pattern.

`DBHealthCheck` adapts a `*sqlx.DB` into a check by pinging it (a nil handle is
treated as unhealthy).

#### Usage

```go
import "github.com/km-scarif/appkit"

server := appkit.NewProbeServer(appkit.ProbeServerConfig{
    Port:        cfg.Port,
    ServiceName: "km-ftpjobs",
    LogHTTP:     cfg.LogHttp == "on",
    Checks: map[string]appkit.HealthCheck{
        "mysql": appkit.DBHealthCheck(mysqlDB),
        "db2":   appkit.DBHealthCheck(db2DB),
    },
}, logger)

go func() {
    if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        logger.Errorf("http server error: %s", err)
    }
}()

// ...on shutdown:
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
_ = server.Shutdown(ctx)
```

A successful `GET /health` returns:

```json
{
    "status": "healthy",
    "service": "km-ftpjobs",
    "timestamp": "2026-06-02T12:00:00Z",
    "required_services": { "mysql": "healthy", "db2": "healthy" }
}
```

### Basic AppKit implementation

One way to use AppKit in an application would be to use the following structure:

```code

├── cmd
│   └── main.go
├── internal
│   ├── app
│   │   ├── app.go

```

Example of the above:

```go
// cmd/main.go

package main

import (
	//...
	"github.com/km-scarif/appkit"
	"myapp/internal/app"
)

func main() {
	var cfg app.AppConfig
	appkit.LoadFromEnv(&cfg)

	logger := appkit.InitLogger(appkit.LoggerConfig{
		Level:      cfg.LogLevel,
		Prefix:     "myapp",
		LoggerType: cfg.LoggerType,
	})

	appkit.LogConfig(cfg, logger)
	logger.Infof("Starting myapp server on port %s...", fmt.Sprintf("%d", cfg.Port))

	// start a new server instance
	server := server.NewEchoServer(cfg.Port, cfg.LogHttp, cfg.ApiKey)

	mysqlDB, mysqlErr := appkit.ConnectToMySQL(appkit.MySQLConfig{
		DSN:    cfg.MySQLDsn,
		Server: cfg.MySQLServer,
		Port:   cfg.MySQLPort,
		User:   cfg.MySQLUser,
		Pwd:    cfg.MySQLPwd,
		DB:     cfg.MySQLDb,
	}, logger)

	if mysqlErr != nil {
		logger.Errorf("Failed to initialize database [%s]: %v", cfg.MySQLServer, mysqlErr)
	} else {
		logger.Infof("Database [%s] initialized successfully", cfg.MySQLServer)
	}

	// set the logger and db connections into app...
	app.Init(logger, mysqlDB, cfg)

    //...
}

```

Setup in the app package:

```go
// internal/app/app.go

package app

import (
    "errors"
    "github.com/km-scarif/appkit"
    "github.com/jmoiron/sqlx"
)

var (
    Logger      appkit.Logger
    MySQLDB     *sqlx.DB
    Config      AppConfig
)

func Init(logger appkit.Logger, mysqlDB *sqlx.DB, cfg AppConfig) {
    Logger  = logger
    MySQLDB = mysqlDB
    Config  = cfg
}

// app config struct...
type AppConfig struct {
	Port                int         `env:"PORT" default:"5500"`
	LogLevel            string      `env:"LOG_LEVEL" default:"info"`
    LogHttp             string      `env:"LOG_HTTP" default:"off"`
    LoggerType          string      `env:"LOGGER_TYPE" default:"charm"`
    ApiKey              string      `env:"API_KEY" default:""`

    MySQLServer         string      `env:"MYSQL_SERVER" default:""`
    MySQLPort           int         `env:"MYSQL_PORT" default:"3306"`
    MySQLUser           string      `env:"MYSQL_USER" default:""`
    MySQLPwd            string      `env:"MYSQL_PWD" default:"" log:"secret"`
    MySQLDb             string      `env:"MYSQL_DB" default:""`
}

// defined errors for application
var ErrInvalidRequestFormat = errors.New("Invalid request format")
var ErrBarNotFound = errors.New("Bar not found")
var ErrBarRetrieval = errors.New("Error retreiving Bar")

```



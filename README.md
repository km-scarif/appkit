# appkit

appkit is a go module that contains modules and functions to use in go services or apps

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
INFO wms-ecom-returns: Port:                5500
INFO wms-ecom-returns: LogLevel:            debug
INFO wms-ecom-returns: LogHttp:             on
INFO wms-ecom-returns: LoggerType:          charm
INFO wms-ecom-returns: ApiKey:              
INFO wms-ecom-returns: MySQLServer:         127.0.0.1
INFO wms-ecom-returns: MySQLPort:           3306
INFO wms-ecom-returns: MySQLUser:           admin
INFO wms-ecom-returns: MySQLPwd:            ********
```


### Database Connections

There are functions for MySQL and ODBC connections.  These functions use the following external libraries:

- `github.com/jmoiron/sqlx` 
- `github.com/alexbrainman/odbc`
- `github.com/go-sql-driver/mysql`

Also, there are helper functions for DB2 database access that integrate with `sqlx` for easy reading and writing to the database

- `DB2TrimmedString`      for DB2 CHAR fields
- `DB2FloatAsString`      for DB2 fields to be used as strings that are in DB2 as DECIMALS 
- `DB2TrimmedFloat64`     for DB2 fields to be used as floats (may be INTEGERS, DECIMALS, or CHAR) 
- `DB2TrimmedInt64`       for DB2 fields to be used as integers (may be INTEGERS, DECIMALS, or CHAR)
- `DB2DATE`               for DB2 fields that contain dates stored as DECIMAL or INTEGER

## Usage




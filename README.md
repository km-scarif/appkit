# appkit

appkit is a go module that contains modules and functions to use in go services or apps

## envconfig

### Usage

The struct should be in the following format:

```go
type Config struct {
	Foo           string        `env:"FOO" default:"5500"`
	Bar           time.Duration `env:"BAR" default:"24h"`
	Baz           int           `env:"BAZ" default:"0"`
	Fiz           bool          `env:"FIZ" default:"false"`
	Buz           float64       `env:"BUZ" default:"3.14"`
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

## logger

### Usage

Returns a Charm stdout logger

Takes a struct in the following format:

```go

type LoggerConfig struct {
    Level   string
    Prefix  string
}
```

```go
    logger := appkit.InitLogger(appkit.LogConfig{
        Level:  cfg.LogLevel,
        Prefix: "my-app",
    })
```

Log Levels:  (case insensitive, defaults to ERROR)

- DEBUG
- INFO
- WARN
- ERROR

## db connections

There are functions for MySQL and ODBC connections

## Usage



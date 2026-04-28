package odbc

import (
	"context"
	"fmt"
	"time"
    "errors"

	_ "github.com/alexbrainman/odbc"
    "github.com/jmoiron/sqlx"
)

// Logger is the minimal logging interface this package needs.
// It is structurally compatible with appkit.Logger, so callers can pass
// an appkit.Logger directly without any conversion.
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	Debugf(format string, args ...any)
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
	Errorf(format string, args ...any)
}

var ErrConnectionInfoIncomplete = errors.New("odbc connection info incomplete")


type OdbcConfig struct {
	DSN    string
	Driver string
	System string
	Uid    string
	Pwd    string
}

// DB switcher here
func ConnectToODBC(cfg OdbcConfig, logger Logger) (*sqlx.DB, error) {
	logger.Debug("Connecting to DB2 ODBC Server...")

	var db *sqlx.DB

	if cfg.DSN != "" || cfg.Driver != "" {

		// use the DSN if set in the env, otherwise build the dsn string...
		var dsn string
		if cfg.DSN != "" {
			dsn = "DSN=" + cfg.DSN
		} else {
			// format...
			// "DRIVER={driver};SYSTEM=system;UID=uid;PWD=pwd"
			dsn = fmt.Sprintf("DRIVER={%s};SYSTEM=%s;UID=%s;PWD=%s",
				cfg.Driver, cfg.System, cfg.Uid, cfg.Pwd)
		}

		logger.Debug(dsn)
        var err error
		db, err = sqlx.Open("odbc", dsn)
		if err != nil {
			return db, fmt.Errorf("failed to open ODBC DB on [%s]: %w", cfg.System, err)
		}

		// set connection pool params
		db.SetMaxOpenConns(10)
		db.SetMaxIdleConns(10)
		db.SetConnMaxLifetime(time.Minute * 1)

		ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelfunc()

		// ping the connection
		err = db.PingContext(ctx)
		if err != nil {
			return db, fmt.Errorf("failed to ping [%s]: %w", cfg.System, err)
		}

		logger.Infof("Connected to ODBC server [%s] successfully...", cfg.System)

	} else {
        return db, ErrConnectionInfoIncomplete
	}

	return db, nil
}

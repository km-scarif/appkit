package appkit

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/alexbrainman/odbc"
)

type OdbcConfig struct {
	DSN    string
	Driver string
	System string
	Uid    string
	Pwd    string
}

// DB switcher here
func ConnectToODBC(cfg OdbcConfig, logger Logger) (*sql.DB, error) {
	logger.Debug("Connecting to DB2 ODBC Server...")

	var db *sql.DB

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
		db, err := sql.Open("odbc", dsn)
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

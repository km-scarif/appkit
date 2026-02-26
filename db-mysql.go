package appkit

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"

)

type MySQLConfig struct {
    DSN         string
    Server      string
    Port        int
    User        string
    Pwd         string
    DB          string
}


// Initializes a MySQL database connection
func ConnectToMySQL(cfg MySQLConfig, logger Logger) (*sql.DB, error) {
    logger.Debug("Connecting to MySQL Server...")

    // return var...
    var db *sql.DB

    // if the dsn isn't configured, or the server var isn't set, return an error...
    if cfg.DSN != "" || cfg.Server != "" {

        // use the DSN if set in the env, otherwise build the dsn string...
        var dsn string
        if cfg.DSN != "" {
            dsn = cfg.DSN
        } else {
            // format... 
            // user:pwd@tcp(server:port)/db?parseTime=true&loc=UTC
            dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&loc=UTC",
                  cfg.User, cfg.Pwd, cfg.Server, cfg.Port, cfg.DB)
        }
        
        logger.Debug(dsn)
        // Open database connection
        db, err := sql.Open("mysql", dsn)
        // MysqlDB, err = sql.Open("mysql", cfg.FormatDSN())
        if err != nil {
            return db, fmt.Errorf("failed to open MySQL DB on [%s]: %w", cfg.Server, err)
        }

        // Configure connection pool settings
        // Maximum number of open connections to the database
        db.SetMaxOpenConns(25)
        
        // Maximum number of idle connections in the pool
        db.SetMaxIdleConns(10)
        
        // Maximum lifetime of a connection (helps with load balancers and connection recycling)
        db.SetConnMaxLifetime(5 * time.Minute)
        
        // Maximum idle time for a connection
        db.SetConnMaxIdleTime(1 * time.Minute)

        // Verify the connection is working
        if err = db.Ping(); err != nil {
            return db, fmt.Errorf("failed to ping [%s]: %w", cfg.Server, err)
        }

       logger.Infof("Connected to MySQL server [%s] successfully...", cfg.Server)

    } else {
        return db, ErrConnectionInfoIncomplete
    }

    return db, nil

}



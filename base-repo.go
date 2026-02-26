package appkit

import (
	"context"
	"database/sql"
	"errors"
)

var ErrNilConnection = errors.New("Database connection is unavailable")

type Repository[T any] struct {
    db              *sql.DB
    scannerFunc     func(*sql.Rows) (*T, error)
    argsFunc        func(*T) []any
}

func NewRepository[T any](db *sql.DB, scanner func(*sql.Rows) (*T, error), args func(*T) []any) *Repository[T] {
    return &Repository[T]{
        db:      db,
        scannerFunc: scanner,
        argsFunc:    args,
    }
}

// Query with custom SQL and parameters
func (r *Repository[T]) FindAll(ctx context.Context, query string, args ...any) ([]T, error) {

    // check for the db connection
    if r.db == nil {
        return nil, ErrNilConnection
    }

    stmt, err := r.db.PrepareContext(ctx, query)
    if err != nil {
        return nil, err
    }
    defer stmt.Close()

    rows, err := stmt.QueryContext(ctx, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var results []T
    for rows.Next() {
        entity, err := r.scannerFunc(rows)
        if err != nil {
            return nil, err
        }
        results = append(results, *entity)
    }
    
    return results, rows.Err()
}

// QueryOne for single row queries
func (r *Repository[T]) FindOne(ctx context.Context, query string, args ...any) (*T, error) {

    // check for the db connection
    if r.db == nil {
        return nil, ErrNilConnection
    }

    stmt, err := r.db.PrepareContext(ctx, query)
    if err != nil {
        return nil, err
    }
    defer stmt.Close()

    rows, err := stmt.QueryContext(ctx, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    if rows.Next() {
        return r.scannerFunc(rows)
    }
    
    return nil, sql.ErrNoRows
}

// Execute for INSERT, UPDATE, DELETE
func (r *Repository[T]) Execute(ctx context.Context, query string, args ...any) (sql.Result, error) {

    // check for the db connection
    if r.db == nil {
        return nil, ErrNilConnection
    }

    stmt, err := r.db.PrepareContext(ctx, query)
    if err != nil {
        return nil, err
    }
    defer stmt.Close()

    return stmt.ExecContext(ctx, args...)
}

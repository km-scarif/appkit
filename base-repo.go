package appkit

import (
	"context"
	"database/sql"
	"errors"

    "github.com/jmoiron/sqlx"
)

var ErrNilConnection = errors.New("Database connection is unavailable")

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
func (r *Repository[T]) FindAll(ctx context.Context, query string, args ...any) ([]T, error) {

    // check for the db connection
    if r.db == nil {
        return nil, ErrNilConnection
    }

    // if there's a scanner func, use it (handles transformation/nullable types)
    if r.scannerFunc != nil {
        rows, err := r.db.QueryxContext(ctx, query, args...)
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

    // if there isn't a scanner func, scan directly in...
    var results []T
    err := r.db.SelectContext(ctx, &results, query, args...)
    
    return results, err
}


// QueryOne for single row queries
func (r *Repository[T]) FindOne(ctx context.Context, query string, args ...any) (*T, error) {

    // check for the db connection
    if r.db == nil {
        return nil, ErrNilConnection
    }

    var result T

    err := r.db.GetContext(ctx, &result, query, args...)
    
    return &result, err
}


// Execute for INSERT, UPDATE, DELETE (positional parameters)
func (r *Repository[T]) Execute(ctx context.Context, query string, args ...any) (sql.Result, error) {
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


// ExecuteNamed for INSERT, UPDATE, DELETE (named parameters)
func (r *Repository[T]) ExecuteNamed(ctx context.Context, query string, arg any) (sql.Result, error) {
    if r.db == nil {
        return nil, ErrNilConnection
    }

    stmt, err := r.db.PrepareNamedContext(ctx, query)
    if err != nil {
        return nil, err
    }
    defer stmt.Close()

    return stmt.ExecContext(ctx, arg)
}

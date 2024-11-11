package safesql

import (
	"context"
	"database/sql"

	safeauth "github.com/stssk/def-prog-exercises/safeauth"
	"github.com/stssk/def-prog-exercises/safesql/internal/raw"
)

type compileTimeConstant string

type TrustedSQL struct {
	s string
}

func New(text compileTimeConstant) TrustedSQL {
	return TrustedSQL{string(text)}
}

type DB struct {
	db *sql.DB
}

func (db *DB) QueryContext(ctx context.Context, query TrustedSQL, args ...interface{}) (*Rows, error) {
	err := safeauth.Must(ctx)
	if err != nil {
		return nil, err
	}
	return db.db.QueryContext(ctx, query.s, args...)
}

type Rows = sql.Rows

type Result = sql.Result

func (db *DB) ExecContext(ctx context.Context, query TrustedSQL, args ...interface{}) (Result, error) {
	return db.db.ExecContext(ctx, query.s, args...)
}

func Open(driverName, dataSourceName string) (*DB, error) {
	d, err := sql.Open(driverName, dataSourceName)
	return &DB{d}, err
}

func init() {
	raw.TrustedSQLCtor = func(unsafe string) TrustedSQL {
		return TrustedSQL{unsafe}
	}
}

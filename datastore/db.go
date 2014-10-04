package datastore

import (
	"database/sql"
	"sync"

	"code.google.com/p/go.net/context"

	"github.com/calhacks/calhacks/config"
	_ "github.com/lib/pq"
)

var DB *sql.DB

var connectOnce sync.Once

func Connect() {
	connectOnce.Do(func() {
		var err error
		DB, err = sql.Open("postgres", config.DatabaseURL())
		if err != nil {
			panic(err)
		}
	})
}

func Disconnect() {
	DB.Close()
}

type key int

const txKey key = 0

func NewContextWithTx(ctx context.Context) (context.Context, error) {
	tx, err := DB.Begin()
	if err != nil {
		return nil, err
	}

	return context.WithValue(ctx, txKey, tx), nil
}

func TxFromContext(ctx context.Context) (*sql.Tx, bool) {
	tx, ok := ctx.Value(txKey).(*sql.Tx)
	return tx, ok
}

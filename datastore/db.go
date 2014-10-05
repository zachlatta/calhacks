package datastore

import (
	"database/sql"
	"errors"
	"net/http"
	"sync"

	"code.google.com/p/go.net/context"

	"github.com/calhacks/calhacks/config"
	"github.com/calhacks/calhacks/httputil"
	"github.com/calhacks/calhacks/model"
	"github.com/dgrijalva/jwt-go"
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

const (
	txKey   key = 0
	userKey key = 1
)

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

func NewContextWithUser(ctx context.Context,
	r *http.Request) (context.Context, error) {
	tok, err := jwt.ParseFromRequest(r, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.JWTSecret()), nil
	})
	if err != nil {
		return nil, err
	}

	id := int64(tok.Claims["id"].(float64))
	user, err := GetUser(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, &httputil.HTTPError{http.StatusNotFound,
				errors.New("user from token not found")}
		}
		return nil, err
	}

	return context.WithValue(ctx, userKey, user), nil
}

func UserFromContext(ctx context.Context,
	r *http.Request) (*model.User, bool) {
	user, ok := ctx.Value(userKey).(*model.User)
	return user, ok
}

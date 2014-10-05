package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"

	"github.com/gorilla/mux"
	"github.com/zachlatta/calhacks/datastore"
	"github.com/zachlatta/calhacks/httputil"
	"github.com/zachlatta/calhacks/router"

	"code.google.com/p/go.net/context"
)

func Handler() *mux.Router {
	m := router.API()
	m.Get(router.SubmitChallenge).Handler(bufHandler(submitChallenge))
	// TODO: m.Get(router.Challenge).Handler(bufHandler(getPost))
	// TODO: m.Get(router.CurrentChallenge).Handler(bufHandler(currentChallenge))
	m.Get(router.WebsocketConnect).Handler(handler(wsConnect))

	m.Get(router.OauthLogin).Handler(bufHandler(oauthLogin))
	m.Get(router.OauthAccessToken).Handler(bufHandler(oauthAccessToken))

	return m
}

func prepareContext(r *http.Request) (context.Context, context.CancelFunc,
	error) {
	ctx, cancel := context.WithCancel(context.Background())
	ctx, err := datastore.NewContextWithTx(ctx)
	if err != nil {
		return nil, nil, err
	}
	if r.Header.Get("Authorization") != "" ||
		r.URL.Query().Get("access_token") != "" {
		ctx, err = datastore.NewContextWithUser(ctx, r)
		if err != nil {
			return nil, nil, err
		}
	}

	return ctx, cancel, nil
}

func addCORSHeaders(w http.ResponseWriter) {
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Methods",
		"POST, PUT, DELETE, GET, OPTIONS")
	w.Header().Add("Access-Control-Request-Method", "*")
	w.Header().Add("Access-Control-Allow-Headers",
		"Origin, Content-Type, Authorization")
}

type handler func(context.Context, http.ResponseWriter, *http.Request)

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	addCORSHeaders(w)
	if r.Method == "OPTIONS" {
		return
	}

	ctx, cancel, err := prepareContext(r)
	if err != nil {
		handleAPIError(w, r, http.StatusInternalServerError, err, false)
	}
	defer cancel()
	tx, _ := datastore.TxFromContext(ctx)
	defer tx.Commit()

	h(ctx, w, r)
}

// bufHandler is a buffered request handler that simplifies returning errors.
// It's great for normal HTTP requests, but won't work for things like
// websockets.
type bufHandler func(context.Context, http.ResponseWriter, *http.Request) error

func (h bufHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	addCORSHeaders(w)
	if r.Method == "OPTIONS" {
		return
	}

	defer func() {
		if rv := recover(); rv != nil {
			err := errors.New("handler panic")
			logError(r, err, rv)
			handleAPIError(w, r, http.StatusInternalServerError, err, false)
		}
	}()

	var (
		rb  httputil.ResponseBuffer
		err error
	)
	ctx, cancel, err := prepareContext(r)
	if err != nil {
		handleAPIError(w, r, http.StatusInternalServerError, err, false)
	}
	defer cancel()

	tx, _ := datastore.TxFromContext(ctx)
	err = h(ctx, &rb, r)
	if err == nil {
		rb.WriteTo(w)
		tx.Commit()
	} else if e, ok := err.(*httputil.HTTPError); ok {
		if e.Status >= 500 {
			logError(r, err, nil)
		}
		handleAPIError(w, r, e.Status, e.Err, true)
		tx.Rollback()
	} else {
		logError(r, err, nil)
		handleAPIError(w, r, http.StatusInternalServerError, err, false)
		tx.Rollback()
	}
}

func logError(req *http.Request, err error, rv interface{}) {
	if err != nil {
		var buf bytes.Buffer
		fmt.Fprintf(&buf, "Error serving %s: %v\n", req.URL, err)
		if rv != nil {
			fmt.Fprintln(&buf, rv)
			buf.Write(debug.Stack())
		}
		log.Println(buf.String())
	}
}

func handleAPIError(resp http.ResponseWriter, req *http.Request,
	status int, err error, showErrorMsg bool) {
	var data struct {
		Error struct {
			Status  int    `json:"status"`
			Message string `json:"message"`
		} `json:"error"`
	}
	data.Error.Status = status
	if showErrorMsg {
		data.Error.Message = err.Error()
	} else {
		data.Error.Message = http.StatusText(status)
	}
	resp.Header().Set("Content-Type", "application/json; charset=utf-8")
	resp.WriteHeader(status)
	json.NewEncoder(resp).Encode(&data)
}

package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"

	"github.com/calhacks/calhacks/datastore"
	"github.com/calhacks/calhacks/httputil"

	"code.google.com/p/go.net/context"
)

func ctxWithCancelAndTx() (context.Context, context.CancelFunc, error) {
	ctx, cancel := context.WithCancel(context.Background())
	ctx, err := datastore.NewContextWithTx(ctx)
	if err != nil {
		return nil, nil, err
	}
	return ctx, cancel, nil
}

type Handler func(context.Context, http.ResponseWriter, *http.Request)

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel, err := ctxWithCancelAndTx()
	if err != nil {
		handleAPIError(w, r, http.StatusInternalServerError, err, false)
	}
	defer cancel()

	h(ctx, w, r)
}

// BufHandler is a buffered request handler that simplifies returning errors.
// It's great for normal HTTP requests, but won't work for things like
// websockets.
type BufHandler func(context.Context, http.ResponseWriter, *http.Request) error

func (h BufHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
	ctx, cancel, err := ctxWithCancelAndTx()
	if err != nil {
		handleAPIError(w, r, http.StatusInternalServerError, err, false)
	}
	defer cancel()

	err = h(ctx, &rb, r)
	if err == nil {
		rb.WriteTo(w)
	} else if e, ok := err.(*httputil.HTTPError); ok {
		if e.Status >= 500 {
			logError(r, err, nil)
		}
		handleAPIError(w, r, e.Status, e.Err, true)
	} else {
		logError(r, err, nil)
		handleAPIError(w, r, http.StatusInternalServerError, err, false)
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

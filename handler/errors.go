package handler

import (
	"errors"
	"net/http"

	"github.com/zachlatta/calhacks/httputil"
)

func validationError(message string) *httputil.HTTPError {
	return &httputil.HTTPError{httputil.StatusUnprocessableEntity,
		errors.New(message)}
}

func badRequest(err error) *httputil.HTTPError {
	return &httputil.HTTPError{http.StatusBadRequest, err}
}

func unauthorized() *httputil.HTTPError {
	return &httputil.HTTPError{http.StatusUnauthorized,
		errors.New("unauthorized")}
}

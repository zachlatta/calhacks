package main

import (
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	r := mux.NewRouter()

	http.Handle("/", r)
	http.ListenAndServe(":"+port, nil)
}

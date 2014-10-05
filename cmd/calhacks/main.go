package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/zachlatta/calhacks"
	"github.com/zachlatta/calhacks/datastore"
	"github.com/zachlatta/calhacks/handler"
)

func httpLogAndApplyCORS(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL)

		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Header().Add("Access-Control-Allow-Methods",
			"POST, PUT, DELETE, GET, OPTIONS")
		w.Header().Add("Access-Control-Request-Method", "*")
		w.Header().Add("Access-Control-Allow-Headers",
			"Origin, Content-Type, Authorization, If-Modified-Since")

		if r.Method != "OPTIONS" {
			handler.ServeHTTP(w, r)
		}

		log.Printf("Completed in %s", time.Now().Sub(start).String())
	})
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	datastore.Connect()
	defer datastore.Disconnect()

	go calhacks.Game.Run()

	m := http.NewServeMux()
	m.Handle("/", handler.Handler())

	http.ListenAndServe(":"+port, httpLogAndApplyCORS(m))
}

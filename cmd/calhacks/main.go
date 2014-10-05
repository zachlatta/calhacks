package main

import (
	"net/http"
	"os"

	"github.com/zachlatta/calhacks"
	"github.com/zachlatta/calhacks/datastore"
	"github.com/zachlatta/calhacks/handler"
)

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

	http.ListenAndServe(":"+port, m)
}

package main

import (
	"net/http"
	"os"

	"github.com/calhacks/calhacks"
	"github.com/calhacks/calhacks/datastore"
	"github.com/calhacks/calhacks/handler"
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
	m.Handle("/api/", http.StripPrefix("/api", handler.Handler()))

	http.ListenAndServe(":"+port, m)
}

package router

import "github.com/gorilla/mux"

func API() *mux.Router {
	m := mux.NewRouter()
	m.Path("/challenges").Methods("POST").Name(SubmitChallenge)
	m.Path("/challenges/current").Methods("GET").Name(CurrentChallenge)
	m.Path("/challenges/{ID:.+}").Methods("GET").Name(Challenge)
	m.Path("/connect").Methods("GET").Name(WebsocketConnect)
	return m
}

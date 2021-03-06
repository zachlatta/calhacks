package handler

import (
	"errors"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/zachlatta/calhacks"
	"github.com/zachlatta/calhacks/datastore"
	"github.com/zachlatta/calhacks/game"

	"code.google.com/p/go.net/context"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func wsConnect(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	user, _ := datastore.UserFromContext(ctx)
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if calhacks.Game.Hub.ConnForUserExists(user) {
		handleAPIError(w, r, http.StatusConflict,
			errors.New("connection for user already exists"), true)
		return
	}
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	c := game.NewConn(ws, make(chan interface{}), user)
	calhacks.Game.Hub.RegisterAndProcessConn(c)
}

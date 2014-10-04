package handler

import (
	"net/http"

	"github.com/calhacks/calhacks"
	"github.com/calhacks/calhacks/game"
	"github.com/gorilla/websocket"

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
	_ = r.URL.Query().Get("access_token")
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	c := game.NewConn(ws, make(chan []byte, 256))
	calhacks.Game.Hub.RegisterAndProcessConn(c)
}
package game

import (
	"time"

	"github.com/zachlatta/calhacks/model"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

type conn struct {
	ws   *websocket.Conn
	send chan []byte
	user *model.User
}

func NewConn(ws *websocket.Conn, send chan []byte, u *model.User) *conn {
	return &conn{ws: ws, send: send, user: u}
}

func (c *conn) readPump(h *hub) {
	defer func() {
		h.unregister <- c
	}()
	c.ws.SetReadLimit(maxMessageSize)
	c.ws.SetReadDeadline(time.Now().Add(pongWait))
	c.ws.SetPongHandler(func(string) error {
		c.ws.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		_, message, err := c.ws.ReadMessage()
		if err != nil {
			break
		}
		h.broadcast <- message
	}
}

func (c *conn) write(mt int, payload []byte) error {
	c.ws.SetWriteDeadline(time.Now().Add(writeWait))
	return c.ws.WriteMessage(mt, payload)
}

func (c *conn) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.ws.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.write(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.write(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.write(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

type hub struct {
	conns      map[*conn]bool
	broadcast  chan []byte
	register   chan *conn
	unregister chan *conn
}

func (h *hub) run() {
	for {
		select {
		case c := <-h.register:
			h.conns[c] = true
		case c := <-h.unregister:
			if _, ok := h.conns[c]; ok {
				delete(h.conns, c)
				close(c.send)
			}
		case m := <-h.broadcast:
			for c := range h.conns {
				select {
				case c.send <- m:
				default:
					close(c.send)
					delete(h.conns, c)
				}
			}
		}
	}
}

func (h *hub) RegisterAndProcessConn(c *conn) {
	h.register <- c
	go c.writePump()
	c.readPump(h)
}

type game struct {
	CurrentChallenge *model.Challenge
	Hub              hub
}

func NewGame() *game {
	return &game{
		Hub: hub{
			broadcast:  make(chan []byte),
			register:   make(chan *conn),
			unregister: make(chan *conn),
			conns:      make(map[*conn]bool),
		},
	}
}

func (g *game) Run() {
	go g.Hub.run()
}

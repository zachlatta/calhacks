package game

import (
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/websocket"
	"github.com/zachlatta/calhacks/config"
	"github.com/zachlatta/calhacks/model"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

type conn struct {
	ws   *websocket.Conn
	send chan interface{}
	user *model.User
}

func NewConn(ws *websocket.Conn, send chan interface{}, u *model.User) *conn {
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
		var evt event
		err := c.ws.ReadJSON(&evt)
		if err != nil {
			break
		}
		evt.UserID = c.user.ID

		h.events <- &evt
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
		case val, ok := <-c.send:
			if !ok {
				c.write(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.ws.WriteJSON(val); err != nil {
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
	events     chan *event
	broadcast  chan interface{}
	register   chan *conn
	unregister chan *conn
	game       *game
}

func (h *hub) run() {
	wg := sync.WaitGroup{}
	wg.Add(8)

	for i := 0; i < 8; i++ {
		go func() {
			defer wg.Done()
			select {
			case c := <-h.register:
				h.conns[c] = true
				if err := h.game.addCurrentUser(c.user); err != nil {
					log.Println(err)
				}
			case c := <-h.unregister:
				if _, ok := h.conns[c]; ok {
					delete(h.conns, c)
					close(c.send)
					if err := h.game.removeCurrentUser(c.user.ID); err != nil {
						log.Println(err)
					}
				}
			case e := <-h.events:
				fmt.Println(e)
				processEvent(h, e)
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
		}()
	}

	wg.Wait()
}

func (h *hub) RegisterAndProcessConn(c *conn) {
	h.register <- c
	go c.writePump()
	c.readPump(h)
}

type redisKey string

const (
	currentChallengeIDKey redisKey = "current_challenge_id"
	currentUserIDsKey     redisKey = "current_users"
)

type game struct {
	CurrentChallenge *model.Challenge
	Hub              hub
	pool             *redis.Pool
}

func NewGame() *game {
	g := &game{
		Hub: hub{
			broadcast:  make(chan interface{}),
			events:     make(chan *event),
			register:   make(chan *conn),
			unregister: make(chan *conn),
			conns:      make(map[*conn]bool),
		},
		pool: &redis.Pool{
			MaxIdle:     3,
			IdleTimeout: 240 * time.Second,
			Dial: func() (redis.Conn, error) {
				c, err := redis.Dial("tcp", config.RedisServer())
				if err != nil {
					return nil, err
				}
				pass := config.RedisPassword()
				if pass != "" {
					if _, err := c.Do("AUTH", pass); err != nil {
						c.Close()
						return nil, err
					}
				}
				return c, err
			},
			TestOnBorrow: func(c redis.Conn, t time.Time) error {
				_, err := c.Do("PING")
				return err
			},
		},
	}
	g.Hub.game = g
	return g
}

func (g *game) Run() {
	go g.Hub.run()
}

func (g *game) currentChallengeID() (int64, error) {
	c := g.pool.Get()
	defer c.Close()
	return redis.Int64(c.Do("GET", currentChallengeIDKey))
}

func (g *game) setCurrentChallengeID(id int64) error {
	c := g.pool.Get()
	defer c.Close()
	return (c.Send("SET", currentChallengeIDKey, id))
}

func (g *game) currentUserIDs() ([]int64, error) {
	c := g.pool.Get()
	defer c.Close()
	reply, err := redis.Strings(c.Do("SMEMBERS", currentUserIDsKey))
	if err != nil {
		return nil, err
	}
	result := make([]int64, len(reply))
	for i := 0; i < len(reply); i++ {
		var err error
		result[i], err = strconv.ParseInt(reply[i], 10, 64)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (g *game) addCurrentUser(u *model.User) error {
	c := g.pool.Get()
	defer c.Close()
	if err := c.Send("SADD", currentUserIDsKey, u.ID); err != nil {
		return err
	}
	evt := event{
		Type:   userJoined,
		UserID: u.ID,
		Body: &userJoinedEvent{
			User: u,
		},
	}
	g.Hub.broadcast <- evt
	return nil
}

func (g *game) removeCurrentUser(id int64) error {
	c := g.pool.Get()
	defer c.Close()
	if err := c.Send("SREM", currentUserIDsKey, id); err != nil {
		return err
	}
	evt := event{
		Type:   userLeft,
		UserID: id,
		Body: &userLeftEvent{
			UserID: id,
		},
	}
	g.Hub.broadcast <- evt
	return nil
}

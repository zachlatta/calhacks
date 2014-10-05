package game

import (
	"bytes"
	"database/sql"
	"fmt"
	"log"
	"runtime/debug"
	"strconv"
	"sync"
	"time"

	"code.google.com/p/go.net/context"

	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/websocket"
	"github.com/zachlatta/calhacks/config"
	"github.com/zachlatta/calhacks/datastore"
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
	conns      map[int64]*conn
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
			for {
				select {
				case c := <-h.register:
					h.conns[c.user.ID] = c
					sendInitialState(h, c)
					if err := h.game.addCurrentUser(c.user); err != nil {
						log.Println(err)
					}
				case c := <-h.unregister:
					if _, ok := h.conns[c.user.ID]; ok {
						delete(h.conns, c.user.ID)
						close(c.send)
						if err := h.game.removeCurrentUser(c.user.ID); err != nil {
							log.Println(err)
						}
					}
				case e := <-h.events:
					processEvent(h, e)
				case m := <-h.broadcast:
					for _, c := range h.conns {
						select {
						case c.send <- m:
						default:
							close(c.send)
							delete(h.conns, c.user.ID)
						}
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

func (h *hub) ConnForUserExists(user *model.User) bool {
	_, ok := h.conns[user.ID]
	return ok
}

type game struct {
	CurrentChallenge *model.Challenge
	Hub              hub
	pool             *redis.Pool
	dockerRunner     *dockerRunner
}

func NewGame() *game {
	g := &game{
		Hub: hub{
			broadcast:  make(chan interface{}),
			events:     make(chan *event),
			register:   make(chan *conn),
			unregister: make(chan *conn),
			conns:      make(map[int64]*conn),
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
		dockerRunner: &dockerRunner{
			WorkerCount: 32,
		},
	}
	g.Hub.game = g
	g.dockerRunner.hub = &g.Hub
	return g
}

type redisKey string

const (
	currentChallengeIDKey redisKey = "current_challenge_id"
	currentUserIDsKey     redisKey = "current_users"
	timeTotalKey          redisKey = "time_total"
	timeRemainingKey      redisKey = "time_remaining"
	breakKey              redisKey = "break"
)

func (g *game) currentChallengeID() (int64, error) {
	c := g.pool.Get()
	defer c.Close()
	return redis.Int64(c.Do("GET", currentChallengeIDKey))
}

func (g *game) setCurrentChallenge(chlng *model.Challenge) error {
	c := g.pool.Get()
	defer c.Close()
	if err := c.Send("SET", currentChallengeIDKey, chlng.ID); err != nil {
		return err
	}

	g.Hub.broadcast <- &event{
		Type:   challengeSet,
		UserID: -1,
		Body: &challengeSetEvent{
			Challenge: chlng,
		},
	}

	return nil
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

func (g *game) timeRemaining() (remaining int, err error) {
	c := g.pool.Get()
	defer c.Close()
	remaining, err = redis.Int(c.Do("GET", timeRemainingKey))
	if err != nil {
		return 0, err
	}
	return remaining, err
}

func (g *game) decrTimeRemaining() (finished bool, remaining int,
	err error) {
	c := g.pool.Get()
	defer c.Close()
	remaining, err = redis.Int(c.Do("DECR", timeRemainingKey))
	if err != nil {
		return false, 0, err
	}
	if remaining <= 0 {
		finished = true
	}
	return finished, remaining, err
}

func (g *game) setTimeRemaining(seconds int) error {
	c := g.pool.Get()
	defer c.Close()
	if err := c.Send("SET", timeTotalKey, seconds); err != nil {
		return err
	}
	if err := c.Send("SET", timeRemainingKey, seconds); err != nil {
		return err
	}
	return nil
}

func (g *game) totalTime() (int, error) {
	c := g.pool.Get()
	defer c.Close()
	return redis.Int(c.Do("GET", timeTotalKey))
}

func (g *game) isBreak() (bool, error) {
	c := g.pool.Get()
	defer c.Close()
	isBreak, err := redis.Bool(c.Do("GET", breakKey))
	if err != nil {
		if err == redis.ErrNil {
			return false, nil
		}
		return false, err
	}
	return isBreak, nil
}

func (g *game) setBreak(isBreak bool) error {
	c := g.pool.Get()
	defer c.Close()
	return c.Send("SET", breakKey, isBreak)
}

func (g *game) startTimer() {
	ticker := time.NewTicker(time.Second)
	for _ = range ticker.C {
		defer func() {
			if r := recover(); r != nil {
				var buf bytes.Buffer
				fmt.Println("Recover in startTimer:", r)
				buf.Write(debug.Stack())
				fmt.Println(buf.String())
			}
		}()

		ctx, cancel := context.WithCancel(context.Background())
		ctx, err := datastore.NewContextWithTx(ctx)
		if err != nil {
			panic(err)
		}

		finished, remaining, err := g.decrTimeRemaining()
		if err != nil {
			panic(err)
		}

		total, err := g.totalTime()
		if err != nil {
			panic(err)
		}

		if finished {
			g.Hub.broadcast <- &event{
				Type:   timerFinished,
				UserID: -1,
			}

			isBreak, err := g.isBreak()
			if err != nil {
				panic(err)
			}

			if isBreak {
				if err := g.setBreak(false); err != nil {
					panic(err)
				}

				var challenge *model.Challenge
				for {
					var err error
					challenge, err = datastore.GetRandomChallenge(ctx)
					if err != nil {
						// TODO: Figure out why this is failing instead of working around
						// with a loop.
						if err == sql.ErrNoRows {
							continue
						}
						panic(err)
					}
					break
				}

				if err := g.setCurrentChallenge(challenge); err != nil {
					panic(err)
				}
				if err := g.setTimeRemaining(challenge.Seconds); err != nil {
					panic(err)
				}
			} else {
				if err := g.setBreak(true); err != nil {
					panic(err)
				}
				if err := g.setTimeRemaining(3); err != nil {
					panic(err)
				}
				g.Hub.broadcast <- &event{
					Type:   breakStarted,
					UserID: -1,
				}
			}
		} else {
			g.Hub.broadcast <- &event{
				Type:   timerChanged,
				UserID: -1,
				Body: &timerChangedEvent{
					Remaining: remaining,
					Total:     total,
				},
			}
		}
		tx, _ := datastore.TxFromContext(ctx)
		tx.Commit()
		cancel()
	}
}

func (g *game) Run() {
	g.setTimeRemaining(5)
	go g.Hub.run()
	go g.startTimer()
	go g.dockerRunner.Run()
}

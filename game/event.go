package game

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"strings"

	"code.google.com/p/go.net/context"

	"github.com/zachlatta/calhacks/datastore"
	"github.com/zachlatta/calhacks/model"
)

type eventType int

const (
	userJoined eventType = iota
	userLeft
	timerChanged
	timerFinished
	challengeSet
	breakStarted
	runCode
	codeRan
	initialState
)

type userJoinedEvent struct {
	User *model.User `json:"user"`
}

type userLeftEvent struct {
	UserID int64 `json:"user_id"`
}

type timerChangedEvent struct {
	Total     int `json:"total"`
	Remaining int `json:"remaining"`
}

type challengeSetEvent struct {
	Challenge *model.Challenge `json:"challenge"`
}

type runCodeEvent struct {
	Code string `json:"code"`
	Lang string `json:"lang"`
}

type codeRanEvent struct {
	Output string `json:"output"`
	Passed bool   `json:"passed"`
}

type initialStateEvent struct {
	CurrentChallenge     *model.Challenge `json:"current_challenge"`
	CurrentUsers         []*model.User    `json:"current_users"`
	CurrentTimeRemaining int              `json:"time_remaining"`
	TotalTime            int              `json:"total_time"`
}

type event struct {
	Type   eventType   `json:"type"`
	UserID int64       `json:"user_id"`
	Body   interface{} `json:"body,omitempty"`
}

func (e *event) UnmarshalJSON(data []byte) error {
	var typeWrapper struct {
		Type eventType `json:"type"`
	}
	if err := json.Unmarshal(data, &typeWrapper); err != nil {
		return err
	}
	e.Type = typeWrapper.Type
	switch e.Type {
	case userJoined:
		var wrapper struct {
			Body userJoinedEvent `json:"body"`
		}
		if err := json.Unmarshal(data, &wrapper); err != nil {
			return err
		}
		e.Body = wrapper.Body
	case userLeft:
		var wrapper struct {
			Body userLeftEvent `json:"body"`
		}
		if err := json.Unmarshal(data, &wrapper); err != nil {
			return err
		}
		e.Body = wrapper.Body
	case timerChanged:
		var wrapper struct {
			Body timerChangedEvent `json:"body"`
		}
		if err := json.Unmarshal(data, &wrapper); err != nil {
			return err
		}
		e.Body = wrapper.Body
	case timerFinished:
		e.Body = nil
	case challengeSet:
		var wrapper struct {
			Body challengeSetEvent `json:"body"`
		}
		if err := json.Unmarshal(data, &wrapper); err != nil {
			return err
		}
		e.Body = wrapper.Body
	case breakStarted:
		e.Body = nil
	case runCode:
		var wrapper struct {
			Body runCodeEvent `json:"body"`
		}
		if err := json.Unmarshal(data, &wrapper); err != nil {
			return err
		}
		e.Body = wrapper.Body
	case codeRan:
		e.Body = nil
	case initialState:
		var wrapper struct {
			Body initialStateEvent `json:"body"`
		}
		if err := json.Unmarshal(data, &wrapper); err != nil {
			return err
		}
		e.Body = wrapper.Body
	}
	return nil
}

func processEvent(h *hub, e *event) {
	switch e.Type {
	case runCode:
		ctx, cancel := context.WithCancel(context.Background())
		ctx, err := datastore.NewContextWithTx(ctx)
		if err != nil {
			log.Println(err)
			return
		}
		defer cancel()

		tx, _ := datastore.TxFromContext(ctx)
		defer tx.Commit()

		evt := e.Body.(runCodeEvent)
		dec := base64.NewDecoder(base64.StdEncoding, strings.NewReader(evt.Code))

		chlngID, err := h.game.currentChallengeID()
		if err != nil {
			log.Println(err)
			return
		}
		chlng, err := datastore.GetChallenge(ctx, chlngID)
		if err != nil {
			log.Println(err)
			return
		}

		h.game.dockerRunner.jobs <- &dockerTask{
			c:     h.conns[e.UserID],
			code:  dec,
			lang:  evt.Lang,
			chlng: chlng,
		}
	}
}

func sendInitialState(h *hub, c *conn) {
	ctx, cancel := context.WithCancel(context.Background())
	ctx, err := datastore.NewContextWithTx(ctx)
	if err != nil {
		log.Println(err)
		return
	}
	defer cancel()

	tx, _ := datastore.TxFromContext(ctx)
	defer tx.Commit()

	chlngID, err := h.game.currentChallengeID()
	if err != nil {
		log.Println(err)
		return
	}

	chlng, err := datastore.GetChallenge(ctx, chlngID)
	if err != nil {
		log.Println(err)
		return
	}

	userIDs, err := h.game.currentUserIDs()
	if err != nil {
		log.Println(err)
		return
	}

	users := make([]*model.User, len(userIDs))
	for i, id := range userIDs {
		var err error
		users[i], err = datastore.GetUser(ctx, id)
		if err != nil {
			log.Println(err)
			return
		}
	}

	timeRemaining, err := h.game.timeRemaining()
	if err != nil {
		log.Println(err)
		return
	}

	totalTime, err := h.game.totalTime()
	if err != nil {
		log.Println(err)
		return
	}

	c.send <- &event{
		Type:   initialState,
		UserID: -1,
		Body: &initialStateEvent{
			CurrentChallenge:     chlng,
			CurrentUsers:         users,
			CurrentTimeRemaining: timeRemaining,
			TotalTime:            totalTime,
		},
	}
}

package game

import (
	"encoding/json"

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
	}
	return nil
}

func processEvent(h *hub, e *event) {
	switch e.Type {
	}
}

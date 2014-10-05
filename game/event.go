package game

import (
	"encoding/json"

	"github.com/zachlatta/calhacks/model"
)

type eventType int

const (
	userJoined eventType = iota
	userLeft
)

type userJoinedEvent struct {
	User *model.User `json:"user"`
}

type userLeftEvent struct {
	UserID int64 `json:"user_id"`
}

type event struct {
	Type   eventType   `json:"type"`
	UserID int64       `json:"user_id"`
	Body   interface{} `json:"body"`
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
	}
	return nil
}

func processEvent(h *hub, e *event) {
	switch e.Type {
	}
}

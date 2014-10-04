package handler

import (
	"encoding/json"
	"net/http"

	"github.com/calhacks/calhacks/datastore"
	"github.com/calhacks/calhacks/model"

	"code.google.com/p/go.net/context"
)

func submitChallenge(ctx context.Context, w http.ResponseWriter,
	r *http.Request) error {
	var c model.Challenge
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		return badRequest(err)
	}

	switch {
	case len(c.Title) <= 5:
		return validationError("title must be at least 5 characters long")
	case c.Seconds <= 0:
		return validationError("seconds must be at least 0")
	case c.ID != 0:
		return validationError("you cannot set the id")
	}

	for _, tc := range c.TestCases {
		switch {
		case tc.ID != 0:
			return validationError("you cannot set the id")
		}
	}

	if c.TestCases == nil {
		c.TestCases = make([]model.TestCase, 0)
	}

	if err := datastore.SaveChallenge(ctx, &c); err != nil {
		return err
	}

	return renderJSON(w, c, http.StatusCreated)
}

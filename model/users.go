package model

import "time"

type User struct {
	ID             int64     `json:"id"`
	Created        time.Time `json:"created"`
	Updated        time.Time `json:"updated"`
	Username       string    `json:"username"`
	ProfilePicture string    `json:"profile_picture"`
	GitHubID       int       `json:"-"`
	GitHubURL      string    `json:"github_url"`
	AccessToken    string    `json:"-"`
	Score          int64     `json:"score"`
}

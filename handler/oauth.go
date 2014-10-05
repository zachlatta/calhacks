package handler

import (
	"database/sql"
	"net/http"

	"github.com/calhacks/calhacks/config"
	"github.com/calhacks/calhacks/datastore"
	"github.com/calhacks/calhacks/model"
	"github.com/google/go-github/github"

	"code.google.com/p/go.net/context"
	"code.google.com/p/goauth2/oauth"
)

func oauthLogin(ctx context.Context, w http.ResponseWriter,
	r *http.Request) error {
	http.Redirect(w, r, config.GitHubOauthConfig().AuthCodeURL(""),
		http.StatusTemporaryRedirect)
	return nil
}

func oauthAccessToken(ctx context.Context, w http.ResponseWriter,
	r *http.Request) error {
	t := &oauth.Transport{Config: config.GitHubOauthConfig()}
	t.Exchange(r.FormValue("code"))
	client := github.NewClient(t.Client())

	ghUser, _, err := client.Users.Get("")
	if err != nil {
		return err
	}

	var user *model.User
	user, err = datastore.GetUserByGitHubID(ctx, *ghUser.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			user = &model.User{}
		} else {
			return err
		}
	}

	user.Username = *ghUser.Login
	user.ProfilePicture = *ghUser.AvatarURL
	user.GitHubID = *ghUser.ID
	user.GitHubURL = *ghUser.HTMLURL
	user.AccessToken = t.Token.AccessToken

	if err := datastore.SaveUser(ctx, user); err != nil {
		return err
	}

	return nil
}
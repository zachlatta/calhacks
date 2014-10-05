package datastore

import (
	"time"

	"github.com/calhacks/calhacks/model"

	"code.google.com/p/go.net/context"
)

const createUserStmt = `INSERT INTO users (created, updated, username,
profile_picture, github_id, github_url, access_token) VALUES ($1, $2, $3, $4,
$5, $6, $7) RETURNING id`

const getUserStmt = `SELECT id, created, updated, username, profile_picture,
github_id, github_url, access_token FROM users WHERE id=$1`

const getUserByGitHubIDStmt = `SELECT id, created, updated, username,
profile_picture, github_id, github_url, access_token FROM users WHERE
github_id=$1`

const updateUserStmt = `UPDATE users SET updated=$2, username=$3,
profile_picture=$4, github_id=$5, github_url=$6, access_token=$7 WHERE id=$1`

func SaveUser(ctx context.Context, u *model.User) error {
	tx, _ := TxFromContext(ctx)

	var newUser bool
	if u.ID == 0 { // User has not previously been saved
		u.Created = time.Now()
		newUser = true
	}
	u.Updated = time.Now()
	if newUser {
		rows, err := tx.Query(createUserStmt, u.Created, u.Updated, u.Username,
			u.ProfilePicture, u.GitHubID, u.GitHubURL, u.AccessToken)
		if err != nil {
			return err
		}
		for rows.Next() {
			if err := rows.Scan(&u.ID); err != nil {
				return err
			}
		}
		if err := rows.Err(); err != nil {
			return err
		}
	} else {
		if _, err := tx.Exec(updateUserStmt, u.ID, u.Updated, u.Username,
			u.ProfilePicture, u.GitHubID, u.GitHubURL, u.AccessToken); err != nil {
			return err
		}
	}
	return nil
}

func GetUser(ctx context.Context, id int64) (*model.User, error) {
	return getUser(ctx, getUserStmt, id)
}

func GetUserByGitHubID(ctx context.Context, id int) (*model.User, error) {
	return getUser(ctx, getUserByGitHubIDStmt, int64(id))
}

func getUser(ctx context.Context, stmt string, id int64) (*model.User, error) {
	tx, _ := TxFromContext(ctx)
	u := model.User{}
	row := tx.QueryRow(stmt, id)
	if err := row.Scan(&u.ID, &u.Created, &u.Updated, &u.Username,
		&u.ProfilePicture, &u.GitHubID, &u.GitHubURL, &u.AccessToken); err != nil {
		return nil, err
	}
	return &u, nil
}

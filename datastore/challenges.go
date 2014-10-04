package datastore

import (
	"code.google.com/p/go.net/context"
	"github.com/calhacks/calhacks/model"
)

const createChlngStmt = `INSERT INTO challenges (created, updated, title,
description, seconds) VALUES ($1, $2, $3, $4, $5) RETURNING id`

const getChlngStmt = `SELECT id, created, updated, title, description, seconds
FROM challenges WHERE id=$1`

const getChlngTestCasesStmt = `
SELECT id, created, updated
FROM challenge_test_cases
WHERE challenge_id=$1
`

func GetChallenge(ctx context.Context, id int64) (*model.Challenge, error) {
	tx, _ := TxFromContext(ctx)

	c := model.Challenge{}
	row := tx.QueryRow(getChlngStmt, id)
	if err := row.Scan(&c.ID, &c.Created, &c.Updated, &c.Title, &c.Description,
		&c.Seconds); err != nil {
		return nil, err
	}
	rows, err := tx.Query(getChlngTestCasesStmt, id)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		t := model.TestCase{}
		if err := rows.Scan(&t.ID, &t.Created, &t.Updated); err != nil {
			return nil, err
		}
		c.TestCases = append(c.TestCases, t)
	}

	return &c, nil
}

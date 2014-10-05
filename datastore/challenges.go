package datastore

import (
	"fmt"
	"time"

	"code.google.com/p/go.net/context"
	"github.com/zachlatta/calhacks/model"
)

const createChlngStmt = `INSERT INTO challenges (created, updated, title,
description, seconds, expected_output) VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id`

const createTestCaseStmt = `INSERT INTO challenge_test_cases (created, updated,
challenge_id) VALUES ($1, $2, $3) RETURNING id`

const getChlngStmt = `SELECT id, created, updated, title, description, seconds,
expected_output FROM challenges WHERE id=$1`

const getChlngTestCasesStmt = `
SELECT id, created, updated
FROM challenge_test_cases
WHERE challenge_id=$1
`

const getRandChlngIDStmt = `
SELECT id FROM challenges
OFFSET random()*(SELECT count(*) FROM challenges)
LIMIT 1`

// TODO: Cancel if context cancels.
func SaveChallenge(ctx context.Context, c *model.Challenge) error {
	tx, _ := TxFromContext(ctx)

	var newChallenge bool
	if c.ID == 0 {
		c.Created = time.Now()
		newChallenge = true
	}
	c.Updated = time.Now()

	if newChallenge {
		rows, err := tx.Query(createChlngStmt, c.Created, c.Updated, c.Title,
			c.Description, c.Seconds, c.ExpectedOutput)
		if err != nil {
			return err
		}
		for rows.Next() {
			if err := rows.Scan(&c.ID); err != nil {
				return err
			}
		}
		if err := rows.Err(); err != nil {
			return err
		}
	} else {
		fmt.Println("NOT IMPLEMENTED")
	}

	for i := 0; i < len(c.TestCases); i++ {
		if err := SaveTestCase(ctx, &c.TestCases[i], c.ID); err != nil {
			return err
		}
	}
	return nil
}

func SaveTestCase(ctx context.Context, tc *model.TestCase,
	challengeID int64) error {
	tx, _ := TxFromContext(ctx)

	var newTc bool
	if tc.ID == 0 {
		tc.Created = time.Now()
		newTc = true
	}
	tc.Updated = time.Now()

	if newTc {
		rows, err := tx.Query(createTestCaseStmt, tc.Created, tc.Updated,
			challengeID)
		if err != nil {
			return err
		}
		for rows.Next() {
			if err := rows.Scan(&tc.ID); err != nil {
				return err
			}
		}
		if err := rows.Err(); err != nil {
			return err
		}
	} else {
		fmt.Println("NOT IMPLEMENTED")
	}
	return nil
}

func GetChallenge(ctx context.Context, id int64) (*model.Challenge, error) {
	tx, _ := TxFromContext(ctx)

	c := model.Challenge{}
	row := tx.QueryRow(getChlngStmt, id)
	if err := row.Scan(&c.ID, &c.Created, &c.Updated, &c.Title, &c.Description,
		&c.Seconds, &c.ExpectedOutput); err != nil {
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

func GetRandomChallenge(ctx context.Context) (*model.Challenge, error) {
	tx, _ := TxFromContext(ctx)
	var id int64
	row := tx.QueryRow(getRandChlngIDStmt)
	if err := row.Scan(&id); err != nil {
		return nil, err
	}
	return GetChallenge(ctx, id)
}

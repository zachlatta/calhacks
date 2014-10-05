package model

import "time"

type TestCase struct {
	ID      int64     `json:"id"`
	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`
}

type Challenge struct {
	ID             int64      `json:"id"`
	Created        time.Time  `json:"created"`
	Updated        time.Time  `json:"updated"`
	Title          string     `json:"title"`
	Description    string     `json:"description"`
	Seconds        int        `json:"seconds"`
	ExpectedOutput string     `json:"-"`
	TestCases      []TestCase `json:"test_cases"`
}

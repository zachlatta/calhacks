package game

import (
	"fmt"
	"io"
	"sync"
)

const (
	imgBase = "zachlatta/calhacks-"

	imgRuby = imgBase + "ruby"
)

type dockerResult struct {
	output  string
	success bool
	err     error
}

type dockerTask struct {
	c    *conn
	lang string
	code io.Reader
}

type dockerRunner struct {
	WorkerCount int
	Timeout     int // TODO: Use this

	hub     *hub
	jobs    chan *dockerTask
	results chan *dockerResult
}

func (b *dockerRunner) Run() {
	b.results = make(chan *dockerResult)
	b.run()
}

func (b *dockerRunner) worker(ch chan *dockerTask) {
	for t := range ch {
		fmt.Println(t)
		t.c.send <- &event{
			Type:   codeRan,
			UserID: t.c.user.ID,
			Body: &codeRanEvent{
				Output: "Foobar",
				Passed: true,
			},
		}
	}
}

func (b *dockerRunner) run() {
	var wg sync.WaitGroup
	wg.Add(b.WorkerCount)
	b.jobs = make(chan *dockerTask)
	for i := 0; i < b.WorkerCount; i++ {
		go func() {
			b.worker(b.jobs)
			wg.Done()
		}()
	}
	wg.Wait()
	close(b.jobs)
}

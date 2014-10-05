package game

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fsouza/go-dockerclient"
	"github.com/zachlatta/calhacks/model"
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
	c     *conn
	lang  string
	code  io.Reader
	chlng *model.Challenge
}

type dockerRunner struct {
	WorkerCount int
	Timeout     int // TODO: Use this

	docker *docker.Client

	hub     *hub
	jobs    chan *dockerTask
	results chan *dockerResult
}

func (b *dockerRunner) Run() {
	c, err := docker.NewClient("unix://var/run/docker.sock")
	if err != nil {
		panic(err)
	}
	b.docker = c
	b.results = make(chan *dockerResult)
	b.run()
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func resolveLangToImg(lang string) (string, error) {
	switch lang {
	case "ruby":
		return imgRuby, nil
	}
	return "", errors.New("cannot resolve language to image")
}

func (b *dockerRunner) worker(ch chan *dockerTask) {
	for t := range ch {
		img, err := resolveLangToImg(t.lang)
		if err != nil {
			log.Println(err)
			continue
		}
		base := fmt.Sprintf("/tmp/calhacks/%s", randSeq(26))
		filename := fmt.Sprintf("%s/%s", base, randSeq(26))

		if err := os.Mkdir(base, 0644); err != nil {
			log.Println(err)
			continue
		}

		file, err := os.Create(filename)
		if err != nil {
			log.Println(err)
			continue
		}

		io.Copy(file, t.code)

		container, err := b.docker.CreateContainer(docker.CreateContainerOptions{
			Config: &docker.Config{
				Image: img,
				Cmd:   []string{filename},
			},
		})
		if err != nil {
			log.Println(err)
			continue
		}

		var buf bytes.Buffer

		if err := b.docker.StartContainer(container.ID, &docker.HostConfig{
			Binds: []string{fmt.Sprintf("%s:%s", base, base)},
		}); err != nil {
			log.Println(err)
			continue
		}
		if err := b.docker.Logs(docker.LogsOptions{
			Container:    container.ID,
			OutputStream: &buf,
			ErrorStream:  &buf,
			Stdout:       true,
			Stderr:       true,
			Follow:       true,
		}); err != nil {
			panic(err)
		}

		t.c.send <- &event{
			Type:   codeRan,
			UserID: t.c.user.ID,
			Body: &codeRanEvent{
				Output: buf.String(),
				Passed: strings.TrimSpace(buf.String()) == t.chlng.ExpectedOutput,
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

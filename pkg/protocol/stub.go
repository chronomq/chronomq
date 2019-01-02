package protocol

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

//SrvStub implements a stub beanstalkd instance
type SrvStub struct {
	tubes map[string]Tube
}

// TubeStub implements a stub beanstalkd tube
type TubeStub struct {
	name     string
	jobs     map[string]*Job
	reserved map[string]*Job
	paused   bool
	jobIDCtr int
}

// Job represents a beanstalkd job
type Job struct {
	delay time.Duration
	id    string
	pri   int32
	ttr   time.Duration
	body  []byte
	size  int
}

// BeanstalkdSrv implements the beanstalkd server responsibilities
type BeanstalkdSrv interface {
	listTubes() []string
	getTube(name string) (Tube, error)
	stop(persist bool)
}

// Tube is a beanstalkd tube
type Tube interface {
	pauseTube(delay time.Duration) error
	put(delay int, pri int32, body []byte, ttr int) (string, error)
	reserve(timeoutSec string) *Job
	deleteJob(id int) error
	stop(persist bool)
}

// ErrTubeNotFound is thrown when doing an operation against an unknown tube
var ErrTubeNotFound = errors.New("tube not found")

// ErrJobNotFound is thrown when doing an operation against an unknown job
var ErrJobNotFound = errors.New("job not found")

// NewSrvStub returns a stub BeanstalkdSrv
func NewSrvStub() BeanstalkdSrv {
	stub := SrvStub{make(map[string]Tube)}
	t := &TubeStub{
		name:     "default",
		jobs:     make(map[string]*Job),
		reserved: make(map[string]*Job),
		paused:   false}
	stub.tubes[t.name] = t
	return &stub
}

func (s *SrvStub) stop(persist bool) {
	// noop
}

func (s *SrvStub) listTubes() []string {
	keys := make([]string, len(s.tubes))
	i := 0
	for k := range s.tubes {
		keys[i] = k
		i++
	}
	return keys
}

func (s *SrvStub) getTube(name string) (Tube, error) {
	t, ok := s.tubes[name]
	if !ok {
		return nil, ErrTubeNotFound
	}
	return t, nil
}

func (t *TubeStub) stop(persist bool) {
	// noop
}

func (t *TubeStub) pauseTube(delay time.Duration) error {
	t.paused = true
	return nil
}

func (t *TubeStub) put(delay int, pri int32, body []byte, ttr int) (string, error) {
	logrus.Debug("Putting job in stub")
	j := &Job{
		id:    strconv.Itoa(t.jobIDCtr),
		delay: time.Duration(delay) * time.Second,
		pri:   pri,
		size:  len(body),
		body:  body,
		ttr:   time.Duration(ttr) * time.Second,
	}

	t.jobs[j.id] = j
	t.jobIDCtr++
	return j.id, nil
}

func (t *TubeStub) reserve(timeoutSec string) *Job {
	// ts, err := strconv.Atoi(timeoutSec)
	// if err != nil {
	// 	return nil
	// }
	for k := range t.jobs {
		j := t.jobs[k]
		t.reserved[j.id] = j
		return j
	}

	return nil
}

func (t *TubeStub) deleteJob(id int) error {
	sid := fmt.Sprintf("%d", id)
	_, ok := t.jobs[sid]
	if ok {
		delete(t.jobs, sid)
		return nil
	}
	_, ok = t.reserved[sid]
	if ok {
		delete(t.reserved, sid)
		return nil
	}
	return ErrJobNotFound
}

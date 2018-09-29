package protocol

import (
	"errors"
	"time"
)

//SrvStub implements a stub beanstalkd instance
type SrvStub struct {
	tubes map[string]Tube
}

// TubeStub implements a stub beanstalkd tube
type TubeStub struct {
	name     string
	jobs     map[int]*Job
	reserved map[int]*Job
	paused   bool
	jobIDCtr int
}

// Job represents a beanstalkd job
type Job struct {
	delay time.Duration
	id    int
	pri   int32
	ttr   time.Duration
	body  []byte
	size  int
}

// BeanstalkdSrv implements the beanstalkd server responsibilities
type BeanstalkdSrv interface {
	listTubes() []string
	getTube(name string) (Tube, error)
}

// Tube is a beanstalkd tube
type Tube interface {
	pauseTube(delay time.Duration) error
	put(delay int, pri int32, body []byte, ttr int) (int, error)
	reserve() *Job
	deleteJob(id int) error
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
		jobs:     make(map[int]*Job),
		reserved: make(map[int]*Job),
		paused:   false}
	stub.tubes[t.name] = t
	return &stub
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

func (t *TubeStub) pauseTube(delay time.Duration) error {
	t.paused = true
	return nil
}

func (t *TubeStub) put(delay int, pri int32, body []byte, ttr int) (int, error) {
	j := &Job{
		id:    t.jobIDCtr,
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

func (t *TubeStub) reserve() *Job {
	for k := range t.jobs {
		j := t.jobs[k]
		t.reserved[j.id] = j
		return j
	}

	return nil
}

func (t *TubeStub) deleteJob(id int) error {
	_, ok := t.jobs[id]
	if ok {
		delete(t.jobs, id)
		return nil
	}
	_, ok = t.reserved[id]
	if ok {
		delete(t.reserved, id)
		return nil
	}
	return ErrJobNotFound
}

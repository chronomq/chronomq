package protocol

import (
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/urjitbhatia/goyaad/pkg/goyaad"
)

// SrvYaad implements a yaad beanstalkd instance
type SrvYaad struct {
	tubes map[string]Tube
}

// TubeYaad implements a yaad hub as a beanstalkd tube
type TubeYaad struct {
	name     string
	paused   bool
	jobIDCtr int
	// Backed by a yaad hub
	hub *goyaad.Hub
}

// NewSrvYaad returns a yaad BeanstalkdSrv
func NewSrvYaad(hub *goyaad.Hub) BeanstalkdSrv {
	y := SrvYaad{make(map[string]Tube)}
	t := &TubeYaad{
		name:   "default",
		paused: false,
		hub:    hub,
	}
	y.tubes[t.name] = t
	return &y
}

func (s *SrvYaad) stop(persist bool) {
	for _, t := range s.tubes {
		t.stop(persist)
	}
}

func (s *SrvYaad) listTubes() []string {
	keys := make([]string, len(s.tubes))
	i := 0
	for k := range s.tubes {
		keys[i] = k
		i++
	}
	return keys
}

func (s *SrvYaad) getTube(name string) (Tube, error) {
	t, ok := s.tubes[name]
	if !ok {
		return nil, ErrTubeNotFound
	}
	return t, nil
}

func (t *TubeYaad) stop(persist bool) {
	t.hub.Stop(persist)
}

func (t *TubeYaad) pauseTube(delay time.Duration) error {
	t.paused = true
	return nil
}

func (t *TubeYaad) put(delay int, pri int32, body []byte, ttr int) (string, error) {
	j := goyaad.NewJobAutoID(time.Now().Add(time.Second*time.Duration(delay)), body)
	j.SetOpts(pri, time.Duration(ttr)*time.Second)

	err := t.hub.AddJob(j)
	if err != nil {
		return "", err
	}
	t.jobIDCtr++
	return j.ID(), nil
}

func (t *TubeYaad) reserve(timeoutSec string) *Job {
	ts, err := strconv.Atoi(timeoutSec)
	if err != nil {
		logrus.Errorf("Error parsing timeout: %s", err)
		return nil
	}

	logrus.Debug("yaad srv reserve")
	// try once
	if j := t.hub.Next(); j != nil {
		return &Job{
			body: j.Body(),
			id:   j.ID(),
			size: len(j.Body()),
		}
	}
	if ts == 0 {
		return nil
	}

	waitTill := time.Now().Add(time.Duration(ts) * time.Second)
	// wait for timeout and keep trying
	logrus.Debug("waiting for reserve: ", timeoutSec)
	for waitTill.After(time.Now()) {
		if j := t.hub.Next(); j != nil {
			return &Job{
				body: j.Body(),
				id:   j.ID(),
				size: len(j.Body()),
			}
		}
		time.Sleep(time.Millisecond * 200)
	}
	logrus.Debug("yaad srv reserve done - no job found")
	return nil
}

func (t *TubeYaad) deleteJob(id int) error {
	strID := strconv.Itoa(id)
	return t.hub.CancelJob(strID)
}

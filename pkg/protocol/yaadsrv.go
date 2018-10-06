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
	reserved map[string]*goyaad.Job
	paused   bool
	jobIDCtr int
	// Backed by a yaad hub
	hub *goyaad.Hub
}

// NewSrvYaad returns a yaad BeanstalkdSrv
func NewSrvYaad() BeanstalkdSrv {
	y := SrvYaad{make(map[string]Tube)}
	t := &TubeYaad{
		name:     "default",
		reserved: make(map[string]*goyaad.Job),
		paused:   false,
		hub:      goyaad.NewHub(time.Second * 5),
	}
	y.tubes[t.name] = t
	return &y
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

func (t *TubeYaad) pauseTube(delay time.Duration) error {
	t.paused = true
	return nil
}

func (t *TubeYaad) put(delay int, pri int32, body []byte, ttr int) (string, error) {
	j := goyaad.NewJobAutoID(time.Now().Add(time.Second*time.Duration(delay)), &body)
	j.SetOpts(pri, time.Duration(ttr)*time.Second)

	t.hub.AddJob(j)
	t.jobIDCtr++
	return j.ID(), nil
}

func (t *TubeYaad) reserve() *Job {
	// Walk with iterator pattern?
	logrus.Debug("yaad srv reserve")
	for _, j := range *t.hub.Walk() {
		return &Job{
			body: *j.Body(),
			id:   j.ID(),
			size: len(*j.Body()),
		}
	}
	logrus.Debug("yaad srv reserve done")
	return nil
}

// Todo: handle cancelations for reserved jobs
func (t *TubeYaad) deleteJob(id int) error {

	strID := strconv.Itoa(id)
	t.hub.CancelJob(strID)
	return nil
}

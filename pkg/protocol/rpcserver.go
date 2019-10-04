package protocol

import (
	"io"
	"net"
	"net/rpc"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/urjitbhatia/goyaad/pkg/goyaad"
)

// ErrTimeout indicates that no new jobs were ready to be consumed within the given timeout duration
var ErrTimeout = errors.New("No new jobs available in given timeout")

// RPCServer exposes a Yaad hub backed RPC endpoint
type RPCServer struct {
	hub *goyaad.Hub
}

// RPCJob is a light wrapper struct representing job data on the wire without extra metadata that is stored internally
type RPCJob struct {
	Body  []byte
	ID    string
	Delay time.Duration
}

func newRPCServer(hub *goyaad.Hub) *RPCServer {
	return &RPCServer{hub: hub}
}

// PutWithID accepts a new job and stores it in a Hub, reply is ignored
func (r *RPCServer) PutWithID(job RPCJob, id *string) error {
	var j *goyaad.Job
	if job.ID == "" {
		// need to generate an id
		j = goyaad.NewJobAutoID(time.Now().Add(job.Delay), job.Body)
		*id = j.ID()
	} else {
		j = goyaad.NewJob(job.ID, time.Now().Add(job.Delay), job.Body)
	}
	return r.hub.AddJobLocked(j)
}

// Cancel deletes the job pointed to by the id, reply is ignored
// If the job doesn't exist, no error is returned so calls to Cancel are idempotent
func (r *RPCServer) Cancel(id string, ignoredReply *int8) error {
	return r.hub.CancelJobLocked(id)
}

// Next sets the reply (job) to a valid job if a job is ready to be triggered
// If not job is ready yet, this call will wait (block) for the given duration and keep searching
// for ready jobs. If no job is ready by the end of the timeout, ErrTimeout is returned
func (r *RPCServer) Next(timeout time.Duration, job *RPCJob) error {
	// try once
	if j := r.hub.NextLocked(); j != nil {
		job.Body = j.Body()
		job.ID = j.ID()
		return nil
	}
	// if we couldn't find a ready job and timeout was set to 0
	if timeout.Seconds() == 0 {
		return ErrTimeout
	}

	waitTill := time.Now().Add(timeout)
	// wait for timeout and keep trying
	log.Debug().
		Dur("timeout", timeout).
		Time("now", time.Now()).
		Time("waitTill", waitTill).
		Msg("waiting for reserve")
	for waitTill.After(time.Now()) {
		if j := r.hub.NextLocked(); j != nil {
			job.Body = j.Body()
			job.ID = j.ID()
			return nil
		}
		time.Sleep(time.Millisecond * 200)
		log.Debug().Dur("timeout", timeout).Msg("waiting for reserve finished sleep duration")
	}

	return ErrTimeout
}

// Ping the server, sets "pong" as the reply
// useful for basic connectivity/liveness check
func (r *RPCServer) Ping(ignore int8, pong *string) error {
	log.Debug().Msg("Received ping from client")
	*pong = "pong"
	return nil
}

// InspectN returns n jobs without removing them for ad-hoc inspection
func (r *RPCServer) InspectN(n int, rpcJobs *[]*RPCJob) error {
	if n == 0 {
		return nil
	}
	log.Debug().Int("count", n).Msg("Returning jobs for inspection")
	jobs := r.hub.GetNJobs(n)

	for j := range jobs {
		rpcJob := &RPCJob{
			Body:  j.Body(),
			ID:    j.ID(),
			Delay: j.TriggerAt().Sub(time.Now()),
		}
		*rpcJobs = append(*rpcJobs, rpcJob)
	}
	return nil
}

// ServeRPC starts serving hub over rpc
func ServeRPC(hub *goyaad.Hub, addr string) (io.Closer, error) {
	srv := newRPCServer(hub)
	rpcSrv := rpc.NewServer()
	rpcSrv.Register(srv)
	l, e := net.Listen("tcp", addr)
	if e != nil {
		return nil, e
	}
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				log.Error().Err(err).Msg("Cannot handle client connection")
				return
			}
			go rpcSrv.ServeConn(conn)
		}
	}()
	return l, nil
}

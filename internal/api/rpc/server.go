package rpc

import (
	"io"
	"net"
	"net/rpc"
	"time"

	"github.com/rs/zerolog/log"

	yaadrpc "github.com/urjitbhatia/goyaad/api/rpc/goyaad"
	"github.com/urjitbhatia/goyaad/pkg/goyaad"
)

// Server exposes a Yaad hub backed RPC endpoint
type Server struct {
	hub *goyaad.Hub
}

func newServer(hub *goyaad.Hub) *Server {
	return &Server{hub: hub}
}

// PutWithID accepts a new job and stores it in a Hub, reply is ignored
func (r *Server) PutWithID(job yaadrpc.Job, id *string) error {
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
func (r *Server) Cancel(id string, ignoredReply *int8) error {
	return r.hub.CancelJobLocked(id)
}

// Next sets the reply (job) to a valid job if a job is ready to be triggered
// If not job is ready yet, this call will wait (block) for the given duration and keep searching
// for ready jobs. If no job is ready by the end of the timeout, ErrTimeout is returned
func (r *Server) Next(timeout time.Duration, job *yaadrpc.Job) error {
	// try once
	if j := r.hub.NextLocked(); j != nil {
		job.Body = j.Body()
		job.ID = j.ID()
		return nil
	}
	// if we couldn't find a ready job and timeout was set to 0
	if timeout.Seconds() == 0 {
		return yaadrpc.ErrTimeout
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

	return yaadrpc.ErrTimeout
}

// Ping the server, sets "pong" as the reply
// useful for basic connectivity/liveness check
func (r *Server) Ping(ignore int8, pong *string) error {
	log.Debug().Msg("Received ping from client")
	*pong = "pong"
	return nil
}

// InspectN returns n jobs without removing them for ad-hoc inspection
func (r *Server) InspectN(n int, rpcJobs *[]*yaadrpc.Job) error {
	if n == 0 {
		return nil
	}
	log.Debug().Int("count", n).Msg("Returning jobs for inspection")
	jobs := r.hub.GetNJobs(n)

	for j := range jobs {
		rpcJob := &yaadrpc.Job{
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
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	yaadRPCSrv := newServer(hub)
	rpcSrv := rpc.NewServer()
	err = rpcSrv.Register(yaadRPCSrv)
	if err != nil {
		return nil, err
	}
	go func() {
		log.Info().Str("Addr", l.Addr().String()).Msg("Starting RPC Server")
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

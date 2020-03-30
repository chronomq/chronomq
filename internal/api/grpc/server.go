package grpc

//go:generate go get github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway
//go:generate go get github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger
//go:generate go get github.com/grpc-ecosystem/grpc-gateway/runtime
//go:generate go get github.com/golang/protobuf/protoc-gen-go
//go:generate go get github.com/ckaznocha/protoc-gen-lint
//go:generate protoc -I../../../api/grpc/chronomq -I./thirdparty -I$GOPATH/bin --lint_out=. --go_out=plugins=grpc:../../../api/grpc/chronomq/ --grpc-gateway_out=logtostderr=true:../../../api/grpc/chronomq/ --swagger_out=logtostderr=true:../../../api/grpc/chronomq/ service.proto

import (
	context "context"
	"errors"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/golang/protobuf/ptypes"
	duration "github.com/golang/protobuf/ptypes/duration"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	api "github.com/chronomq/chronomq/api/grpc/chronomq"
	"github.com/chronomq/chronomq/internal/monitor"
	"github.com/chronomq/chronomq/pkg/chronomq"
)

var memMonitor monitor.MemMonitor

// Server is used to implement Chronomq GRPCServer
type Server struct {
	*grpc.Server
	hub *chronomq.Hub
}

// Put saves a new job. It can error if another job with the ID already exists in the system
func (s *Server) Put(ctx context.Context, pr *api.PutRequest) (*api.PutResponse, error) {
	memMonitor.Fence()

	log.Debug().Msgf("GRPC: Putting new job : %v", pr)

	// Validate input
	if pr.GetJob().GetDelay() == nil {
		err := errors.New("PutRequest cannot have a nil delay")
		log.Error().Err(err)
		return nil, err
	}
	delay, err := ptypes.Duration(pr.GetJob().GetDelay())
	if err != nil {
		err := errors.New("Cannot parse put request delay")
		log.Error().Err(err)
		return nil, err
	}

	// Add job to hub
	triggerAt := time.Now().Add(delay)

	if pr.GetJob().GetId() == "" {
		err := errors.New("PutRequest cannot have an empty Job ID")
		log.Error().Err(err)
		return nil, err
	}

	j := chronomq.NewJob(pr.GetJob().GetId(), triggerAt, pr.GetJob().GetBody())
	err = s.hub.AddJobLocked(j)
	if err != nil {
		return nil, err
	}
	defer memMonitor.Increment(j)
	return &api.PutResponse{}, nil
}

// Cancel a job with the given ID if it exists. This call is idempotent
func (s *Server) Cancel(ctx context.Context, cr *api.CancelRequest) (*api.CancelResponse, error) {
	var err error
	if cr.GetId() != "" {
		job, err := s.hub.CancelJobLocked(cr.GetId())
		if err == nil && job != nil {
			defer memMonitor.Decrement(job)
		}
	}
	return &api.CancelResponse{}, err
}

func asNextResponse(job *chronomq.Job) *api.NextResponse {
	return &api.NextResponse{
		Next: &api.NextResponse_Job{
			Job: &api.Job{
				Id:    job.ID(),
				Delay: &duration.Duration{Seconds: int64(job.TriggerAt().Sub(time.Now()).Seconds())},
				Body:  job.Body(),
			},
		},
	}
}

// Next returns the next job that is ready to be consumed. If no job is ready, response is empty
func (s *Server) Next(ctx context.Context, nr *api.NextRequest) (*api.NextResponse, error) {
	job := s.hub.NextLocked()
	if job != nil {
		memMonitor.Decrement(job)

		log.Debug().
			Str("jobID", job.ID()).
			Bytes("jobBody", job.Body()).
			Msgf("Responding with job with id and body")
		return asNextResponse(job), nil
	}

	timeout, err := ptypes.Duration(nr.GetTimeout())
	if err != nil {
		log.Error().Err(err).Msg("Cannot parse next request timeout")
		return nil, err
	}

	waitTill := time.Now().Add(timeout)
	// wait for timeout and keep trying
	log.Debug().
		Dur("timeout", timeout).
		Time("now", time.Now()).
		Time("waitTill", waitTill).
		Msg("waiting for reserve")

	for waitTill.After(time.Now()) {
		if job = s.hub.NextLocked(); job != nil {
			defer monitor.GetMemMonitor().Decrement(job)
			return asNextResponse(job), nil
		}
		time.Sleep(time.Millisecond * 200)
		log.Debug().Dur("timeout", timeout).Msg("waiting for reserve finished sleep duration")
	}

	return &api.NextResponse{}, nil
}

// Inspect returns upto N jobs. Can return none
func (s *Server) Inspect(req *api.InspectRequest, srv api.Chronomq_InspectServer) error {
	jobs := s.hub.GetNJobs(int(req.GetCount()))
	log.Info().Int32("count", req.GetCount()).Int("joblen", len(jobs)).Msg("Returning jobs for inspection")

	for j := range jobs {
		grpcJob := &api.Job{
			Id:    j.ID(),
			Delay: ptypes.DurationProto(j.TriggerAt().Sub(time.Now())),
			Body:  j.Body(),
		}
		err := srv.SendMsg(&api.InspectResponse{
			Job: grpcJob,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// Close the server listener (gracefully)
func (s *Server) Close() error {
	s.GracefulStop()
	return nil
}

// ServeHTTP starts a proxy listener that listens runs a gRPC-JSON
func (s *Server) ServeHTTP(hAddr, gAddr string) (io.Closer, error) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	// Register gRPC server endpoint
	// Note: Make sure the gRPC server is running properly and accessible
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}
	err := api.RegisterChronomqHandlerFromEndpoint(ctx, mux, gAddr, opts)
	if err != nil {
		log.Fatal().Err(err).Msg("GRPC:HTTPGW: Failed to register grpc-http-gateway")
	}
	// Start HTTP server (and proxy calls to gRPC server endpoint)
	srv := &http.Server{
		Addr:    hAddr,
		Handler: mux,
	}

	go func() {
		defer cancel()
		log.Info().Str("Addr", srv.Addr).Msg("Starting GRPC-JSON http Server")
		if err = srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("GRPC:HTTPGW: Failed to serve grpc-http-gateway requests")
		}
	}()
	return srv, nil
}

// ServeGRPC creates a new GRPC capable server backed by a yaad hub &
// and starts accepting connections on its GRPC listener
func ServeGRPC(hub *chronomq.Hub, addr string) (io.Closer, error) {
	memMonitor = monitor.GetMemMonitor()
	s := &Server{hub: hub, Server: grpc.NewServer()}

	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	reflection.Register(s.Server)
	api.RegisterChronomqServer(s.Server, s)
	go func() {
		log.Info().Str("Addr", l.Addr().String()).Msg("Starting GRPC Server")
		if err := s.Serve(l); err != nil {
			log.Fatal().Err(err).Msg("GRPC: Failed to serve requests")
		}
	}()
	return s, nil
}

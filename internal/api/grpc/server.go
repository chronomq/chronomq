package grpc

//go:generate go install github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway
//go:generate go install github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger
//go:generate go install github.com/ckaznocha/protoc-gen-lint
//go:generate protoc -I../../../api/grpc/goyaad -I./thirdparty -I$GOPATH/bin --lint_out=. --go_out=plugins=grpc:../../../api/grpc/goyaad/ --grpc-gateway_out=logtostderr=true:../../../api/grpc/goyaad/ --swagger_out=logtostderr=true:../../../api/grpc/goyaad/ service.proto

import (
	context "context"
	"errors"
	"io"
	"net"
	"net/http"
	"time"

	duration "github.com/golang/protobuf/ptypes/duration"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	yaadgrpc "github.com/urjitbhatia/goyaad/api/grpc/goyaad"
	"github.com/urjitbhatia/goyaad/pkg/goyaad"
)

// Server is used to implement YaadServer
type Server struct {
	*grpc.Server
	hub *goyaad.Hub
}

// Put saves a new job. It can error if another job with the ID already exists in the system
func (s *Server) Put(ctx context.Context, pr *yaadgrpc.PutRequest) (*yaadgrpc.PutResponse, error) {
	log.Debug().Msgf("GRPC: Putting new job : %v", pr)

	// Validate input
	delay := pr.GetJob().GetDelay()
	if delay == nil {
		err := errors.New("PutRequest cannot have a nil delay")
		log.Error().Err(err)
		return nil, err
	}
	if pr.GetJob().GetId() == "" {
		err := errors.New("PutRequest cannot have an empty Job ID")
		log.Error().Err(err)
		return nil, err
	}
	d, err := time.ParseDuration(delay.String())
	if err != nil {
		log.Error().Err(err).Msg("Could not parse job delay duration")
		return nil, err
	}

	// Add job to hub
	triggerAt := time.Now().Add(d)
	j := goyaad.NewJob(pr.GetJob().GetId(), triggerAt, pr.GetJob().GetBody())
	err = s.hub.AddJobLocked(j)
	if err != nil {
		return nil, err
	}
	return &yaadgrpc.PutResponse{}, nil
}

// Cancel a job with the given ID if it exists. This call is idempotent
func (s *Server) Cancel(ctx context.Context, cr *yaadgrpc.CancelRequest) (*yaadgrpc.CancelResponse, error) {
	var err error
	if cr.GetId() != "" {
		err = s.hub.CancelJobLocked(cr.GetId())
	}
	return &yaadgrpc.CancelResponse{}, err
}

// Next returns the next job that is ready to be consumed. If no job is ready, response is empty
func (s *Server) Next(ctx context.Context, nr *yaadgrpc.NextRequest) (*yaadgrpc.NextResponse, error) {
	log.Info().Msg("Got a next request for grpc server")
	job := s.hub.NextLocked()

	// TODO: Implement timeout wait loop

	if job == nil {
		return &yaadgrpc.NextResponse{}, nil
	}
	log.Info().
		Str("jobID", job.ID()).
		Bytes("jobBody", job.Body()).
		Msgf("Responding with job with id and body")
	return &yaadgrpc.NextResponse{
		Next: &yaadgrpc.NextResponse_Job{
			Job: &yaadgrpc.Job{Id: job.ID(),
				Delay: &duration.Duration{Seconds: int64(job.TriggerAt().Sub(time.Now()).Seconds())},
				Body:  job.Body()}}}, nil
}

// Inspect returns upto N jobs. Can return none
func (s *Server) Inspect(req *yaadgrpc.InspectRequest, srv yaadgrpc.Yaad_InspectServer) error {
	log.Debug().Int32("count", req.GetCount()).Msg("Returning jobs for inspection")
	jobs := s.hub.GetNJobs(int(req.GetCount()))

	for j := range jobs {
		grpcJob := &yaadgrpc.Job{
			Id:    j.ID(),
			Delay: &duration.Duration{Seconds: int64(j.TriggerAt().Sub(time.Now()).Seconds())},
			Body:  j.Body(),
		}
		srv.Send(&yaadgrpc.InspectResponse{
			Job: grpcJob,
		})
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
	defer cancel()

	// Register gRPC server endpoint
	// Note: Make sure the gRPC server is running properly and accessible
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}
	err := yaadgrpc.RegisterYaadHandlerFromEndpoint(ctx, mux, gAddr, opts)
	if err != nil {
		log.Fatal().Err(err).Msg("GRPC:HTTPGW: Failed to register grpc-http-gateway")
	}
	// Start HTTP server (and proxy calls to gRPC server endpoint)
	srv := &http.Server{
		Addr:    hAddr,
		Handler: mux,
	}
	go func() {
		log.Info().Str("Addr", srv.Addr).Msg("Starting GRPC-JSON http Server")
		if err = srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("GRPC:HTTPGW: Failed to serve grpc-http-gateway requests")
		}
	}()
	return srv, nil
}

// NewGRPCServer creates a new GRPC capable server backed by a yaad hub
func NewGRPCServer(hub *goyaad.Hub) *Server {
	return &Server{hub: hub, Server: grpc.NewServer()}
}

// ServeGRPC starts the GRPC listener and starts accepting connections
func (s *Server) ServeGRPC(addr string) (io.Closer, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	reflection.Register(s.Server)
	yaadgrpc.RegisterYaadServer(s.Server, s)
	go func() {
		log.Info().Str("Addr", l.Addr().String()).Msg("Starting GRPC Server")
		if err := s.Serve(l); err != nil {
			log.Fatal().Err(err).Msg("GRPC: Failed to serve requests")
		}
	}()
	return s, nil
}

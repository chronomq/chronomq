package grpc

//go:generate go install github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway
//go:generate go install github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger
//go:generate protoc -I. -I../../../ -I$GOPATH/bin --go_out=plugins=grpc:. --grpc-gateway_out=logtostderr=true:. --swagger_out=logtostderr=true:. service.proto
import (
	context "context"
	"io"
	"net"
	"net/http"
	"time"

	duration "github.com/golang/protobuf/ptypes/duration"
	empty "github.com/golang/protobuf/ptypes/empty"
	wrappers "github.com/golang/protobuf/ptypes/wrappers"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/urjitbhatia/goyaad/pkg/goyaad"
)

const (
	port = ":50051"
)

// Server is used to implement YaadServer
type Server struct {
	*grpc.Server
	hub *goyaad.Hub
}

var emptyMemoized = &empty.Empty{}
var emptyNextMemoized = &NextResponse{Next: &NextResponse_Empty{Empty: emptyMemoized}}

// PutWithID saves a new job. It can error if another job with the ID already exists in the system
func (s *Server) PutWithID(ctx context.Context, job *Job) (*empty.Empty, error) {
	log.Info().Msgf("Job : %v", job)
	d := time.Duration(job.Delay.Nanos)
	triggerAt := time.Now().Add(d)
	j := goyaad.NewJob(job.Id, triggerAt, job.GetBody())
	err := s.hub.AddJobLocked(j)
	log.Info().Msgf("Saved job with body: %v", j.Body())
	return emptyMemoized, err
}

// Cancel a job with the given ID if it exists. This call is idempotent
func (s *Server) Cancel(ctx context.Context, id *wrappers.StringValue) (*empty.Empty, error) {
	err := s.hub.CancelJobLocked(id.Value)
	return emptyMemoized, err
}

// Next returns the next job that is ready to be consumed. If no job is ready, response is empty
func (s *Server) Next(ctx context.Context, dur *duration.Duration) (*NextResponse, error) {
	log.Info().Msg("Got a next request for grpc server")
	job := s.hub.NextLocked()
	if job == nil {
		return emptyNextMemoized, nil
	}
	log.Info().Msgf("Responding with job with id:  %s and body: %v", job.ID(), job.Body())
	return &NextResponse{
		Next: &NextResponse_Job{
			Job: &Job{Id: job.ID(),
				Delay: &duration.Duration{Seconds: int64(job.TriggerAt().Sub(time.Now()).Seconds())},
				Body:  job.Body()}}}, nil
}

// InspectN returns upto N jobs. Can return none
func (s *Server) InspectN(req *InspectNRequest, srv Yaad_InspectNServer) error {
	log.Debug().Uint64("count", req.N).Msg("Returning jobs for inspection")
	jobs := s.hub.GetNJobs(int(req.N))

	for j := range jobs {
		grpcJob := &Job{
			Id:    j.ID(),
			Delay: &duration.Duration{Seconds: int64(j.TriggerAt().Sub(time.Now()).Seconds())},
			Body:  j.Body(),
		}
		srv.Send(grpcJob)
	}
	return nil
}

// Close the server listener (gracefully)
func (s *Server) Close() error {
	s.GracefulStop()
	return nil
}

// Serve initializes a new GRPC server and starts accepting connections
func Serve(hub *goyaad.Hub, gaddr string) (io.Closer, error) {
	l, err := net.Listen("tcp", gaddr)
	if err != nil {
		return nil, err
	}
	grpcSrv := grpc.NewServer()
	yaadGRPCServer := &Server{hub: hub, Server: grpcSrv}
	reflection.Register(grpcSrv)
	RegisterYaadServer(grpcSrv, yaadGRPCServer)
	go func() {
		log.Info().Str("Addr", l.Addr().String()).Msg("Starting GRPC Server")
		if err := grpcSrv.Serve(l); err != nil {
			log.Fatal().Err(err).Msg("GRPC: Failed to serve requests")
		}
	}()

	go func() {
		ctx := context.Background()
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		// Register gRPC server endpoint
		// Note: Make sure the gRPC server is running properly and accessible
		mux := runtime.NewServeMux()
		opts := []grpc.DialOption{grpc.WithInsecure()}
		err := RegisterYaadHandlerFromEndpoint(ctx, mux, gaddr, opts)
		if err != nil {
			log.Fatal().Err(err).Msg("GRPC:HTTPGW: Failed to register grpc-http-gateway")
		}
		// Start HTTP server (and proxy calls to gRPC server endpoint)
		err = http.ListenAndServe(":8081", mux)
		if err != nil {
			log.Fatal().Err(err).Msg("GRPC:HTTPGW: Failed to serve grpc-http-gateway requests")
		}
	}()

	return yaadGRPCServer, nil
}

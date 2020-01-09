package cmd

import (
	"io"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/urjitbhatia/goyaad/pkg/goyaad"
	"github.com/urjitbhatia/goyaad/pkg/metrics"
	"github.com/urjitbhatia/goyaad/pkg/persistence"
	"github.com/urjitbhatia/goyaad/pkg/protocol"
)

var logLevel = "INFO"
var friendlyLog bool

// Server running the Yaad App
type Server struct {
	RPCAddr   string // RPC Listener Addr
	GRPCAddr  string // GRPC Listener Addr
	StatsAddr string // StatsD Server Addr

	StoreCfg persistence.StoreConfig // Persistence Storage config

	Restore   bool          // If true, hub will attempt restore on startup
	SpokeSpan time.Duration // Spoke duration
}

var server = Server{
	RPCAddr:  ":11301",
	GRPCAddr: ":9999", StatsAddr: ":8125"}

func (s *Server) runCobraCommand(cmd *cobra.Command, args []string) {
	// Check for store args
	switch server.StoreCfg.Store {
	case persistence.FileStore, persistence.S3Store:
	default:
		log.Fatal().Msg("Unknown persistence store option specified")
	}
	log.Info().Int("PID", os.Getpid()).Msg("Starting Yaad")
	metrics.InitMetrics(server.StatsAddr)
	server.run()
	log.Info().Msg("Shutdown ok")
}

func setLogLevel() {
	lvl, err := zerolog.ParseLevel(strings.ToLower(logLevel))
	if err != nil {
		log.Fatal().Str("LevelStr", logLevel).Msg("Invalid log-level provided")
	}
	zerolog.SetGlobalLevel(lvl)
	if friendlyLog {
		log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.Kitchen}).
			With().
			Timestamp().
			Logger()
	}
}

func (s *Server) run() {
	// More Aggressive GC
	if os.Getenv("GOGC") == "" {
		log.Info().Msg("Applying default GC tuning")
		debug.SetGCPercent(5)
	} else {
		log.Info().Str("GCPercent", os.Getenv("GOGC")).Msg("Using custom GC tuning")
	}
	go func() {
		err := http.ListenAndServe(":6060", nil)
		if err != nil {
			log.Error().Err(err).Msg("pprof server has stopped")
		}
	}()

	log.Info().Msg("Starting Goyaad")

	storage, err := server.StoreCfg.Storage()
	if err != nil {
		log.Fatal().Err(err).Msg("Cannot initialize storage")
	}

	opts := &goyaad.HubOpts{
		AttemptRestore: s.Restore,
		SpokeSpan:      s.SpokeSpan,
		Persister:      persistence.NewJournalPersister(storage),
		MaxCFSize:      goyaad.DefaultMaxCFSize,
	}

	hub := goyaad.NewHub(opts)
	var rpcSRV io.Closer
	wg := sync.WaitGroup{}
	go func() {
		rpcSRV, _ = protocol.ServeRPC(hub, s.RPCAddr)
	}()

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGUSR1, os.Interrupt)

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-sigc
		log.Info().Msg("Stopping rpc protocol server")
		rpcSRV.Close()
		log.Info().Msg("Stopping rpc protocol server - Done")
		hub.Stop(true)
	}()

	wg.Wait()
}

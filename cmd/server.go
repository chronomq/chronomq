package cmd

import (
	"io"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/urjitbhatia/goyaad/pkg/goyaad"
	"github.com/urjitbhatia/goyaad/pkg/persistence"
	"github.com/urjitbhatia/goyaad/pkg/protocol"
)

var (
	defaultAddrs = &addrs{
		rpcAddr:   ":11301",
		grpcAddr:  ":9999",
		statsAddr: ":8125",
	}
	defaultServer = &server{
		addrs: defaultAddrs,
	}
	serverCmd = &cobra.Command{
		Use:   "server",
		Short: "Run the Yaad server",
	}
)

func init() {
	serverCmd.PersistentFlags().DurationVarP(&defaultServer.spokeSpan, "spokeSpan", "S", time.Second*10, "Spoke span (golang duration string format)")
	serverCmd.PersistentFlags().BoolVarP(&defaultServer.restore, "restore", "r", false, "Restore existing data if possible from store")

	rootCmd.AddCommand(serverCmd)
}

type addrs struct {
	rpcAddr   string // RPC Listener Addr
	grpcAddr  string // GRPC Listener Addr
	statsAddr string // StatsD listener Addr
}

// Server running Yaad
type server struct {
	addrs *addrs

	storeCfg persistence.StoreConfig // Persistence Storage config

	restore   bool          // If true, hub will attempt restore on startup
	spokeSpan time.Duration // Spoke duration
}

func run(s *server) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		// Check for store args
		switch s.storeCfg.Store {
		case persistence.FileStore, persistence.S3Store:
		default:
			log.Fatal().Msg("Unknown persistence store option specified. Aborting")
		}
		log.Info().Int("PID", os.Getpid()).Msg("Starting Server")
		s.serve()
		log.Info().Msg("Shutdown ok")
	}
}

func (s *server) serve() {
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

	storage, err := s.storeCfg.Storage()
	if err != nil {
		log.Fatal().Err(err).Msg("Cannot initialize storage")
	}

	opts := &goyaad.HubOpts{
		AttemptRestore: s.restore,
		SpokeSpan:      s.spokeSpan,
		Persister:      persistence.NewJournalPersister(storage),
		MaxCFSize:      goyaad.DefaultMaxCFSize,
	}

	hub := goyaad.NewHub(opts)
	var rpcSRV io.Closer
	wg := sync.WaitGroup{}
	go func() {
		rpcSRV, _ = protocol.ServeRPC(hub, s.addrs.rpcAddr)
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

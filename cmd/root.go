// Package cmd contains the entrypoints to the binary
package cmd

import (
	"io"
	"os"
	"os/signal"
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
var raddr = ":11301"
var gaddr = ":9999"
var statsAddr = ":8125"
var dataDir string
var restore bool
var spokeSpan string

func init() {
	// Global persistent flags
	rootCmd.PersistentFlags().StringVarP(&logLevel, "log-level", "", "INFO", "Set log level: INFO, DEBUG")
	rootCmd.PersistentFlags().StringVar(&raddr, "raddr", raddr, "Set RPC server listen addr (host:port)")
	rootCmd.PersistentFlags().StringVar(&gaddr, "gaddr", gaddr, "Set GRPC server listen addr (host:port)")

	rootCmd.PersistentFlags().StringVarP(&statsAddr, "statsAddr", "s", statsAddr, "Stats addr (host:port)")
	rootCmd.PersistentFlags().StringVarP(&spokeSpan, "spokeSpan", "S", "10s", "Spoke span (golang duration string format)")

	dataDir, _ = os.Getwd()
	rootCmd.Flags().StringVarP(&dataDir, "dataDir", "d", dataDir, `Data dir location - persits state here when SIGUSR1 is received. 
	Restores from this location at start if journal files are present.`)
	rootCmd.Flags().BoolVarP(&restore, "restore", "r", false, "Restore existing data if possible (from dataDir)")
}

var rootCmd = &cobra.Command{
	Short: "goyaad",
	Long: `For advanced gc tuning:
	Set GOGC levels using the env variable.
	See: https://golang.org/pkg/runtime/#hdr-Environment_Variables`,
	Run: func(cmd *cobra.Command, args []string) {
		setLogLevel()
		metrics.InitMetrics(statsAddr)
		runServer()
	},
}

func runServer() {
	log.Info().Msg("Starting Goyaad")
	ss, err := time.ParseDuration(spokeSpan)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	opts := &goyaad.HubOpts{
		AttemptRestore: restore,
		SpokeSpan:      ss,
		Persister:      persistence.NewJournalPersister(dataDir)}

	hub := goyaad.NewHub(opts)
	var rpcSRV io.Closer
	wg := sync.WaitGroup{}
	go func() {
		rpcSRV, _ = protocol.ServeRPC(hub, raddr)
	}()

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGUSR1)

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

// Execute root cmd by default
func Execute() {
	log.Info().Msgf("Running as pid: %d", os.Getpid())
	if err := rootCmd.Execute(); err != nil {
		log.Error().Err(err).Send()
		os.Exit(1)
	}
	log.Info().Msg("Shutdown ok")
}

func setLogLevel() {
	lvl, err := zerolog.ParseLevel(strings.ToLower(logLevel))
	if err != nil {
		log.Fatal().Str("LevelStr", logLevel).Msg("Invalid log-level provided")
	}
	zerolog.SetGlobalLevel(lvl)
}

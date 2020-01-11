// Package cmd contains the entrypoints to the binary
package cmd

import (
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/urjitbhatia/goyaad/pkg/metrics"
)

var (
	logLevel    = "INFO"
	friendlyLog bool
	rootCmd     = &cobra.Command{
		Long: `For advanced gc tuning: Set GOGC levels using the env variable.
		See: https://golang.org/pkg/runtime/#hdr-Environment_Variables`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			setLogLevel()
			metrics.InitMetrics(defaultAddrs.statsAddr)
		},
	}
)

func init() {
	// Global persistent flags
	rootCmd.PersistentFlags().StringVarP(&logLevel, "log-level", "l", "INFO", "Set log level: INFO, DEBUG")
	rootCmd.PersistentFlags().BoolVarP(&friendlyLog, "friendly-log", "L", false, "Use a human-friendly logging style")

	rootCmd.PersistentFlags().StringVar(&defaultAddrs.rpcAddr, "raddr", defaultAddrs.rpcAddr, "Bind RPC listener to (host:port)")
	rootCmd.PersistentFlags().StringVar(&defaultAddrs.grpcAddr, "gaddr", defaultAddrs.grpcAddr, "Bind GRPC listener to (host:port)")
	rootCmd.PersistentFlags().StringVar(&defaultAddrs.statsAddr, "statsAddr", defaultAddrs.statsAddr, "Remote StatsD listener (host:port)")
}

// addrs - holds common net address configurations
type addrs struct {
	rpcAddr   string // RPC Listener Addr
	grpcAddr  string // GRPC Listener Addr
	statsAddr string // StatsD listener Addr
}

// Execute root cmd by default
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Error().Err(err).Send()
		os.Exit(1)
	}
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

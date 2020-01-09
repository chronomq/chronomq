// Package cmd contains the entrypoints to the binary
package cmd

import (
	"os"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Long: `For advanced gc tuning: Set GOGC levels using the env variable.
	See: https://golang.org/pkg/runtime/#hdr-Environment_Variables`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		setLogLevel()
	},
	Run: func(cmd *cobra.Command, args []string) {
		// Run default wiring
		filestoreCmd.PreRun(cmd, args)
		filestoreCmd.Run(cmd, args)
	},
}

func init() {
	// Global persistent flags
	rootCmd.PersistentFlags().StringVarP(&logLevel, "log-level", "l", "INFO", "Set log level: INFO, DEBUG")
	rootCmd.PersistentFlags().BoolVarP(&friendlyLog, "friendly-log", "L", false, "Use a human-friendly logging style")
	rootCmd.PersistentFlags().StringVar(&server.RPCAddr, "raddr", server.RPCAddr, "Set RPC server listen addr (host:port)")
	rootCmd.PersistentFlags().StringVar(&server.GRPCAddr, "gaddr", server.GRPCAddr, "Set GRPC server listen addr (host:port)")
	rootCmd.PersistentFlags().StringVarP(&server.StatsAddr, "statsAddr", "s", server.StatsAddr, "Stats addr (host:port)")

	rootCmd.Flags().DurationVarP(&server.SpokeSpan, "spokeSpan", "S", time.Second*10, "Spoke span (golang duration string format)")
	rootCmd.Flags().BoolVarP(&server.Restore, "restore", "r", false, "Restore existing data if possible (from persistence store)")
}

// Execute root cmd by default
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Error().Err(err).Send()
		os.Exit(1)
	}
}

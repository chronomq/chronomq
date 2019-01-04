package cmd

import (
	"log"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/urjitbhatia/goyaad/pkg/goyaad"
	"github.com/urjitbhatia/goyaad/pkg/metrics"
	"github.com/urjitbhatia/goyaad/pkg/persistence"
	"github.com/urjitbhatia/goyaad/pkg/protocol"
)

var logLevel = "INFO"
var addr = ":11300"
var statsAddr = ":8125"
var dataDir string
var restore bool
var spokeSpan string

func init() {
	// Global persistent flags
	rootCmd.PersistentFlags().StringVarP(&logLevel, "log-level", "", "INFO", "Set log level: INFO, DEBUG")
	rootCmd.PersistentFlags().StringVarP(&addr, "addr", "a", ":11300", "Set listen addr (host:port)")
	rootCmd.PersistentFlags().StringVarP(&statsAddr, "statsAddr", "s", ":8125", "Stats addr (host:port)")
	rootCmd.PersistentFlags().StringVarP(&spokeSpan, "spokeSpan", "S", "10s", "Spoke span (golang duration string format)")

	dataDir, _ = os.Getwd()
	rootCmd.Flags().StringVarP(&dataDir, "dataDir", "d", dataDir, `Data dir location - persits state here when SIGUSR1 is received. 
	Restores from this location at start if journal files are present.`)
	rootCmd.Flags().BoolVarP(&restore, "restore", "r", false, "Restore existing data if possible (from dataDir)")
}

var rootCmd = &cobra.Command{
	Short: "goyaad",
	Run: func(cmd *cobra.Command, args []string) {
		setLogLevel()
		runServer()
	},
}

func runServer() {
	logrus.Info("Starting Goyaad")
	metrics.InitMetrics(statsAddr)
	ss, err := time.ParseDuration(spokeSpan)
	if err != nil {
		log.Fatal(err)
	}
	opts := &goyaad.HubOpts{
		AttemptRestore: restore,
		SpokeSpan:      ss,
		Persister:      persistence.NewJournalPersister(dataDir)}
	s := protocol.NewYaadServer(false, opts)
	log.Fatalf("SHUTDOWN. Error: %v", s.ListenAndServe("tcp", addr))
}

// Execute root cmd by default
func Execute() {
	logrus.Infof("Runnig as pid: %d", os.Getpid())
	if err := rootCmd.Execute(); err != nil {
		logrus.Error(err)
		os.Exit(1)
	}
}

func setLogLevel() {
	lvl, err := logrus.ParseLevel(logLevel)
	if err != nil {
		logrus.Fatal("Invalid log-level provided: ", logLevel)
	}
	logrus.SetLevel(lvl)
}

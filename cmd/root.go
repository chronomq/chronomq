package cmd

import (
	"io"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/urjitbhatia/goyaad/pkg/goyaad"
	"github.com/urjitbhatia/goyaad/pkg/metrics"
	"github.com/urjitbhatia/goyaad/pkg/persistence"
	"github.com/urjitbhatia/goyaad/pkg/protocol"
)

var logLevel = "INFO"
var baddr = ":11300"
var raddr = ":11301"
var statsAddr = ":8125"
var dataDir string
var restore bool
var spokeSpan string
var rpc bool

func init() {
	// Global persistent flags
	rootCmd.PersistentFlags().StringVarP(&logLevel, "log-level", "", "INFO", "Set log level: INFO, DEBUG")
	rootCmd.PersistentFlags().StringVar(&raddr, "raddr", raddr, "Set RPC server listen addr (host:port)")
	rootCmd.PersistentFlags().StringVar(&baddr, "baddr", baddr, "Set Beanstalkd server listen addr (host:port)")

	rootCmd.PersistentFlags().StringVarP(&statsAddr, "statsAddr", "s", statsAddr, "Stats addr (host:port)")
	rootCmd.PersistentFlags().StringVarP(&spokeSpan, "spokeSpan", "S", "10s", "Spoke span (golang duration string format)")

	rootCmd.PersistentFlags().BoolVarP(&rpc, "rpc", "R", false, "Expose an rpc server")

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

	hub := goyaad.NewHub(opts)
	var rpcSRV io.Closer
	var beanSRV io.Closer
	wg := sync.WaitGroup{}
	if rpc {
		go func() {
			rpcSRV, _ = protocol.ServeRPC(hub, raddr)
		}()
	}
	go func() {
		beanSRV = protocol.ServeBeanstalkd(hub, baddr)
	}()

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGUSR1)

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-sigc
		logrus.Info("Stopping bean protocol server")
		beanSRV.Close()
		logrus.Info("Stopping bean protocol server - Done")
		if rpc {
			logrus.Info("Stopping rpc protocol server")
			rpcSRV.Close()
			logrus.Info("Stopping rpc protocol server - Done")
		}
		hub.Stop(true)
	}()

	wg.Wait()
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

package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/urjitbhatia/goyaad/pkg/goyaad"
	"github.com/urjitbhatia/goyaad/pkg/protocol"
)

var logLevel = "INFO"
var addr = ":11300"
var statsAddr = ":8125"

func init() {
	rootCmd.Flags().StringVarP(&logLevel, "log-level", "", "INFO", "Set log level: INFO, DEBUG")

	// Global persistent flags
	rootCmd.PersistentFlags().StringVarP(&addr, "addr", "a", ":11300", "Set listen addr (host:port)")
	rootCmd.PersistentFlags().StringVarP(&statsAddr, "statsAddr", "s", ":8125", "Stats addr (host:port)")
}

var rootCmd = &cobra.Command{
	Short: "goyaad",
	Run: func(cmd *cobra.Command, args []string) {
		runServer()
	},
}

func runServer() {
	logrus.Info("Starting Goyaad")
	lvl, err := logrus.ParseLevel(logLevel)
	if err != nil {
		logrus.Fatal("Invalid log-level provided: ", logLevel)
	}
	logrus.SetLevel(lvl)
	goyaad.InitMetrics(statsAddr)

	s := protocol.NewYaadServer(false)
	log.Fatal(s.ListenAndServe("tcp", addr))
}

// Execute root cmd by default
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

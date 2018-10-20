package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/urjitbhatia/goyaad/pkg/protocol"
)

var rootCmd = &cobra.Command{
	Short: "goyaad",
	Run: func(cmd *cobra.Command, args []string) {
		runServer()
	},
}

func runServer() {
	logrus.Info("I AM GROOT!")
	// logrus.SetLevel(logrus.DebugLevel)

	s := protocol.NewYaadServer()

	log.Fatal(s.ListenAndServe("tcp", ":11300"))
}

// Execute root cmd by default
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

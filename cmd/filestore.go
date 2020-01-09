package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/urjitbhatia/goyaad/pkg/persistence"
)

// filestoreCmd represents the filestore command
var filestoreCmd = &cobra.Command{
	Use:   "filestore",
	Short: "Run server with a filestore",
	Long: `Persists jobs state in this s3 dir on filesystem when SIGUSR1 is received.
	Restores from this location at start if journal files are present (and restore flag is set).`,
	PreRun: func(cmd *cobra.Command, args []string) {
		defaultServer.storeCfg.Store = persistence.FileStore
	},
	Run: run(defaultServer), // defers actual run to server with the correct PreRun set
}

func init() {
	dataDir, _ := os.Getwd()
	filestoreCmd.Flags().StringVarP(&defaultServer.storeCfg.FSCfg.BaseDir, "datadir", "d", dataDir, `Store directory path. Default PWD`)

	serverCmd.AddCommand(filestoreCmd)
}

package cmd

import (
	"github.com/spf13/cobra"

	"github.com/urjitbhatia/goyaad/pkg/persistence"
)

var s3storeCmd = &cobra.Command{
	Use:   "s3store",
	Short: "Run server with a s3Filestore",
	Long: `Persists jobs state in this s3 bucket/prefix when SIGUSR1 is received.
Restores from this location at start if journal files are present (and restore flag is set).`,
	PreRun: func(cmd *cobra.Command, args []string) {
		defaultServer.storeCfg.Store = persistence.S3Store
	},
	Run: run(defaultServer), // defer to server run with the right PreRun set to configure store
}

func init() {
	s3storeCmd.Flags().StringVar(&defaultServer.storeCfg.S3Cfg.Bucket, "bucket", "", `S3 Bucket Name`)
	s3storeCmd.Flags().StringVar(&defaultServer.storeCfg.S3Cfg.Prefix, "prefix", "", `S3 Path prefix`)

	s3storeCmd.MarkFlagRequired("bucket")

	serverCmd.AddCommand(s3storeCmd)
}

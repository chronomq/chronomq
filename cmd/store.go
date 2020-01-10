package cmd

import (
	"errors"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var storeCmd = &cobra.Command{
	Use:   "store",
	Short: "Run server with a bucket data store",
	Long: `Persists jobs state in this s3 bucket/prefix when SIGUSR1 is received.
Restores from this location at start if journal files are present (and restore flag is set).
Supported s3-compatible storage backends: s3, gs, azblob and others: https://gocloud.dev/howto/blob/#s3-compatible`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		u, err := url.Parse(storeURL)
		if err != nil {
			return err
		}
		switch u.Scheme {
		case "s3", "gs", "azblob", "file":
			// supported
		case "":
			// assume file
			u.Scheme = "file"
		default:
			return errors.New("Bucket scheme not supported")
		}
		q := u.Query()
		if !strings.HasSuffix(storePrefix, "/") {
			storePrefix = storePrefix + "/"
		}
		q.Add("prefix", storePrefix)
		u.RawQuery = q.Encode()
		defaultServer.storeCfg.Bucket = u
		return nil
	},
	Run: run(defaultServer), // defer to server run with the right PreRun set to configure store
}

var storeURL = ""
var storePrefix = ""

func init() {
	dataDir, _ := os.Getwd()
	storeCmd.Flags().StringVar(&storeURL, "store-url", dataDir, `Filesystem dir or S3-style url.
Examples: filesystemdir/subdir {file|s3|gs|azblob}://bucket`)
	storeCmd.Flags().StringVar(&storePrefix, "store-prefix", "", `Store path prefix`)

	serverCmd.AddCommand(storeCmd)
}

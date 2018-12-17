package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var buildInfo = struct {
	version string
	date    string
	commit  string
}{
	"dev",
	time.Now().Format(time.RFC3339Nano),
	"none",
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

// SetBuildInfo initializes the build info used by the version command
func SetBuildInfo(version, date, commit string) {
	buildInfo.version = version
	buildInfo.commit = commit
	buildInfo.date = date
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Prints full build info of GoYaad",
	Long:  `All software has versions. This is GoYaad's`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Version: %s\nCommit: %s\nBuild Date: %s\n",
			buildInfo.version,
			buildInfo.commit,
			buildInfo.date)
	},
}

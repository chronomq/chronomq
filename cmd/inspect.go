package cmd

import (
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/urjitbhatia/goyaad/pkg/protocol"
)

const delimiter = "-----------------------------------------"

var (
	num        int
	outfile    string
	inspectCmd = &cobra.Command{
		Use:   "inspect",
		Short: "Fetches upto num jobs from the server without consuming them",
		Long: `Use this commmand carefully. If num is too large, it might cause the server
		to slow down during the inspect operation as well as place memory pressure. For large
		num, it will also put mem pressure on the client call`,
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Info().Msg("inspecting")
			return inspect()
		},
	}
)

func init() {
	inspectCmd.Flags().IntVarP(&num, "num", "n", 1, "Max Number of jobs to inspect")
	inspectCmd.Flags().StringVarP(&outfile, "out", "o", "", "Write output to outfile (default: stdout)")

	inspectCmd.Flags().StringVar(&defaultAddrs.rpcAddr, "raddr", defaultAddrs.rpcAddr, "Set RPC server addr (host:port)")
	rootCmd.AddCommand(inspectCmd)
}

func inspect() error {
	client := &protocol.RPCClient{}
	log.Info().Str("address", defaultAddrs.rpcAddr).Msg("Connecting to server")
	// This ensures all contexts get a running server
	err := client.Connect(defaultAddrs.rpcAddr)
	if err != nil {
		return err
	}
	output := os.Stdout
	if outfile != "" {
		log.Warn().Str("outfile", outfile).Msg("Writing to file")
		output, err = os.OpenFile(outfile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0777)
		if err != nil {
			return err
		}
	}

	jobs := []*protocol.RPCJob{}
	err = client.InspectN(num, &jobs)
	if err != nil {
		return err
	}

	for _, j := range jobs {
		_, err = output.WriteString(fmt.Sprintf(`
%s
ID:	%s
DelayFromNow:	%s
Body:
%s`, delimiter, j.ID, j.Delay, string(j.Body)))
		if err != nil {
			return err
		}
	}
	return nil
}

package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/urjitbhatia/goyaad/pkg/protocol"
)

type putArgs struct {
	id      string
	payload string
	delay   time.Duration
}

type nextArgs struct {
	timeout time.Duration
	json    bool
}
type nextJSON struct {
	ID   string
	Body string
}

var (
	putCmdArgs = putArgs{}
	putCmd     = &cobra.Command{
		Use:   "put",
		Short: "Enqueue a job",
		Long: `Enqueues a job. Errors if another job with the same id already exists.
		If no id is specific, the command will auto-generate a random id and return it.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := &protocol.RPCClient{}
			err := client.Connect(defaultAddrs.rpcAddr)
			if err != nil {
				return err
			}
			if putCmdArgs.id != "" {
				err = client.PutWithID(putCmdArgs.id, []byte(putCmdArgs.payload), putCmdArgs.delay)
			} else {
				var id string
				id, err = client.Put([]byte(putCmdArgs.payload), putCmdArgs.delay)
				putCmdArgs.id = id
			}
			if err == nil {
				fmt.Println(putCmdArgs.id)
			}
			return err
		},
		SilenceUsage: true,
	}
	nextCmdArgs = nextArgs{}
	nextCmd     = &cobra.Command{
		Use:   "next",
		Short: "Get next ready job",
		Long:  `Gets the next job that is ready to trigger`,
		Run: func(cmd *cobra.Command, args []string) {
			client := &protocol.RPCClient{}
			err := client.Connect(defaultAddrs.rpcAddr)
			if err != nil {
				log.Error().Err(err).Send()
				return
			}
			id, body, err := client.Next(nextCmdArgs.timeout)
			if err != nil {
				log.Error().Err(err).Send()
				return
			}
			switch nextCmdArgs.json {
			case true:
				var j []byte
				j, err = json.Marshal(nextJSON{ID: id, Body: string(body)})
				if err != nil {
					log.Error().Err(err).Send()
					return
				}
				fmt.Printf("%s", j)
			case false:
				fmt.Printf("ID: %s\nBODY: %s", id, body)
			}
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}
)

func init() {
	putCmd.PersistentFlags().StringVarP(&putCmdArgs.id, "id", "i", "", "ID for the job")
	putCmd.PersistentFlags().DurationVarP(&putCmdArgs.delay, "delay", "d", 0, "Job trigger delay relative to now (golang duration string format)")
	putCmd.PersistentFlags().StringVarP(&putCmdArgs.payload, "body", "b", "", "Job body. Defaults to reading stdin if not specified")

	nextCmd.PersistentFlags().DurationVarP(&nextCmdArgs.timeout, "timeout", "t", 0, "Wait at most timeout duration for a job to be available")
	nextCmd.PersistentFlags().BoolVarP(&nextCmdArgs.json, "json", "j", false, "Print job response in json format")
	rootCmd.AddCommand(putCmd)
	rootCmd.AddCommand(nextCmd)
}

package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/chronomq/chronomq/api/rpc/chronomq"
)

type putArgs struct {
	id      string
	payload *bufValue
	delay   time.Duration
}

type bufValue struct {
	buf   []byte
	isSet bool
}

func (b *bufValue) String() string { return string(b.buf) }
func (b *bufValue) Set(s string) error {
	b.buf = []byte(s)
	b.isSet = true
	return nil
}
func (b *bufValue) Type() string {
	return "bufValue"
}

var (
	putCmdArgs = putArgs{
		payload: &bufValue{},
	}
	putCmd = &cobra.Command{
		Use:   "put",
		Short: "Enqueue a job",
		Long: `Enqueues a job. Errors if another job with the same id already exists.
		If no id is specific, the command will auto-generate a random id and return it.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := chronomq.NewClient(defaultAddrs.rpcAddr)
			if err != nil {
				return err
			}

			var payload []byte
			if putCmdArgs.payload.isSet {
				payload = putCmdArgs.payload.buf
			} else {
				// read stdin
				reader := bufio.NewReader(os.Stdin)
				payload, err = ioutil.ReadAll(reader)
				if err != nil {
					return err
				}
			}

			if putCmdArgs.id != "" {
				err = client.PutWithID(putCmdArgs.id, payload, putCmdArgs.delay)
			} else {
				var id string
				id, err = client.Put(payload, putCmdArgs.delay)
				putCmdArgs.id = id
			}
			if err == nil {
				fmt.Println(putCmdArgs.id)
			}
			return err
		},
		SilenceErrors: true,
		SilenceUsage:  true,
	}
)

type nextArgs struct {
	timeout time.Duration
	json    bool
}
type nextJSON struct {
	ID   string
	Body string
}

var (
	nextCmdArgs = nextArgs{}
	nextCmd     = &cobra.Command{
		Use:   "next",
		Short: "Get next ready job",
		Long:  `Gets the next job that is ready to trigger`,
		Run: func(cmd *cobra.Command, args []string) {
			client, err := chronomq.NewClient(defaultAddrs.rpcAddr)
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

type cancelArgs struct {
	id string
}

var (
	cancelCmdArgs = cancelArgs{}
	cancelCmd     = &cobra.Command{
		Use:   "cancel",
		Short: "Cancel a job",
		Long:  `Cancels the job with the given id. Cancel is ignore is no job matches the id`,
		Run: func(cmd *cobra.Command, args []string) {
			client, err := chronomq.NewClient(defaultAddrs.rpcAddr)
			if err != nil {
				log.Error().Err(err).Send()
				return
			}
			err = client.Cancel(cancelCmdArgs.id)
			if err != nil {
				log.Error().Err(err).Send()
				return
			}
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}
)

func init() {
	putCmd.PersistentFlags().StringVarP(&putCmdArgs.id, "id", "i", "", "ID for the job")
	putCmd.PersistentFlags().DurationVarP(&putCmdArgs.delay, "delay", "d", 0, "Job trigger delay relative to now (golang duration string format)")
	putCmd.PersistentFlags().VarP(putCmdArgs.payload, "body", "b", "Job body. Defaults to reading stdin if not specified")

	nextCmd.PersistentFlags().DurationVarP(&nextCmdArgs.timeout, "timeout", "t", 0, "Wait at most timeout duration for a job to be available")
	nextCmd.PersistentFlags().BoolVarP(&nextCmdArgs.json, "json", "j", false, "Print job response in json format")

	cancelCmd.PersistentFlags().StringVarP(&cancelCmdArgs.id, "id", "i", "", "ID for the job")
	cancelCmd.MarkPersistentFlagRequired("id")

	rootCmd.AddCommand(putCmd)
	rootCmd.AddCommand(nextCmd)
	rootCmd.AddCommand(cancelCmd)
}

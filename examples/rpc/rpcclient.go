package main

import (
	"os"
	"time"

	"github.com/chronomq/chronomq/api/rpc/chronomq"
	"github.com/chronomq/chronomq/cmd"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.Kitchen}).
		With().
		Timestamp().
		Logger().Level(zerolog.InfoLevel)

	srvAddr := "localhost:11301" // default rpc server addr

	// Run a demo server
	os.Args = append(os.Args, []string{"server", "--raddr", srvAddr}...)
	go cmd.Execute()

	getClient := func() (*chronomq.Client, error) {
		var err error
		var client *chronomq.Client
		for i := 0; i < 10; i++ {
			client, err = chronomq.NewClient(srvAddr)
			if err == nil {
				return client, err
			}
			time.Sleep(time.Millisecond * 500)
		}
		return nil, err
	}

	client, err := getClient()
	if err != nil {
		log.Panic().Err(err).Send()
	}

	// Ping for connectivity checks
	if err := client.Ping(); err != nil {
		log.Panic().Err(err).Send()
	}

	// Put a job
	if err := client.PutWithID("job1", []byte{'h', 'i'}, time.Millisecond*100); err != nil {
		log.Panic().Err(err).Send()
	}

	// Fetch a job
	if id, data, err := client.Next(time.Second); err != nil {
		log.Panic().Err(err).Send()
	} else {
		log.Info().Msgf("Next job ready with ID: %s and body: %s", id, data)
	}
}

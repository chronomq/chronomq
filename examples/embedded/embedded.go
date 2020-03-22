package main

import (
	"os"
	"time"

	"github.com/chronomq/chronomq/pkg/chronomq"
	"github.com/chronomq/chronomq/pkg/persistence"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.Kitchen}).
		With().
		Timestamp().
		Logger().Level(zerolog.InfoLevel)

	memStore, err := persistence.InMemStorage()
	if err != nil {
		log.Panic().Err(err).Send()
	}
	opts := chronomq.HubOpts{
		Persister:      persistence.NewJournalPersister(memStore),
		AttemptRestore: false,
		SpokeSpan:      time.Second,
	}
	hub := chronomq.NewHub(&opts)

	job := chronomq.NewJob("jobID1", time.Now().Add(time.Millisecond*15), []byte("Hello World!"))
	err = hub.AddJobLocked(job)
	if err != nil {
		log.Panic().Err(err).Send()
	}
	log.Info().Msgf("Put job. ID: %s Data: %s TriggerAt: %s", job.ID(), job.Body(), job.TriggerAt())

	job = chronomq.NewJob("jobID2", time.Now().Add(time.Millisecond*5), []byte("Hi, I'm Chronomq, I am a time traveler."))
	err = hub.AddJobLocked(job)
	if err != nil {
		log.Panic().Err(err).Send()
	}
	log.Info().Msgf("Put job. ID: %s Data: %s TriggerAt: %s", job.ID(), job.Body(), job.TriggerAt())

	getJob := func() *chronomq.Job {
		var job *chronomq.Job
		for {
			job = hub.NextLocked()
			if job != nil {
				return job
			}
			time.Sleep(time.Millisecond * 5)
		}
	}

	job = getJob()
	log.Info().Msgf("Got next job. ID: %s Data: %s", job.ID(), job.Body())

	job = getJob()
	log.Info().Msgf("Got next job. ID: %s Data: %s", job.ID(), job.Body())
}

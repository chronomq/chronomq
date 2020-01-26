package main

import (
	"fmt"
	"syscall/js"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/urjitbhatia/goyaad/pkg/hub"
	"github.com/urjitbhatia/goyaad/pkg/job"
	"github.com/urjitbhatia/goyaad/pkg/persistence"
)

func RegisterFunction(funcName string, myfunc func(val js.Value, args []js.Value) interface{}) {
	js.Global().Set(funcName, js.FuncOf(myfunc))
}

var h *hub.Hub

func addJob(val js.Value, args []js.Value) (retval interface{}) {
	log.Printf("Adding job with value: %v args: %v", val, args)
	for _, i := range args {
		id := i.Get("id").String()
		delay, err := time.ParseDuration(i.Get("delay").String())
		if err != nil {
			log.Printf("Error parsing js job: %v", err)
		}
		body := i.Get("body").String()
		j := job.NewJob(id, time.Now().Add(delay), []byte(body))
		h.AddJobLocked(j)
	}
	return
}

func nextJob(val js.Value, args []js.Value) (retval interface{}) {
	callback := args[len(args)-1:][0]
	log.Info().Msgf("Fetching job with value: %v args: %v", val, args)
	if j := h.NextLocked(); j != nil {
		retval = string(j.Body())
		log.Info().Msg("found ready job in first call")
		return
	}
	sleep, err := time.ParseDuration(fmt.Sprintf("%ss", args[0].Get("maxWaitSec").String()))
	if err != nil {
		sleep = time.Second * 2
	}
	go func() {
		log.Info().Msg("Waiting for a job to be ready")
		time.Sleep(sleep)
		if j := h.NextLocked(); j != nil {
			body := string(j.Body())
			log.Info().Msgf("Found ready job after sleep: %s", body)
			callback.Invoke(body)
			return
		}
		callback.Invoke(js.Null())
	}()
	return
}

func nextJobPromise(val js.Value, args []js.Value) (retval interface{}) {
	cb := args[len(args)-1:][0]
	timeout, err := time.ParseDuration(fmt.Sprintf("%ss", args[0].Get("maxWaitSec").String()))
	if err != nil {
		timeout = time.Second * 2
	}
	go func() {
		if j := h.NextLocked(); j != nil {
			log.Info().Msg("returning job")
			cb.Invoke(js.Null(), string(j.Body()))
			return
		}
		waitTill := time.Now().Add(timeout)
		log.Info().Msg("sleeping....")
		for waitTill.After(time.Now()) {
			if j := h.NextLocked(); j != nil {
				cb.Invoke(js.Null(), string(j.Body()))
				return
			}
			time.Sleep(time.Millisecond * 200)
		}
		cb.Invoke("No job ready", js.Null())
		return
	}()
	return
}

func main() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	opts := &hub.HubOpts{
		AttemptRestore: false,
		SpokeSpan:      time.Second * 1,
		Persister:      &persistence.NoopPersister{},
		MaxCFSize:      hub.DefaultMaxCFSize,
	}
	h = hub.NewHub(opts)
	RegisterFunction("addJob", addJob)
	RegisterFunction("nextJob", nextJob)
	RegisterFunction("nextJobPromise", nextJobPromise)

	select {}
}

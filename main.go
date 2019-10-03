/*
Goyaad is a cancelable job scheduler.

There are two main primitives that Goyaad provides
	- Time ordered scheduleable jobs
	- Cancelation of scheduled jobs

Installation:
	- go get github.com/urjitbhatia/goyaad
	- Docker image: https://hub.docker.com/r/urjitbhatia/goyaad
*/
package main

import (
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime/debug"

	"github.com/rs/zerolog/log"

	"github.com/urjitbhatia/goyaad/cmd"
)

// Set by gorelease during binary
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func init() {
	cmd.SetBuildInfo(version, date, commit)
}

func main() {
	// More Aggressive GC
	if os.Getenv("GOGC") == "" {
		log.Info().Msg("Applying default GC tuning")
		debug.SetGCPercent(5)
	} else {
		log.Info().Str("GCPercent", os.Getenv("GOGC")).Msg("Using custom GC tuning")
	}
	go func() {
		err := http.ListenAndServe(":6060", nil)
		if err != nil {
			log.Error().Err(err).Msg("pprof server has stopped")
		}
	}()
	cmd.Execute()
}

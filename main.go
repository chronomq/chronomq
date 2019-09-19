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

	"github.com/sirupsen/logrus"

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
	// logrus.SetLevel(logrus.DebugLevel)
	// More Aggressive GC
	if os.Getenv("GOGC") == "" {
		logrus.Infof("Applying default GC tuning")
		debug.SetGCPercent(5)
	} else {
		logrus.Infof("Custom GCPercent set to: %s", os.Getenv("GOGC"))
	}
	go func() {
		logrus.Info(http.ListenAndServe(":6060", nil))
	}()
	cmd.Execute()
}

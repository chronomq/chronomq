/*
yaad is a cancelable job scheduler.

There are two main primitives that yaad provides
	- Time ordered scheduleable jobs
	- Cancelation of scheduled jobs

Installation:
	- go get github.com/urjitbhatia/yaad
	- Docker image: https://hub.docker.com/r/urjitbhatia/yaad
*/
package main

import (
	_ "net/http/pprof"

	"github.com/urjitbhatia/yaad/cmd"
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
	cmd.Execute()
}

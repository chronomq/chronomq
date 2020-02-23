/*
chronomq is a cancelable job/message scheduler.

There are two main primitives that Chronomq provides
	- Time ordered scheduleable jobs
	- Cancelation of scheduled jobs

Installation:
	- go get github.com/chronomq/chronomq
	- Docker image: https://hub.docker.com/r/chronomq/chronomq
*/
package main

import (
	_ "net/http/pprof"

	"github.com/chronomq/chronomq/cmd"
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

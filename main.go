package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime/debug"

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
		log.Println("Applying default GC tuning")
		debug.SetGCPercent(5)
	}
	go func() {
		log.Println(http.ListenAndServe(":6060", nil))
	}()
	cmd.Execute()
}

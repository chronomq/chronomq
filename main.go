package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime/debug"

	"github.com/urjitbhatia/goyaad/cmd"
)

func main() {
	// logrus.SetLevel(logrus.DebugLevel)
	// More Aggresive GC
	if os.Getenv("GOGC") == "" {
		log.Println("Applying default GC tuning")
		debug.SetGCPercent(5)
	}
	go func() {
		log.Println(http.ListenAndServe(":6060", nil))
	}()
	cmd.Execute()
}

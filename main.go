package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
	"runtime/debug"

	"github.com/urjitbhatia/goyaad/cmd"
)

func main() {
	// logrus.SetLevel(logrus.DebugLevel)
	// More Aggresive GC
	debug.SetGCPercent(5)
	go func() {
		log.Println(http.ListenAndServe(":6060", nil))
	}()
	cmd.Execute()
}

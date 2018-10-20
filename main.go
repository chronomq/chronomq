package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"

	"github.com/urjitbhatia/goyaad/cmd"
)

func main() {
	go func() {
		log.Println(http.ListenAndServe(":6060", nil))
	}()
	cmd.Execute()
}

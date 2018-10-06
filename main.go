package main

import (
	"log"

	"github.com/sirupsen/logrus"
	"github.com/urjitbhatia/goyaad/pkg/protocol"
)

func main() {
	logrus.Info("I AM GROOT!")
	logrus.SetLevel(logrus.DebugLevel)

	s := protocol.NewYaadServer()

	log.Fatal(s.ListenAndServe("tcp", ":11300"))
}

package goyaad_test

import (
	"log"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"testing"
)

func TestYaad(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(GinkgoWriter)
	log.SetOutput(GinkgoWriter)
	RegisterFailHandler(Fail)
	RunSpecs(t, "GoYaad Suite")
}

package monitor

import (
	"testing"

	"github.com/rs/zerolog/log"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func init() {
	log.Logger = log.Output(GinkgoWriter)
}

func TestMonitor(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Monitor Suite")
}

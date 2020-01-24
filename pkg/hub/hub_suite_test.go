package hub_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog/log"
)

func TestYaad(t *testing.T) {
	defer GinkgoRecover()

	log.Logger = log.Output(GinkgoWriter)

	RegisterFailHandler(Fail)
	RunSpecs(t, "GoYaad Suite")
}

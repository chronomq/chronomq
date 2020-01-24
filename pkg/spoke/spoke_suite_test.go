package spoke_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog/log"
)

func TestSpoke(t *testing.T) {
	defer GinkgoRecover()
	log.Logger = log.Output(GinkgoWriter)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Spoke Suite")
}

package rpc_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func TestChronomq(t *testing.T) {
	defer GinkgoRecover()
	log.Logger = zerolog.New(GinkgoWriter)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Chronomq Protocol Suite")
}

func ExpectNoErr(err error) {
	defer GinkgoRecover()
	Expect(err).To(BeNil())
}

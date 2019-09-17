package protocol_test

import (
	"log"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestYaad(t *testing.T) {
	defer GinkgoRecover()

	logrus.SetOutput(GinkgoWriter)
	log.SetOutput(GinkgoWriter)

	RegisterFailHandler(Fail)
	RunSpecs(t, "GoYaad Protocol Suite")
}

func ExpectNoErr(err error) {
	defer GinkgoRecover()
	Expect(err).To(BeNil())
}

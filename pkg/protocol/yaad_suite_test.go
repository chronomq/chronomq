package protocol_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestYaad(t *testing.T) {
	defer GinkgoRecover()

	RegisterFailHandler(Fail)
	RunSpecs(t, "GoYaad Protocol Suite")
}

func ExpectNoErr(err error) {
	defer GinkgoRecover()
	Expect(err).To(BeNil())
}

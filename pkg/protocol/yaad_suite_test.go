package protocol_test

import (
	"log"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"testing"
)

func TestYaad(t *testing.T) {
	logrus.SetOutput(GinkgoWriter)
	log.SetOutput(GinkgoWriter)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Yaad Protocol Suite")
}

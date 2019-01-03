package persistence_test

import (
	"log"
	"testing"

	"github.com/sirupsen/logrus"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestPersistence(t *testing.T) {
	defer GinkgoRecover()

	logrus.SetOutput(GinkgoWriter)
	log.SetOutput(GinkgoWriter)

	RegisterFailHandler(Fail)
	RunSpecs(t, "Persistence Suite")
}

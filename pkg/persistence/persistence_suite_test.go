package persistence_test

import (
	"testing"

	"github.com/sirupsen/logrus"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestPersistence(t *testing.T) {
	logrus.SetOutput(GinkgoWriter)
	defer GinkgoRecover()
	RegisterFailHandler(Fail)
	RunSpecs(t, "Persistence Suite")
}

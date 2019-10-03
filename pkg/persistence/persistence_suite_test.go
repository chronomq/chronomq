package persistence_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func TestPersistence(t *testing.T) {
	defer GinkgoRecover()

	log.Logger = zerolog.New(GinkgoWriter)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Persistence Suite")
}

package goyaad_test

import (
	"log"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"github.com/urjitbhatia/goyaad/pkg/metrics"

	"testing"
)

func TestYaad(t *testing.T) {
	metrics.InitMetrics("unix://dev/null")
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(GinkgoWriter)
	log.SetOutput(GinkgoWriter)
	RegisterFailHandler(Fail)
	RunSpecs(t, "GoYaad Suite")
}

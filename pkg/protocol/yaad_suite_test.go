package protocol_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestYaad(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Yaad Protocol Suite")
}

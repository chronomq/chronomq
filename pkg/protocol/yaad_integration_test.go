package protocol_test

import (
	"net"
	"net/textproto"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/urjitbhatia/goyaad/pkg/protocol"
)

var _ = Describe("Test protocol:", func() {
	var srv *protocol.Server
	var addr = ":9000"
	var proto = "tcp"

	BeforeSuite(func() {
		srv = protocol.NewServer()
		go func() {
			ExpectNoErr(srv.ListenAndServe(proto, addr))
		}()

		// This ensures all contexts get a running server
		Eventually(func() error {
			c, err := net.Dial(proto, addr)
			if err == nil {
				c.Close()
			}
			return err
		}, "1s").Should(BeNil())
	})

	It("echos command", func(done Done) {
		defer close(done)

		c, err := net.Dial(proto, addr)
		ExpectNoErr(err)
		tc := textproto.NewConn(c)
		defer tc.Close()

		hw := "Hello world"
		_, err = tc.Cmd("%s", hw)
		ExpectNoErr(err)

		resp, err := tc.ReadLine()
		ExpectNoErr(err)
		Expect(resp).To(Equal(hw))
	}, 0.1)
})

func ExpectNoErr(err error) {
	defer GinkgoRecover()
	Expect(err).To(BeNil())
}

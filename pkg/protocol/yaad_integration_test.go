package protocol_test

import (
	"net"
	"net/textproto"
	"time"

	"github.com/kr/beanstalk"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/urjitbhatia/goyaad/pkg/protocol"
)

var _ = Describe("Test protocol:", func() {
	var srv *protocol.Server
	var addr = ":9000"
	var proto = "tcp"
	var bconn *beanstalk.Conn

	BeforeSuite(func(done Done) {
		defer close(done)

		srv = protocol.NewYaadServer(false)
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

		conn, err := beanstalk.Dial(proto, addr)
		ExpectNoErr(err)
		bconn = conn
	}, 0.1)

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

	Describe("Producer commands using beanstalkd client", func() {
		It("Pause tube", func(done Done) {
			defer close(done)
			t := beanstalk.Tube{Conn: bconn, Name: "default"}
			err := t.Pause(1 * time.Millisecond)
			ExpectNoErr(err)
		}, 0.2)

		It("Lists tubes", func(done Done) {
			defer close(done)
			tubes, err := bconn.ListTubes()
			ExpectNoErr(err)
			Expect(len(tubes)).To(Equal(1))
		}, 0.1)

		It("Pausing unknown tube fails", func(done Done) {
			defer close(done)
			t := beanstalk.Tube{Conn: bconn, Name: "foo"}
			err := t.Pause(time.Millisecond * 1)
			Expect(err).ToNot(BeNil())
		}, 0.1)

		It("Puts a job and then reads it", func(done Done) {
			defer close(done)
			hw := "Hello world"
			id, err := bconn.Put([]byte(hw), 10, 1, 1)
			ExpectNoErr(err)
			Expect(id).To(BeNumerically(">=", 0))

			id2, err := bconn.Put([]byte(hw), 10, 1, 1)
			ExpectNoErr(err)
			Expect(id2).To(BeNumerically(">=", id))

			rid, body, err := bconn.Reserve(1 * time.Minute)
			ExpectNoErr(err)
			Expect(rid).To(Or(Equal(id), Equal(id2)))
			Expect(string(body)).To(Equal(hw))
		})

		It("Puts a job and then deletes it", func(done Done) {
			defer close(done)
			hw := "Hello world"
			id, err := bconn.Put([]byte(hw), 10, 1, 1)
			ExpectNoErr(err)
			Expect(id).To(BeNumerically(">=", 0))

			//delete
			ExpectNoErr(bconn.Delete(id))
		})
	})
})

func ExpectNoErr(err error) {
	defer GinkgoRecover()
	Expect(err).To(BeNil())
}

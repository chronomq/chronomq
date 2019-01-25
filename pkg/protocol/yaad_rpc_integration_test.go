package protocol_test

import (
	"fmt"
	"io"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/urjitbhatia/goyaad/pkg/goyaad"
	"github.com/urjitbhatia/goyaad/pkg/persistence"
	"github.com/urjitbhatia/goyaad/pkg/protocol"
)

var _ = Describe("Test rpc protocol:", func() {
	defer GinkgoRecover()
	var port = 9001
	var client *protocol.RPCClient

	var srv io.Closer
	var ctr int
	var hub *goyaad.Hub

	BeforeEach(func(done Done) {
		defer close(done)
		var opts = goyaad.HubOpts{
			AttemptRestore: false,
			Persister:      persistence.NewJournalPersister(""),
			SpokeSpan:      time.Second * 5}
		ctr++
		var err error
		hub = goyaad.NewHub(&opts)
		addr := fmt.Sprintf(":%d", port)
		srv, err = protocol.ServeRPC(hub, addr)
		Expect(err).NotTo(HaveOccurred())
		port++

		client = &protocol.RPCClient{}

		// This ensures all contexts get a running server
		Eventually(func() error {
			err := client.Connect(addr)
			return err
		}, "1s").Should(BeNil())
	}, 0.5)

	AfterEach(func(done Done) {
		defer close(done)
		err := srv.Close()
		Expect(err).NotTo(HaveOccurred())
		srv = nil
		hub = nil
		client = nil
	})

	It("pings rpc server", func(done Done) {
		defer close(done)
		defer GinkgoRecover()
		Expect(client.Ping()).NotTo(HaveOccurred())
	}, 0.1)

	It("Puts a job and then reads it", func(done Done) {
		defer close(done)
		defer GinkgoRecover()

		hw := "Hello world"
		id, err := client.Put([]byte(hw), 1)
		Expect(err).NotTo(HaveOccurred())
		Expect(id).ToNot(BeEmpty())

		id2, err := client.Put([]byte(hw), 1)
		Expect(err).NotTo(HaveOccurred())
		Expect(id2).ToNot(BeEmpty())

		id3, err := client.Put([]byte(hw), 1)
		Expect(err).NotTo(HaveOccurred())
		Expect(id3).ToNot(BeEmpty())

		rid, body, err := client.Next(1 * time.Minute)
		Expect(err).NotTo(HaveOccurred())
		Expect(rid).To(Equal(id))
		Expect(string(body)).To(Equal(hw))
	}, 20)

	It("Puts a job with an id and then reads it", func(done Done) {
		defer close(done)
		defer GinkgoRecover()

		Expect(client.Ping()).NotTo(HaveOccurred())

		hw := "Hello world"
		err := client.PutWithID("foo", []byte(hw), time.Nanosecond)
		ExpectNoErr(err)

		rid, body, err := client.Next(1 * time.Minute)
		Expect(err).NotTo(HaveOccurred())
		Expect(rid).To(Equal("foo"))
		Expect(string(body)).To(Equal(hw))
	}, 20)

	It("Puts a job and then deletes it", func(done Done) {
		defer close(done)
		defer GinkgoRecover()
		hw := "Hello world"
		id, err := client.Put([]byte(hw), 1)
		Expect(err).NotTo(HaveOccurred())
		Expect(id).ToNot(BeEmpty())

		//delete
		ExpectNoErr(client.Cancel(id))
	})
})

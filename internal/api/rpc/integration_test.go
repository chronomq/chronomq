package rpc_test

import (
	"fmt"
	"io"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	yaadrpc "github.com/urjitbhatia/goyaad/api/rpc/goyaad"
	. "github.com/urjitbhatia/goyaad/internal/api/rpc"
	"github.com/urjitbhatia/goyaad/pkg/goyaad"
	"github.com/urjitbhatia/goyaad/pkg/persistence"
)

var _ = Describe("Test rpc protocol:", func() {
	defer GinkgoRecover()
	var port = 9001
	var client *yaadrpc.Client

	var srv io.Closer
	var hub *goyaad.Hub

	BeforeEach(func(done Done) {
		defer close(done)
		var opts = goyaad.HubOpts{
			AttemptRestore: false,
			Persister:      persistence.NewJournalPersister(""),
			SpokeSpan:      time.Second * 5}
		var err error
		hub = goyaad.NewHub(&opts)
		addr := fmt.Sprintf(":%d", port)
		srv, err = ServeRPC(hub, addr)
		Expect(err).NotTo(HaveOccurred())
		port++

		client = &yaadrpc.Client{}

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

		// We can inspect without consuming too
		rpcJobs := []*yaadrpc.Job{}
		err = client.InspectN(2, &rpcJobs)
		Expect(err).To(BeNil())
		Expect(len(rpcJobs)).To(Equal(1))
		Expect(rpcJobs[0].ID).To(Equal("foo"))
		Expect(rpcJobs[0].Body).To(Equal([]byte(hw)))

		rid, body, err := client.Next(1 * time.Minute)
		Expect(err).NotTo(HaveOccurred())
		Expect(rid).To(Equal("foo"))
		Expect(string(body)).To(Equal(hw))
	}, 20)

	It("Puts multiple jobs with ids and then inpects them", func(done Done) {
		defer close(done)
		defer GinkgoRecover()

		Expect(client.Ping()).NotTo(HaveOccurred())

		n := 10
		hw := "Hello world"
		for i := 0; i < n; i++ {
			err := client.PutWithID("foo"+string(i), []byte(hw), time.Nanosecond)
			ExpectNoErr(err)
		}

		// InspectN < n
		inspectN := 5
		rpcJobs := []*yaadrpc.Job{}
		err := client.InspectN(inspectN, &rpcJobs)
		Expect(err).To(BeNil())
		Expect(len(rpcJobs)).To(Equal(inspectN))
		for i := 0; i < inspectN; i++ {
			Expect(rpcJobs[i].ID).To(Equal("foo" + string(i)))
			Expect(rpcJobs[i].Body).To(Equal([]byte(hw)))
		}
		// InspectN == n
		inspectN = n
		rpcJobs = []*yaadrpc.Job{}
		err = client.InspectN(inspectN, &rpcJobs)
		Expect(err).To(BeNil())
		Expect(len(rpcJobs)).To(Equal(inspectN))
		for i := 0; i < inspectN; i++ {
			Expect(rpcJobs[i].ID).To(Equal("foo" + string(i)))
			Expect(rpcJobs[i].Body).To(Equal([]byte(hw)))
		}

		// InspectN > n
		inspectN = n + 3
		rpcJobs = []*yaadrpc.Job{}
		err = client.InspectN(inspectN, &rpcJobs)
		Expect(err).To(BeNil())
		Expect(len(rpcJobs)).To(Equal(n))
		for i := 0; i < n; i++ {
			Expect(rpcJobs[i].ID).To(Equal("foo" + string(i)))
			Expect(rpcJobs[i].Body).To(Equal([]byte(hw)))
		}

		// Read them all
		for i := 0; i < n; i++ {
			rid, body, err := client.Next(1 * time.Minute)
			Expect(err).NotTo(HaveOccurred())
			Expect(rid).To(Equal("foo" + string(i)))
			Expect(string(body)).To(Equal(hw))
		}
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

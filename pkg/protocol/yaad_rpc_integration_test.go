package protocol_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/urjitbhatia/goyaad/pkg/goyaad"
	"github.com/urjitbhatia/goyaad/pkg/persistence"
	"github.com/urjitbhatia/goyaad/pkg/protocol"
)

var _ = Describe("Test rpc protocol:", func() {
	var addr = ":9001"
	var client = &protocol.RPCClient{}

	var opts = goyaad.HubOpts{
		AttemptRestore: false,
		Persister:      persistence.NewJournalPersister(""),
		SpokeSpan:      time.Second * 5}
	go func() {
		protocol.ServeRPC(goyaad.NewHub(&opts), addr)
	}()

	BeforeEach(func(done Done) {
		defer close(done)
		// This ensures all contexts get a running server
		Eventually(func() error {
			err := client.Connect(addr)
			return err
		}, "1s").Should(BeNil())
	}, 0.5)

	It("pings rpc server", func(done Done) {
		defer close(done)
		ExpectNoErr(client.Ping())
	}, 0.1)

	Describe("Producer commands using rpc client", func() {
		It("Puts a job and then reads it", func(done Done) {
			defer close(done)
			hw := "Hello world"
			id, err := client.Put([]byte(hw), 1)
			ExpectNoErr(err)
			Expect(id).ToNot(BeEmpty())

			id2, err := client.Put([]byte(hw), 1)
			ExpectNoErr(err)
			Expect(id2).ToNot(BeEmpty())

			rid, body, err := client.Next(1 * time.Minute)
			ExpectNoErr(err)
			Expect(rid).To(Equal(id))
			Expect(string(body)).To(Equal(hw))
		})

		It("Puts a job and then deletes it", func(done Done) {
			defer close(done)
			hw := "Hello world"
			id, err := client.Put([]byte(hw), 1)
			ExpectNoErr(err)
			Expect(id).ToNot(BeEmpty())

			//delete
			ExpectNoErr(client.Cancel(id))
		})
	})
})

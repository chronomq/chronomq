package grpc_test

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/duration"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc"

	api "github.com/chronomq/chronomq/api/grpc/chronomq"
	server "github.com/chronomq/chronomq/internal/api/grpc"
	"github.com/chronomq/chronomq/pkg/chronomq"
	"github.com/chronomq/chronomq/pkg/persistence"
)

var _ = Describe("Test grpc protocol:", func() {
	defer GinkgoRecover()
	var port = 9001
	var client api.ChronomqClient

	var srv io.Closer
	var h *chronomq.Hub

	BeforeEach(func(done Done) {
		defer close(done)
		store, err := persistence.InMemStorage()
		Expect(err).NotTo(HaveOccurred())
		var opts = chronomq.HubOpts{
			AttemptRestore: false,
			Persister:      persistence.NewJournalPersister(store),
			SpokeSpan:      time.Second * 5}
		h = chronomq.NewHub(&opts)
		addr := fmt.Sprintf(":%d", port)
		srv, err = server.ServeGRPC(h, addr)
		Expect(err).NotTo(HaveOccurred())
		port++

		// This ensures all contexts get a running server
		Eventually(func() error {
			conn, err := grpc.Dial(addr,
				grpc.WithInsecure(),
				grpc.WithTimeout(time.Second*2),
			)
			if err != nil {
				return err
			}
			client = api.NewChronomqClient(conn)
			return err
		}, "2s").Should(BeNil())
	}, 2.5)

	AfterEach(func(done Done) {
		defer close(done)
		err := srv.Close()
		Expect(err).NotTo(HaveOccurred())
		srv = nil
		h = nil
		client = nil
	})

	putRequest := func(id string, body []byte, delay time.Duration) *api.PutRequest {
		return &api.PutRequest{
			Job: &api.Job{
				Id:    id,
				Body:  body,
				Delay: ptypes.DurationProto(delay),
			},
		}
	}

	It("Puts a job and then reads it", func(done Done) {
		defer close(done)
		defer GinkgoRecover()
		ctx := context.Background()

		hw := []byte("Hello world")
		_, err := client.Put(ctx, putRequest("job1", hw, time.Second))
		Expect(err).NotTo(HaveOccurred())

		_, err = client.Put(ctx, putRequest("job2", hw, time.Second*2))
		Expect(err).NotTo(HaveOccurred())

		_, err = client.Put(ctx, putRequest("job3", hw, time.Second*3))
		Expect(err).NotTo(HaveOccurred())

		resp, err := client.Next(ctx, &api.NextRequest{Timeout: &duration.Duration{Seconds: 10}})
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.GetJob().GetId()).To(Equal("job1"))
		Expect(resp.GetJob().GetBody()).To(Equal(hw))
	}, 20)

	It("Puts a job with an id and then reads it", func(done Done) {
		defer close(done)
		defer GinkgoRecover()

		hw := []byte("Hello world")
		_, err := client.Put(context.Background(), putRequest("foo", hw, time.Nanosecond))
		ExpectNoErr(err)

		// We can inspect without consuming too
		responses := []*api.InspectResponse{}
		inspectClient, err := client.Inspect(context.Background(), &api.InspectRequest{Count: 10})
		Expect(err).To(BeNil())
		for {
			r := &api.InspectResponse{}
			err = inspectClient.RecvMsg(r)
			if err == io.EOF {
				break
			}
			Expect(err).ToNot(HaveOccurred())
			responses = append(responses, r)
		}
		Expect(len(responses)).To(Equal(1))
		Expect(responses[0].GetJob().GetId()).To(Equal("foo"))
		Expect(responses[0].GetJob().GetBody()).To(Equal(hw))

		resp, err := client.Next(context.Background(),
			&api.NextRequest{Timeout: ptypes.DurationProto(time.Minute)})
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.GetJob().GetId()).To(Equal("foo"))
		Expect(resp.GetJob().GetBody()).To(Equal(hw))
	}, 20)

	It("Puts multiple jobs with ids and then inpects them", func(done Done) {
		defer close(done)
		defer GinkgoRecover()

		n := 10
		hw := []byte("Hello world")
		for i := 0; i < n; i++ {
			_, err := client.Put(context.Background(),
				putRequest(fmt.Sprintf("foo%d", i), hw, time.Nanosecond))
			ExpectNoErr(err)
		}

		// InspectN < n
		inspectN := int32(5)
		stream, err := client.Inspect(context.Background(),
			&api.InspectRequest{Count: inspectN})
		Expect(err).To(BeNil())

		recvCount := 0
		for {
			resp := &api.InspectResponse{}
			err = stream.RecvMsg(resp)
			if err == io.EOF {
				break
			}
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.GetJob().GetId()).To(Equal(fmt.Sprintf("foo%d", recvCount)))
			Expect(resp.GetJob().GetBody()).To(Equal(hw))
			recvCount++
		}
		Expect(recvCount).To(BeEquivalentTo(inspectN))

		// InspectN == n
		inspectN = int32(n)
		stream, err = client.Inspect(context.Background(),
			&api.InspectRequest{Count: inspectN})
		Expect(err).To(BeNil())
		recvCount = 0
		for {
			resp := &api.InspectResponse{}
			err = stream.RecvMsg(resp)
			if err == io.EOF {
				break
			}
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.GetJob().GetId()).To(Equal(fmt.Sprintf("foo%d", recvCount)))
			Expect(resp.GetJob().GetBody()).To(Equal(hw))
			recvCount++
		}
		Expect(recvCount).To(BeEquivalentTo(inspectN))

		// InspectN > n
		inspectN = int32(n + 3)
		stream, err = client.Inspect(context.Background(),
			&api.InspectRequest{Count: inspectN})
		Expect(err).To(BeNil())
		recvCount = 0
		for {
			resp := &api.InspectResponse{}
			err = stream.RecvMsg(resp)
			if err == io.EOF {
				break
			}
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.GetJob().GetId()).To(Equal(fmt.Sprintf("foo%d", recvCount)))
			Expect(resp.GetJob().GetBody()).To(Equal(hw))
			recvCount++
		}
		Expect(recvCount).To(BeEquivalentTo(n))

		// Read them all
		for i := 0; i < n; i++ {
			resp, err := client.Next(context.Background(),
				&api.NextRequest{Timeout: ptypes.DurationProto(time.Second * 10)})
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.GetJob().GetId()).To(Equal(fmt.Sprintf("foo%d", i)))
			Expect(resp.GetJob().GetBody()).To(Equal(hw))
		}
	}, 20)

	It("Puts a job and then deletes it", func(done Done) {
		defer close(done)
		defer GinkgoRecover()
		hw := []byte("Hello world")
		_, err := client.Put(context.Background(), putRequest("foo", hw, time.Second))
		Expect(err).NotTo(HaveOccurred())

		//delete
		_, err = client.Cancel(context.Background(), &api.CancelRequest{Id: "foo"})
		ExpectNoErr(err)
	})
})

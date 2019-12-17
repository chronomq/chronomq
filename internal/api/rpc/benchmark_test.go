package rpc_test

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	yaadrpc "github.com/urjitbhatia/goyaad/api/rpc/goyaad"
	. "github.com/urjitbhatia/goyaad/internal/api/rpc"
	"github.com/urjitbhatia/goyaad/pkg/goyaad"
)

var opts = goyaad.HubOpts{
	AttemptRestore: false,
	SpokeSpan:      time.Second * 5}

type jobPutter interface {
	Put(body []byte, delay time.Duration) (string, error)
}

func benchPut(b *testing.B, bodySize int, putter jobPutter) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		body := randStringBytes(bodySize)
		delay := time.Second * time.Duration(rand.Intn(50000))
		b.StartTimer()
		_, err := putter.Put(body, delay)
		b.StopTimer()
		if err != nil {
			b.Fatal("Error submitting job", err)
		}
	}
}

func BenchmarkRPCJobPuts(b *testing.B) {
	log.Logger = zerolog.New(ioutil.Discard)
	go func() {
		ServeRPC(goyaad.NewHub(&opts), ":8001")
	}()

	time.Sleep(15 * time.Millisecond) // wait for server to start
	client := &yaadrpc.Client{}
	err := client.Connect(":8001")
	if err != nil {
		b.Error(err)
	}
	ExpectNoErr(client.Ping())

	for bs := 1000; bs <= 20000; bs += 5000 {
		b.Run(fmt.Sprintf("PutJob_%d", bs), func(b *testing.B) { benchPut(b, bs, client) })
	}
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randStringBytes(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return b
}

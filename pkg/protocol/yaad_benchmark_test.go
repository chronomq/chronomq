package protocol_test

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/kr/beanstalk"
	"github.com/urjitbhatia/goyaad/pkg/goyaad"
	"github.com/urjitbhatia/goyaad/pkg/persistence"
	"github.com/urjitbhatia/goyaad/pkg/protocol"
)

var opts = goyaad.HubOpts{
	AttemptRestore: false,
	Persister:      persistence.NewJournalPersister(""),
	SpokeSpan:      time.Second * 5}

type jobPutter interface {
	Put(body []byte, delay time.Duration) (string, error)
}

type beanstalkdPutter struct {
	*beanstalk.Conn
}

func (b *beanstalkdPutter) Put(body []byte, delay time.Duration) (string, error) {
	_, err := b.Conn.Put(body, 10, delay, 1)
	return "", err
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

func BenchmarkBeanstalkdJobPuts(b *testing.B) {
	addr := ":8000"
	go func() {
		protocol.ServeBeanstalkd(goyaad.NewHub(&opts), addr)
	}()

	time.Sleep(5 * time.Millisecond) // wait for server to start
	conn, _ := beanstalk.Dial("tcp", addr)
	bputter := &beanstalkdPutter{conn}

	for bs := 1000; bs <= 20000; bs += 5000 {
		b.Run(fmt.Sprintf("PutJob_%d", bs), func(b *testing.B) { benchPut(b, bs, bputter) })
	}
}

func BenchmarkRPCJobPuts(b *testing.B) {
	go func() {
		protocol.ServeRPC(goyaad.NewHub(&opts), ":8001")
	}()

	time.Sleep(15 * time.Millisecond) // wait for server to start
	client := &protocol.RPCClient{}
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

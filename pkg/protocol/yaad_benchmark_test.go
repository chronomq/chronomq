package protocol_test

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/kr/beanstalk"
	"github.com/urjitbhatia/goyaad/pkg/protocol"
)

var srv = protocol.NewYaadServer(false)
var addr = ":9000"
var proto = "tcp"
var bconn *beanstalk.Conn

func benchPut(b *testing.B, bodySize int) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		body := randStringBytes(bodySize)
		delay := time.Second * time.Duration(rand.Intn(50000))
		b.StartTimer()
		_, err := bconn.Put(body, 10, delay, 1)
		b.StopTimer()
		if err != nil {
			b.Fatal("Error submitting job", err)
		}
	}
}

func BenchmarkJobPuts(b *testing.B) {

	go func() {
		ExpectNoErr(srv.ListenAndServe(proto, addr))
	}()

	time.Sleep(5 * time.Millisecond) // wait for server to start
	conn, _ := beanstalk.Dial(proto, addr)
	bconn = conn

	for bs := 1000; bs <= 50000; bs += 5000 {
		b.Run(fmt.Sprintf("PutJob_%d", bs), func(b *testing.B) { benchPut(b, bs) })
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

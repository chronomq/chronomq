package chronomq_test

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"testing"
	"time"

	"github.com/chronomq/chronomq/pkg/chronomq"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var body = []byte("")

func benchCancels(b *testing.B, jobCount int) {
	b.StopTimer()

	jobs := make([]*chronomq.Job, jobCount)
	for i := 0; i < jobCount; i++ {
		// put the jobs into the spoke
		jobs[i] = chronomq.NewJobAutoID(time.Now().Add(time.Hour), body)
	}

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		var s = chronomq.NewSpoke(time.Now(), time.Now().Add(time.Hour*10))
		for i := 0; i < jobCount; i++ {
			s.AddJobLocked(jobs[i])
		}

		// Shuffle the ids before deleting
		rand.Shuffle(len(jobs), func(i, j int) {
			jobs[i], jobs[j] = jobs[j], jobs[i]
		})

		b.StartTimer()
		for i := 0; i < jobCount; i++ {
			// cancel all of them randomly
			s.CancelJobLocked(jobs[i].ID())
		}
	}
}

func BenchmarkSpokeCancels(b *testing.B) {
	log.Logger = zerolog.New(ioutil.Discard)
	jobCounts := []int{500, 1000, 5000, 10000, 20000, 40000}
	for _, jc := range jobCounts {
		b.Run(fmt.Sprintf("CancelJob_%d", jc), func(b *testing.B) { benchCancels(b, jc) })
	}
}

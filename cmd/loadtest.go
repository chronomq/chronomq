package cmd

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/kr/beanstalk"
	"github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

var jobs = 1000
var connections = 1000
var maxDelaySec = 0
var minDelaySec = 0
var statsDPort = 8125
var enqueueMode = false
var dequeueMode = false
var addr = ":11300"

func init() {

	loadTestCmd.Flags().IntVarP(&jobs, "num", "n", 1000, "Number of total jobs")
	loadTestCmd.Flags().IntVarP(&connections, "con", "c", 5, "Number of total connections")
	loadTestCmd.Flags().IntVarP(&maxDelaySec, "delayMax", "M", 60, "Max delay in seconds (Delay is random over delayMin, delayMax)")
	loadTestCmd.Flags().IntVarP(&minDelaySec, "delayMin", "N", 0, "Min delay in seconds (Delay is random over delayMin, delayMax)")
	loadTestCmd.Flags().IntVarP(&statsDPort, "statsd", "s", 8125, "StatsD port")
	loadTestCmd.Flags().BoolVarP(&enqueueMode, "enqueue", "e", false, "Enqueue jobs")
	loadTestCmd.Flags().BoolVarP(&dequeueMode, "dequeueMode", "d", false, "Dequeue jobs")
	loadTestCmd.Flags().StringVarP(&addr, "addr", "a", ":11300", "Server address (host:port)")

	rootCmd.AddCommand(loadTestCmd)
}

var loadTestCmd = &cobra.Command{
	Use:   "loadtest",
	Short: "Run a yaad loadtest",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Running Yaad load test")
		runLoadTest()
	},
}

func runLoadTest() {
	logrus.WithFields(logrus.Fields{
		"MaxJobs":        jobs,
		"MaxConnections": connections,
		"MaxDelaySec":    maxDelaySec,
		"EnqueueMode":    enqueueMode,
		"DequeueMode":    dequeueMode,
		"Addr":           addr,
	}).Info("Setting up load test parameters")

	enqWG := &sync.WaitGroup{}
	deqWG := &sync.WaitGroup{}

	stopDeq := make(chan struct{})
	deqJobs := make(chan struct{})

	if jobs == 0 {
		dequeueMode = false
		enqueueMode = false
	}

	if dequeueMode {
		dequeueCount := 0
		go func() {
			for range deqJobs {
				dequeueCount++
				if dequeueCount == jobs {
					logrus.Infof("Dequeued all jobs: %d", dequeueCount)
					for c := 0; c < connections; c++ {
						stopDeq <- struct{}{}
					}
				}
			}
		}()
	}

	var enqJobs chan *testJob
	if enqueueMode {
		enqJobs = generateJobs()
	}

	for c := 0; c < connections; c++ {
		logrus.Infof("Creating connection: %d", c)
		conn, err := beanstalk.Dial("tcp", addr)
		if err != nil {
			logrus.WithError(err).Fatalf("Failed to connect for worker: %d", c)
		}

		if enqueueMode {
			logrus.Infof("Enqueuing using connection: %d", c)
			enqWG.Add(connections)
			go enqueue(enqWG, c, conn, enqJobs)
		}

		if dequeueMode {
			deqWG.Add(1)
			logrus.Infof("Dequeuing using connection: %d", c)
			go dequeue(deqWG, c, conn, deqJobs, stopDeq)
		}
	}

	enqWG.Wait()
	deqWG.Wait()
}

func dequeue(deqWG *sync.WaitGroup, c int, conn *beanstalk.Conn, deqJobs chan struct{}, stopDeq chan struct{}) {

	go func() {
		for {
			id, _, err := conn.Reserve(time.Second * 1)
			if err != nil {
				switch err {
				case beanstalk.ErrTimeout:
					// ok?
					continue
				default:
					logrus.WithError(err).Fatalf("Failed to dequeue for worker: %d", c)
				}
			}
			conn.Delete(id)
			deqJobs <- struct{}{}
		}
	}()

	<-stopDeq
	deqWG.Done()
	logrus.Infof("Stopping dequeue for connection: %d", c)
}

func enqueue(wg *sync.WaitGroup, c int, conn *beanstalk.Conn, jobs chan *testJob) {
	defer wg.Done()
	for j := range jobs {
		_, err := conn.Put(j.data, 0, time.Second*time.Duration(j.delaySec), time.Second*10)
		if err != nil {
			logrus.WithError(err).Fatalf("Failed to enqueue for worker: %d", c)
		}
	}
	logrus.Infof("Connection: %c done enqueueing", c)
}

func generateJobs() chan *testJob {
	out := make(chan *testJob, connections)

	go func() {
		for i := 0; i < jobs; i++ {
			j := &testJob{
				data:     randStringBytes(1000), // 1Kb body
				delaySec: rand.Intn(maxDelaySec-minDelaySec) + minDelaySec,
			}
			out <- j
		}
		close(out)
	}()

	return out
}

type testJob struct {
	data     []byte
	delaySec int
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randStringBytes(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return b
}

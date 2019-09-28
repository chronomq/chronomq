package cmd

import (
	"bytes"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/urjitbhatia/goyaad/pkg/metrics"
	"github.com/urjitbhatia/goyaad/pkg/protocol"
)

var jobs = 1000
var connections = 1000
var maxDelaySec = 0
var minDelaySec = 0

var enqueueMode = false
var dequeueMode = false
var nsTolerance int64 = 1000
var enableTolerance = false

var sizeBytes = 100

func init() {

	loadTestCmd.Flags().Int64VarP(&nsTolerance, "nsTolerance", "t", 1000, "Dequeued jobs time order tolerance in ns")
	loadTestCmd.Flags().BoolVarP(&enableTolerance, "enableTolerance", "T", false, "Calculate time tolerance")

	loadTestCmd.Flags().IntVarP(&sizeBytes, "size", "z", 1000, "Job size in bytes")
	loadTestCmd.Flags().IntVarP(&jobs, "num", "n", 1000, "Number of total jobs")
	loadTestCmd.Flags().IntVarP(&connections, "con", "c", 5, "Number of connections to use")
	loadTestCmd.Flags().IntVarP(&maxDelaySec, "delayMax", "M", 60, "Max delay in seconds (Delay is random over delayMin, delayMax)")
	loadTestCmd.Flags().IntVarP(&minDelaySec, "delayMin", "N", 0, "Min delay in seconds (Delay is random over delayMin, delayMax)")

	loadTestCmd.Flags().BoolVarP(&enqueueMode, "enqueue", "e", false, "Enqueue jobs")
	loadTestCmd.Flags().BoolVarP(&dequeueMode, "dequeueMode", "d", false, "Dequeue jobs")

	rootCmd.AddCommand(loadTestCmd)
}

var loadTestCmd = &cobra.Command{
	Use:   "loadtest",
	Short: "Run a yaad loadtest",
	Run: func(cmd *cobra.Command, args []string) {
		setLogLevel()
		fmt.Println("Running Yaad load test")

		if !enqueueMode && !dequeueMode {
			logrus.Fatal("One of enqueue mode or dequeue mode required. See --help.")
		}
		runLoadTest()
	},
}

func runLoadTest() {
	metrics.InitMetrics(statsAddr)

	logrus.WithFields(logrus.Fields{
		"MaxJobs":        jobs,
		"MaxConnections": connections,
		"MaxDelaySec":    maxDelaySec,
		"MinDelaySec":    minDelaySec,
		"EnqueueMode":    enqueueMode,
		"DequeueMode":    dequeueMode,
		"RPCAddr":        raddr,
	}).Info("Setting up load test parameters")

	enqWG := &sync.WaitGroup{}
	deqWG := &sync.WaitGroup{}

	stopDeq := make(chan struct{}, connections)
	deqJobs := make(chan struct{})
	data := randStringBytes(sizeBytes)

	clients := []*protocol.RPCClient{}
	for c := 0; c < connections; c++ {
		client := &protocol.RPCClient{}
		err := client.Connect(raddr)
		if err != nil {
			logrus.WithError(err).Fatalf("Failed to connect for worker: %d", c)
		}

		client.Ping()
		clients = append(clients, client)
	}

	if jobs == 0 {
		dequeueMode = false
		enqueueMode = false
	}

	if dequeueMode {
		dequeueCount := 0
		go func() {
			logrus.Info("Dequeue sink chan starting...")
			for range deqJobs {
				dequeueCount++
				metrics.Incr("loadtest.dequeue")

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
		enqJobs = generateJobs(data)
	}

	for c, client := range clients {
		if enqueueMode {
			logrus.Infof("RPC Enqueuing using connection: %d", c)
			enqWG.Add(1)
			go enqueueRPC(enqWG, c, client, enqJobs)
		}

		if dequeueMode {
			logrus.Infof("RPC Dequeuing using connection: %d", c)
			deqWG.Add(1)
			go dequeueRPC(deqWG, c, client, deqJobs, stopDeq, data)
		}
	}

	logrus.Info("waiting for enqueue to end")
	enqWG.Wait()
	logrus.Info("waiting for dequeue to end")
	deqWG.Wait()
}

func dequeueRPC(deqWG *sync.WaitGroup, workerID int, rpcClient *protocol.RPCClient, deqJobs chan struct{}, stopDeq chan struct{}, data []byte) {
	go func() {
		var prevTriggerAt int64
		for {
			id, body, err := rpcClient.Next(time.Second * 1)
			if err != nil {
				if err.Error() == protocol.ErrTimeout.Error() {
					continue
				}
				// legit error
				logrus.Fatal("Error reading from rpc client: ", err)
			}
			logrus.Debugf("Canceling id: %s", id)
			err = rpcClient.Cancel(id)
			if err != nil {
				logrus.Fatal("Error canceling rpc job", err)
			}
			validateJob(data, body, prevTriggerAt, workerID)
			deqJobs <- struct{}{}
		}
	}()

	<-stopDeq
	deqWG.Done()
	logrus.Infof("Stopping dequeue for connection: %d", workerID)
}

func validateJob(testData []byte, body []byte, prevTriggerAt int64, workerID int) {
	parts := bytes.Split(body, []byte(` `))
	if len(parts) == 2 {
		body = parts[1] // leave just the body for equality check
	}

	// check order with tolerance because clocks are not monotonous
	if enableTolerance {
		triggerAt, err := strconv.ParseInt(string(parts[0]), 10, 64)
		if err != nil {
			logrus.WithError(err).Fatalf("Failed to dequeue and ready delay for worker: %d", workerID)
		}

		triggerAtTime := time.Unix(0, triggerAt)
		prevTriggerAtTime := time.Unix(0, prevTriggerAt)
		logrus.Debugf("Prev trigger at: %s Trigger at: %s", prevTriggerAtTime, triggerAtTime)

		if triggerAtTime.Before(prevTriggerAtTime) {
			diff := prevTriggerAt - triggerAt
			if diff >= nsTolerance {
				logrus.Errorf("Dequeue got jobs out of order for worker: %d\n\ttriggerAt:\t%s,\n\tprevTriggerAt:\t%s\n\tdelta:\t%d,\n\ttrigger.sub(prev):\t%f ms",
					workerID, triggerAtTime, prevTriggerAtTime, prevTriggerAtTime.Sub(triggerAtTime), float64(diff)/1e6)
				logrus.Fatalf("\ttriggerAtNS:\t%d,\n\tprevTriggerAtNS:\t%d\n\ttrigger.sub(prev):\t%d ns",
					triggerAt, prevTriggerAt, diff)
			}
		}
		prevTriggerAt = triggerAt
	}

	if !bytes.Equal(body, testData) {
		logrus.Fatalf("Dequeue got wrong body for worker: %d Expected: %s Got: %s",
			workerID, testData, body)
	}
}

func enqueueRPC(wg *sync.WaitGroup, workerID int, client *protocol.RPCClient, jobs chan *testJob) {
	defer wg.Done()
	for j := range jobs {
		var err error
		// To use PutWithID - have to ensure ids are globally unique among the multiple producer goroutines
		_, err = client.Put(j.data, time.Second*time.Duration(j.delaySec))
		metrics.Incr("loadtest.enqueuerpc")

		if err != nil {
			logrus.WithError(err).Fatalf("Failed to enqueue for worker: %d", workerID)
		}
	}
	logrus.Infof("Connection: %c done enqueueing", workerID)
}

func generateJobs(data []byte) chan *testJob {
	out := make(chan *testJob, connections)
	go func() {
		for i := 0; i < jobs; i++ {
			delaySec := rand.Intn(maxDelaySec-minDelaySec) + minDelaySec
			triggerAt := time.Now().Add(time.Second * time.Duration(delaySec))

			// Save the trigger at delay with the body so that we can verify order later on dequeue
			// By converting it to a time, it gives us global natural ordering...
			var j *testJob
			if enableTolerance {
				body := []byte(strconv.FormatInt(triggerAt.UnixNano(), 10) + " ")
				j = &testJob{
					data:     append(body, data...),
					delaySec: delaySec,
				}
			} else {
				j = &testJob{
					data:     data,
					delaySec: delaySec,
				}
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
	rand.Seed(0)
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return b
}

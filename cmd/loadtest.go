package cmd

import (
	"bytes"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/urjitbhatia/goyaad/api/rpc/goyaad"
	"github.com/urjitbhatia/goyaad/pkg/metrics"
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
			log.Fatal().Msg("One of enqueue mode or dequeue mode required. See --help.")
		}
		runLoadTest()
	},
}

func runLoadTest() {
	metrics.InitMetrics(statsAddr)

	log.Info().Int("MaxJobs", jobs).
		Int("MaxConnections", connections).
		Int("MaxDelaySec", maxDelaySec).
		Int("MinDelaySec", minDelaySec).
		Bool("EnqueueMode", enqueueMode).
		Bool("DequeueMode", dequeueMode).
		Str("RPCAddr", raddr).
		Msg("Setting up load test parameters")

	enqWG := &sync.WaitGroup{}
	deqWG := &sync.WaitGroup{}

	stopDeq := make(chan struct{}, connections)
	deqJobs := make(chan struct{})
	data := randStringBytes(sizeBytes)

	clients := []*goyaad.Client{}
	for c := 0; c < connections; c++ {
		client := &goyaad.Client{}
		err := client.Connect(raddr)
		if err != nil {
			log.Fatal().Int("WorkerID", c).Err(err).Msg("Failed to connect for worker")
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
			log.Info().Msg("Dequeue sink chan starting...")
			for range deqJobs {
				dequeueCount++
				metrics.Incr("loadtest.dequeue")

				if dequeueCount == jobs {
					log.Info().Int("DequeueCount", dequeueCount).Msg("Dequeued all jobs")
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
			log.Info().Int("ConnectionID", c).Msg("RPC Enqueuing")
			enqWG.Add(1)
			go enqueueRPC(enqWG, c, client, enqJobs)
		}

		if dequeueMode {
			log.Info().Int("ConnectionID", c).Msg("RPC Dequeuing")
			deqWG.Add(1)
			go dequeueRPC(deqWG, c, client, deqJobs, stopDeq, data)
		}
	}

	log.Info().Msg("waiting for enqueue to end")
	enqWG.Wait()
	log.Info().Msg("waiting for dequeue to end")
	deqWG.Wait()
}

func dequeueRPC(deqWG *sync.WaitGroup, workerID int, rpcClient *goyaad.Client, deqJobs chan struct{}, stopDeq chan struct{}, data []byte) {
	go func() {
		var prevTriggerAt int64
		for {
			id, body, err := rpcClient.Next(time.Second * 1)
			if err != nil {
				if err.Error() == goyaad.ErrTimeout.Error() {
					continue
				}
				// legit error
				log.Fatal().Err(err).Msg("Error reading from rpc client")
			}
			log.Debug().Str("JobID", id).Msg("Canceling job")
			err = rpcClient.Cancel(id)
			if err != nil {
				log.Fatal().Err(err).Msg("Error canceling rpc job")
			}
			validateJob(data, body, prevTriggerAt, workerID)
			deqJobs <- struct{}{}
		}
	}()

	<-stopDeq
	deqWG.Done()
	log.Info().Int("workerID", workerID).Msg("Stopping dequeue for connection")
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
			log.Fatal().Int("workerID", workerID).Err(err).Msg("Failed to dequeue and ready delay")
		}

		triggerAtTime := time.Unix(0, triggerAt)
		prevTriggerAtTime := time.Unix(0, prevTriggerAt)
		log.Debug().Time("prevTriggerAtTime", prevTriggerAtTime).
			Time("triggetAtTime", triggerAtTime).Send()

		if triggerAtTime.Before(prevTriggerAtTime) {
			diff := prevTriggerAt - triggerAt
			if diff >= nsTolerance {
				log.Error().Int("workerID", workerID).
					Time("triggerAtTime", triggerAtTime).
					Time("prevTriggerAtTime", prevTriggerAtTime).
					Dur("delta", prevTriggerAtTime.Sub(triggerAtTime)).
					Float64("triggerDeltaMS", float64(diff)/1e6).
					Msg("Dequeue got jobs out of order for worker")

				log.Fatal().Int64("triggerAtNS", triggerAt).
					Int64("prevTriggerAtNS", prevTriggerAt).
					Int64("triggerDeltaNS", diff).
					Send()
			}
		}
		prevTriggerAt = triggerAt
	}

	if !bytes.Equal(body, testData) {
		log.Fatal().Int("workderID", workerID).
			Bytes("expectedBody", testData).
			Bytes("receivedBody", body).
			Msg("Dequeue got wrong body")
	}
}

func enqueueRPC(wg *sync.WaitGroup, workerID int, client *goyaad.Client, jobs chan *testJob) {
	defer wg.Done()
	for j := range jobs {
		var err error
		// To use PutWithID - have to ensure ids are globally unique among the multiple producer goroutines
		err = client.PutWithID(j.id, j.data, time.Second*time.Duration(j.delaySec))
		metrics.Incr("loadtest.enqueuerpc")

		if err != nil {
			log.Fatal().Int("workderID", workerID).Err(err).Msg("Failed to enqueue")
		}
	}
	log.Info().Int("workerID", workerID).Msg("Connection done enqueueing")
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
					id:       fmt.Sprintf("%d", i),
					data:     append(body, data...),
					delaySec: delaySec,
				}
			} else {
				j = &testJob{
					id:       fmt.Sprintf("%d", i),
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
	id       string
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

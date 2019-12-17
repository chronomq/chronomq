package yaad_test

import (
	"math/rand"
	"os"
	"path"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/urjitbhatia/yaad/pkg/yaad"
	"github.com/urjitbhatia/yaad/pkg/persistence"
)

var dataDir = path.Join(os.TempDir(), "yaadtest")
var persister persistence.Persister

var _ = Describe("Test hub", func() {
	defer GinkgoRecover()

	BeforeEach(func() {
		persister = persistence.NewJournalPersister(dataDir)
		Expect(persister.ResetDataDir()).To(BeNil())
	})

	It("can create a hub", func() {
		// A hub with 10ms spokes
		h := NewHub(&HubOpts{SpokeSpan: time.Millisecond * 10, Persister: persister, AttemptRestore: false})
		Expect(h.Stats().CurrentJobs).To(Equal(int64(0)))
	})

	It("accepts jobs with random times and random spoke durations into a hub", func() {
		for i := 0; i < 50; i++ {
			h := NewHub(&HubOpts{
				SpokeSpan:      time.Second * time.Duration(rand.Intn(2999)+1),
				Persister:      persister,
				AttemptRestore: false})

			j := NewJobAutoID(time.Now().Add(time.Millisecond*time.Duration(rand.Intn(999999))), nil)
			h.AddJobLocked(j)
			Expect(h.Stats().CurrentJobs).To(Equal(int64(1)))

		}
	})

	It("walks job from a hub in proper order - with timeout", func(done Done) {
		defer close(done)

		// hub with spokes spanning  3000 nanosec (Faster for testing)
		opts := &HubOpts{
			SpokeSpan:      time.Nanosecond * 3000,
			Persister:      persister,
			AttemptRestore: false}
		h := NewHub(opts)

		// Add a jobs with a random trigger time in the future - max 9999 nanosec
		jobs := [1000]*Job{}
		for i := 0; i < len(jobs); i++ {
			// Some jobs could already be in the past
			triggerAt := time.Now().Add(time.Nanosecond * time.Duration(rand.Intn(9999)))
			if rand.Float32() <= 0.2 {
				triggerAt = time.Now().Add(time.Nanosecond * time.Duration(-1*rand.Intn(9999)))
			}

			j := NewJobAutoID(triggerAt, nil)

			jobs[i] = j
		}

		// Shuffle jobs
		rand.Shuffle(len(jobs), func(i, j int) {
			jobs[i], jobs[j] = jobs[j], jobs[i]
		})

		// Add all of them
		for i, j := range jobs {
			h.AddJobLocked(j)
			Expect(h.Stats().CurrentJobs).To(Equal(int64(i + 1)))
		}

		// Walk should return all jobs in global order
		walked := []*Job{}
		for h.Stats().CurrentJobs > 0 {
			walked = append(walked, h.NextLocked())
		}

		// Expect correct order
		var prev *Job = nil
		for _, j := range walked {
			if prev != nil {
				Expect(prev.TriggerAt().Before(j.TriggerAt()))
			}
			prev = j
		}

		Expect(h.Stats().CurrentJobs).To(Equal(int64(0)))

	}, 1.500)

	It("Persists and recovers from disk", func(done Done) {
		defer close(done)

		opts := &HubOpts{
			SpokeSpan:      time.Nanosecond * 3000,
			Persister:      persister,
			AttemptRestore: false}
		h := NewHub(opts)

		// Add a jobs with a random trigger time in the future - max 9999 nanosec
		jobs := [1000]*Job{}
		for i := 0; i < len(jobs); i++ {
			// Some jobs could already be in the past
			triggerAt := time.Now().Add(time.Nanosecond * time.Duration(rand.Intn(9999)))
			if rand.Float32() <= 0.2 {
				triggerAt = time.Now().Add(time.Nanosecond * time.Duration(-1*rand.Intn(9999)))
			}

			j := NewJobAutoID(triggerAt, nil)

			jobs[i] = j
		}

		jobMap := make(map[string]*Job, len(jobs))
		// Add all of them
		for i, j := range jobs {
			h.AddJobLocked(j)
			jobMap[j.ID()] = j
			Expect(h.Stats().CurrentJobs).To(Equal(int64(i + 1)))
		}

		// Reserve some jobs
		Expect(h.NextLocked()).ToNot(BeNil())
		Expect(h.NextLocked()).ToNot(BeNil())
		Expect(h.NextLocked()).ToNot(BeNil())

		// Persist
		persistErrs := h.PersistLocked()

		// if any errors pop up, fail the test
		for e := range persistErrs {
			Fail("Persist failed due to error: " + e.Error())
		}

		entries, err := persister.Recover()
		Expect(err).To(BeNil())
		counter := 0
		for range entries {
			counter++
		}

		Expect(int64(counter)).To(Equal(h.Stats().CurrentJobs))
	}, 15)

	It("bootstraps a new hub from a golden peristence record", func(done Done) {
		defer close(done)
		wd, _ := os.Getwd()
		persister := persistence.NewJournalPersister(path.Join(wd, "../../testdata/persist_golden"))
		opts := &HubOpts{
			SpokeSpan:      time.Nanosecond * 3000,
			Persister:      persister,
			AttemptRestore: false}
		h := NewHub(opts)
		err := h.Restore()
		Expect(err).NotTo(HaveOccurred())

		Expect(h.Stats().CurrentJobs).To(Equal(int64(1000)))
	})
})

package persistence_test

import (
	"io/ioutil"
	"os"
	"path"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/urjitbhatia/goyaad/pkg/goyaad"
	"github.com/urjitbhatia/goyaad/pkg/persistence"
)

var _ = Describe("Test stores", func() {

	Context("Filesystem store", func() {
		testDir := path.Join(os.TempDir(), "goyaadtest_storefs")

		BeforeEach(func() {
			s, err := persistence.NewFS(testDir)
			Expect(err).ToNot(HaveOccurred())
			err = s.Reset()
			Expect(err).ToNot(HaveOccurred())
		})

		It("stores data and reads it back", func() {
			s, err := persistence.NewFS(testDir)
			Expect(err).ToNot(HaveOccurred())

			j := goyaad.NewJobAutoID(time.Now(), testBody)
			w, err := s.Writer()
			Expect(err).To(BeNil())

			// write a job
			b, err := j.GobEncode()
			Expect(err).To(BeNil())

			n, err := w.Write(b)
			Expect(err).To(BeNil())
			Expect(len(b)).To(Equal(n))

			err = w.Close()
			Expect(err).To(BeNil())

			// read it back
			r, err := s.Reader()
			Expect(err).To(BeNil())
			rb, err := ioutil.ReadAll(r)
			Expect(err).To(BeNil())
			Expect(b).To(Equal(rb))

			// convert back to job and compare
			jj := &goyaad.Job{}
			err = jj.GobDecode(rb)
			Expect(err).To(BeNil())

			Expect(j.Body()).To(Equal(jj.Body()))
			Expect(j.ID()).To(Equal(jj.ID()))
			Expect(j.TriggerAt()).To(BeTemporally("==", jj.TriggerAt()))
		})
	})
})

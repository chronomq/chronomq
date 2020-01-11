package persistence_test

import (
	"io/ioutil"
	"net/url"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/urjitbhatia/goyaad/pkg/goyaad"
	"github.com/urjitbhatia/goyaad/pkg/persistence"
)

var _ = Describe("Test stores", func() {

	Context("Filesystem store", func() {
		bucketURL := &url.URL{Scheme: "mem", Path: "test"}
		var store persistence.Storage

		BeforeEach(func() {
			var err error
			store, err = persistence.StoreConfig{Bucket: bucketURL}.Storage()
			Expect(err).ToNot(HaveOccurred())
			err = store.Reset()
			Expect(err).ToNot(HaveOccurred())
		})

		It("stores data and reads it back", func() {
			j := goyaad.NewJobAutoID(time.Now(), testBody)
			w, err := store.Writer()
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
			r, err := store.Reader()
			Expect(err).To(BeNil())
			defer r.Close()
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

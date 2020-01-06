package persistence

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	s3util "github.com/rlmcpherson/s3gof3r"
)

// implements the bucket interface for testing
// implements io.Closeable as well
// not concurrent safe
type inMemBucket struct {
	name string
	data map[string]*bytes.Buffer
	*bytes.Buffer
}

// Close implements io.Closeable for inMemBucket
func (b *inMemBucket) Close() error {
	return nil
}

// provides an s3 storage backed by this in-mem bucket
func (b *inMemBucket) storage() (Storage, error) {
	var s s3
	s.b = b
	s.bucketName = "testbucket"
	s.key = "testprefix"
	return &s, s.setup()
}

func (b *inMemBucket) Delete(key string) error {
	// drop all data - ok for testing
	b.data = make(map[string]*bytes.Buffer)
	return nil
}

func (b *inMemBucket) PutWriter(path string, h http.Header, c *s3util.Config) (io.WriteCloser, error) {
	b.data[path] = b.Buffer
	return b, nil
}

func (b *inMemBucket) GetReader(path string, c *s3util.Config) (io.ReadCloser, http.Header, error) {
	if data, ok := b.data[path]; ok {
		return ioutil.NopCloser(data), http.Header{}, nil
	}
	return nil, nil, errors.New("Not found path: " + path)
}

var _ = Describe("Test s3 stores", func() {
	Context("S3 store", func() {
		tb := &inMemBucket{name: "testbucket",
			data:   make(map[string]*bytes.Buffer),
			Buffer: new(bytes.Buffer)}

		It("stores data and reads it back", func() {
			s, err := tb.storage()
			Expect(err).ToNot(HaveOccurred())
			w, err := s.Writer()
			Expect(err).ToNot(HaveOccurred())

			// write a job
			b := []byte(`actual testdata`)
			n, err := w.Write(b)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(b)).To(Equal(n))

			err = w.Close()
			Expect(err).ToNot(HaveOccurred())

			// read it back
			r, err := s.Reader()
			Expect(err).ToNot(HaveOccurred())
			rb, err := ioutil.ReadAll(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(b).To(Equal(rb))

			Expect(bytes.Equal(b, rb)).To(BeTrue())
		})
	})
})

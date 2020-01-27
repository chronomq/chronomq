package monitor

import (
	"sync"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type testSizeable struct {
	size uint64
}

func (ts *testSizeable) SizeOf() uint64 {
	return ts.size
}

var _ = Describe("Test memory monitor", func() {
	defer GinkgoRecover()
	It("alarms on breaching watermark", func() {
		mm := &memMonitor{
			watermark:         100,
			recoveryWatermark: 90,
			breachCond:        sync.NewCond(&sync.Mutex{}),
		}

		mm.Increment(&testSizeable{101}) // current 101
		Expect(mm.Breached()).To(BeTrue())
		mm.Decrement(&testSizeable{101}) // current 0
		Expect(mm.Breached()).To(BeFalse())
		Expect(mm.current).To(BeEquivalentTo(0)) // current 0

		mm.Increment(&testSizeable{99}) // current 99
		Expect(mm.Breached()).To(BeFalse())
		mm.Increment(&testSizeable{1}) // current 100
		Expect(mm.Breached()).To(BeTrue())
		Expect(mm.current).To(BeEquivalentTo(100)) // current 100

		var unfenced = make(chan struct{})
		go func() {
			mm.Fence()
			unfenced <- struct{}{}
		}()
		Eventually(unfenced).ShouldNot(Receive())
		Expect(mm.current).To(BeEquivalentTo(100)) // current 100
		mm.Decrement(&testSizeable{80})            // current 20 - unfenced
		Expect(mm.current).To(BeEquivalentTo(20))  // current 20
		Eventually(unfenced).Should(Receive())
	})
})

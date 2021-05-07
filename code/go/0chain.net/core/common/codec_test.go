package common

import (
	"sync"
	"testing"
)

type CodecTestStruct struct {
	Numbers []int `json:"numbers" msgpack:"nums"`
	mutex   sync.RWMutex
}

func (c *CodecTestStruct) getNumbers() []int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.Numbers
}

func (c *CodecTestStruct) setNumbers(numbers []int) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.Numbers = numbers
}

func (c *CodecTestStruct) DoReadLock() {
	c.mutex.RLock()
}

func (c *CodecTestStruct) DoReadUnlock() {
	c.mutex.RUnlock()
}

func TestConcurrentCodec(t *testing.T) {
	var o CodecTestStruct
	var wg sync.WaitGroup
	count := 0
	for idx := 0; idx < 10; idx++ {
		o.setNumbers(append(o.getNumbers(), 1))
		for i := 0; i < 10; i++ {
			var mi = i
			go func() {
				wg.Add(1)
				var nums = []int{}
				nums = append(nums, o.getNumbers()...)
				for j := 0; j < 1; j++ {
					nums = append(nums, 100*mi+j)
				}
				o.setNumbers(nums)
				wg.Done()
			}()
		}
		for i := 0; i < 10; i++ {
			encoded := ToMsgpack(&o)
			if encoded.Len() > 16 {
				count++
			}
		}
		wg.Wait()
	}
}

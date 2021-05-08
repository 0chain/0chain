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
	nums := make([]int, len(c.Numbers))
	copy(nums, c.Numbers)
	return nums
}

func (c *CodecTestStruct) setNumbers(numbers []int) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.Numbers = make([]int, len(numbers))
	copy(c.Numbers, numbers)
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
	for idx := 0; idx < 10; idx++ {
		o.setNumbers(append(o.getNumbers(), 1))
		for i := 0; i < 10; i++ {
			go func(mi int, wg *sync.WaitGroup) {
				wg.Add(1)
				nums := o.getNumbers()
				for j := 0; j < 1; j++ {
					nums = append(nums, 100*mi+j)
				}
				o.setNumbers(nums)
				wg.Done()
			}(i, &wg)
		}
		for i := 0; i < 10; i++ {
			_ = ToMsgpack(&o)
		}
		wg.Wait()
	}
}

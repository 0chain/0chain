package common

import (
	"fmt"
	"sync"
	"testing"
	//"encoding/hex"
)

type CodecTestStruct struct {
	Numbers []int `json:"numbers" msgpack:"nums"`
}

func TestConcurrentCodec(t *testing.T) {
	var o CodecTestStruct
	var wg sync.WaitGroup
	count := 0
	for idx := 0; idx < 100; idx++ {
		o.Numbers = append(o.Numbers, 1)
		for i := 0; i < 100; i++ {
			var mi = i
			go func() {
				wg.Add(1)
				nums := o.Numbers
				for j := 0; j < 1; j++ {
					nums = append(nums, 100*mi+j)
				}
				o.Numbers = nums
				wg.Done()
			}()
		}
		for i := 0; i < 100; i++ {
			encoded := ToMsgpack(o)
			if encoded.Len() > 16 {
				count++
			}
			//fmt.Printf("encoded: %v %v\n",len(o.Numbers),hex.EncodeToString(encoded.Bytes())[:16])
		}
		wg.Wait()
		fmt.Printf("all done: %v\n", count)
	}
}

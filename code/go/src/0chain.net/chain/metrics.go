package chain

import (
	"container/ring"
)

/*Metrics - interface for metric data*/
type Metrics interface {
	Collect(value int64, data interface{})
	Retrieve() []interface{}
}

/*PowerMetrics - struct for buffered power values*/
type PowerMetrics struct {
	power       int
	bufferLen   int
	powerBuffer []*ring.Ring
	values      []interface{}
}

/*NewPowerMetrics - creates, initializes PowerMetrics*/
func NewPowerMetrics(power int, bufferLen int) *PowerMetrics {
	buffer := make([]*ring.Ring, bufferLen)
	for idx := 0; idx < bufferLen; idx++ {
		buffer[idx] = ring.New(power)
	}
	return &PowerMetrics{
		power:       power,
		bufferLen:   bufferLen,
		powerBuffer: buffer,
		values:      make([]interface{}, power*bufferLen),
	}
}

/*Collect - checks if its a powered value and then adds data*/
func (pm *PowerMetrics) Collect(value int64, data interface{}) {
	var scale = int64(pm.power)
	for i := 0; i < pm.bufferLen; i++ {
		if value%scale != 0 {
			return
		} else {
			pm.powerBuffer[i].Value = data
			pm.powerBuffer[i] = pm.powerBuffer[i].Next()
		}
		scale *= int64(pm.power)
	}
}

/*Retrieve - gives the list of recent powered values*/
func (pm *PowerMetrics) Retrieve() []interface{} {
	var arr = make([]interface{}, pm.power)
	var arrIdx = len(arr) - 1

	var index = 0
	for idx := 0; idx < pm.bufferLen; idx++ {
		r := pm.powerBuffer[idx]
		r.Do(func(val interface{}) {
			if val != nil {
				arr[arrIdx] = val
				arrIdx--
			}
		})

		for i := arrIdx + 1; i < len(arr); i++ {
			pm.values[index] = arr[i]
			index++
		}
		arrIdx = len(arr) - 1
	}
	return pm.values[:index]
}

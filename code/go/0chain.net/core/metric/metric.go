package metric

import (
	"container/ring"
	"fmt"
	"time"
)

/*Metric - interface*/
type Metric interface {
	GetKey() int64
	GetTime() *time.Time
}

/*PowerMetrics - struct for buffered power values*/
type PowerMetrics struct {
	power        int
	bufferLen    int
	powerBuffer  []*ring.Ring
	CurrentValue Metric
}

// FormattedTime - get the formatted time
func FormattedTime(metric Metric) string {
	t := metric.GetTime()
	return fmt.Sprintf("%02d:%02d", t.Minute(), t.Second())
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
	}
}

/*Collect - checks for power value and then adds it to the buffer*/
func (pm *PowerMetrics) Collect(data Metric) {
	var scale = int64(pm.power)
	for i := 0; i < pm.bufferLen; i++ {
		if data.GetKey()%scale != 0 {
			return
		} else {
			pm.powerBuffer[i].Value = data
			pm.powerBuffer[i] = pm.powerBuffer[i].Next()
		}
		scale *= int64(pm.power)
	}
}

/*GetAll - gives list of recent power values*/
func (pm *PowerMetrics) GetAll() []Metric {
	values := make([]Metric, (pm.power)*(pm.bufferLen)+1)
	var index = 0
	if pm.CurrentValue != nil {
		values[0] = pm.CurrentValue
		index = 1
	}

	arr := make([]Metric, pm.power)
	var arrIdx = len(arr) - 1
	for idx := 0; idx < pm.bufferLen; idx++ {
		r := pm.powerBuffer[idx]
		r.Do(func(val interface{}) {
			if val != nil {
				arr[arrIdx] = val.(Metric)
				arrIdx--
			}
		})

		for i := arrIdx + 1; i < len(arr); i++ {
			values[index] = arr[i]
			index++
		}
		arrIdx = len(arr) - 1
	}
	return values[:index]
}

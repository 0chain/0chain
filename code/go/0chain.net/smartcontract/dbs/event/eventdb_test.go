package event

import (
	"context"
	"reflect"
	"runtime"
	"strconv"
	"testing"
)

func TestChannelBufferIntermediate(t *testing.T) {
	runtime.GOMAXPROCS(3)
	eventDb := &EventDb{
		eBufferChannel: make(chan eventCtx, 100),
		eChannel:       make(chan eventCtx, 100),
	}
	go eventDb.channelBufferIntermediate()

	for i := 0; i < 100000; i++ {
		select {
		case eventDb.eBufferChannel <- eventCtx{
			context.Background(),
			[]Event{
				{
					Index: strconv.Itoa(i),
				},
			},
		}:
			// Have to do this or else this goroutine runs forever and hits the default once the buffer is full
			runtime.Gosched()
		default:
			t.Errorf("Failed to send the event in order %v ", i)
			return
		}
	}

	for i := 0; i < 100000; i++ {
		select {
		case e := <-eventDb.eChannel:
			if !reflect.DeepEqual(e, eventCtx{
				context.Background(),
				[]Event{
					{
						Index: strconv.Itoa(i),
					},
				},
			}) {
				t.Errorf("error event not found in order %v", i)
				return
			}
		default:
			t.Errorf("error while recieving events in order %v", i)
			return
		}

	}
}

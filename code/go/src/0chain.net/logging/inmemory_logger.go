package logging

import (
	"container/ring"

	"go.uber.org/zap/zaptest/observer"

	. "go.uber.org/zap/zapcore"
)

//TODO make buffer size configurable
const (
	BUFFER_SIZE = 1024
)

type MemCore struct {
	LevelEnabler
	enc Encoder
	r   *ring.Ring
}

type MemLogger struct {
	core *MemCore
	logs [BUFFER_SIZE]*observer.LoggedEntry
}

func NewMemLogger(enc Encoder, enab LevelEnabler) *MemLogger {
	return &MemLogger{
		core: &MemCore{
			LevelEnabler: enab,
			enc:          enc,
			r:            ring.New(BUFFER_SIZE),
		},
	}
}

func (ml *MemLogger) GetCore() Core {
	return ml.core
}

func (ml *MemLogger) GetLogs() [BUFFER_SIZE]*observer.LoggedEntry {
	var index = 0
	mc := ml.core
	mc.r.Do(func(val interface{}) {
		if val != nil {
			ml.logs[index] = val.(*observer.LoggedEntry)
			index++
		}
	})
	return ml.logs
}

func (mc *MemCore) With(fields []Field) Core {
	clone := mc.clone()
	for i := range fields {
		fields[i].AddTo(clone.enc)
	}
	return clone
}

func (mc *MemCore) Check(ent Entry, ce *CheckedEntry) *CheckedEntry {
	if mc.Enabled(ent.Level) {
		return ce.AddCore(ent, mc)
	}
	return ce
}

func (mc *MemCore) Write(ent Entry, fields []Field) error {
	if mc.r != nil {
		mc.r.Value = &observer.LoggedEntry{
			Entry:   ent,
			Context: fields,
		}
		mc.r = mc.r.Next()
	}
	return nil
}

func (mc *MemCore) Sync() error {
	return nil
}

func (mc *MemCore) clone() *MemCore {
	return &MemCore{
		LevelEnabler: mc.LevelEnabler,
		enc:          mc.enc.Clone(),
		r:            mc.r,
	}
}

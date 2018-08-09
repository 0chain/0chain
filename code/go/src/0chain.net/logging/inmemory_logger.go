package logging

import (
	"container/ring"
	"io"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"go.uber.org/zap/zapcore"
)

//TODO make buffer size configurable
const (
	BufferSize = 1024
)

const (
	IncludeMessage    = 1
	IncludeFields     = 2
	IncludeStacktrace = 3
)

/*MemCore - a struct for ring buffered inmemory core */
type MemCore struct {
	zapcore.LevelEnabler
	enc zapcore.Encoder
	r   *ring.Ring
}

/*MemLogger - a struct for ring buffered inmemory logger */
type MemLogger struct {
	core *MemCore
}

/*NewMemLogger - create a new memory logger */
func NewMemLogger(enc zapcore.Encoder, enab zapcore.LevelEnabler) *MemLogger {
	return &MemLogger{
		core: &MemCore{
			LevelEnabler: enab,
			enc:          enc,
			r:            ring.New(BufferSize),
		},
	}
}

/*GetCore - get the core associted with this logger */
func (ml *MemLogger) GetCore() zapcore.Core {
	return ml.core
}

/*GetLogs - get the inmemory logs */
func (ml *MemLogger) GetLogs() []*observer.LoggedEntry {
	var index = BufferSize - 1
	mc := ml.core
	logs := make([]*observer.LoggedEntry, BufferSize)
	mc.r.Do(func(val interface{}) {
		if val != nil {
			logs[index] = val.(*observer.LoggedEntry)
			index--
		}
	})
	return logs[index+1 : BufferSize]
}

/*WriteLogs - write the logs to a io.Writer */
func (ml *MemLogger) WriteLogs(w io.Writer, detailLevel int) {
	cfg := zap.NewDevelopmentConfig()
	cfg.EncoderConfig.LevelKey = "level"
	cfg.EncoderConfig.NameKey = "name"
	cfg.EncoderConfig.MessageKey = "msg"
	cfg.EncoderConfig.CallerKey = "caller"

	if detailLevel >= IncludeStacktrace {
		cfg.EncoderConfig.StacktraceKey = "stacktrace"
	} else {
		cfg.EncoderConfig.StacktraceKey = ""
	}
	encoder := zapcore.NewConsoleEncoder(cfg.EncoderConfig)
	logs := ml.GetLogs()
	for idx := 0; idx < len(logs); idx++ {
		if logs[idx] != nil {
			ml.writeEntry(w, encoder, logs[idx], detailLevel)
		}
	}
}

func (ml *MemLogger) writeEntry(w io.Writer, encoder zapcore.Encoder, entry *observer.LoggedEntry, detailLevel int) {
	var fields []zapcore.Field
	if detailLevel >= IncludeFields {
		fields = entry.Context
	}
	buf, err := encoder.EncodeEntry(entry.Entry, fields)
	if err != nil {
		return
	}
	w.Write(buf.Bytes())
}

/*With - implement interface */
func (mc *MemCore) With(fields []zapcore.Field) zapcore.Core {
	clone := mc.clone()
	for i := range fields {
		fields[i].AddTo(clone.enc)
	}
	return clone
}

/*Check - implement interface */
func (mc *MemCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if mc.Enabled(ent.Level) {
		return ce.AddCore(ent, mc)
	}
	return ce
}

/*Write - implement interface */
func (mc *MemCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	mc.r.Value = &observer.LoggedEntry{
		Entry:   ent,
		Context: fields,
	}
	mc.r = mc.r.Next()
	return nil
}

/*Sync - implement interface */
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

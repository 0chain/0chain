package logging

import (
	"bytes"
	"container/ring"
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"0chain.net/core/viper"
)

func init() {
	viper.Set("logging.console", true)
	InitLogging("development")
}

func TestNewMemLogger(t *testing.T) {
	t.Parallel()

	logger := &MemLogger{
		core: &MemCore{
			LevelEnabler: nil,
			enc:          nil,
			r:            ring.New(BufferSize),
			mu:           &sync.RWMutex{},
		},
	}
	mc := logger.core
	for r := mc.r; r.Value == nil; r = r.Next() {
		r.Value = &observer.LoggedEntry{}
	}

	type args struct {
		enc  zapcore.Encoder
		enab zapcore.LevelEnabler
	}
	tests := []struct {
		name string
		args args
		want *MemLogger
	}{
		{
			name: "Test_NewMemLogger_OK",
			want: logger,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := NewMemLogger(tt.args.enc, tt.args.enab); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewMemLogger() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMemLogger_GetLogs(t *testing.T) {
	t.Parallel()

	ml := NewMemLogger(nil, nil)

	var index = BufferSize - 1
	mc := ml.core
	logs := make([]*observer.LoggedEntry, BufferSize)
	mc.r.Do(func(val interface{}) {
		if val != nil {
			logs[index] = val.(*observer.LoggedEntry)
			index--
		}
	})

	type fields struct {
		core *MemCore
	}
	tests := []struct {
		name   string
		fields fields
		want   []*observer.LoggedEntry
	}{
		{
			name:   "Test_MemLogger_GetLogs_OK",
			fields: fields{core: ml.core},
			want:   logs[index+1 : BufferSize],
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ml := &MemLogger{
				core: tt.fields.core,
			}
			if got := ml.GetLogs(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetLogs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMemLogger_WriteLogs_Lvl_2(t *testing.T) {
	t.Parallel()

	var (
		ml  = NewMemLogger(nil, nil)
		w   = bytes.NewBuffer(nil)
		cfg = zap.NewDevelopmentConfig()
	)
	cfg.EncoderConfig.LevelKey = "level"
	cfg.EncoderConfig.NameKey = "name"
	cfg.EncoderConfig.MessageKey = "msg"
	cfg.EncoderConfig.CallerKey = "caller"
	cfg.EncoderConfig.StacktraceKey = ""

	encoder := zapcore.NewConsoleEncoder(cfg.EncoderConfig)
	logs := ml.GetLogs()

	for idx := 0; idx < len(logs); idx++ {
		if logs[idx] != nil {
			fields := logs[idx].Context
			buf, err := encoder.EncodeEntry(logs[idx].Entry, fields)
			if err != nil {
				return
			}
			w.Write(buf.Bytes())
		}
	}

	type fields struct {
		core *MemCore
	}
	type args struct {
		detailLevel int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		wantW  string
	}{
		{
			name:   "Test_MemLogger_WriteLogs_OK",
			fields: fields{core: ml.core},
			args:   args{detailLevel: 2},
			wantW:  w.String(),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ml := &MemLogger{
				core: tt.fields.core,
			}
			w := &bytes.Buffer{}
			ml.WriteLogs(w, tt.args.detailLevel)
			if gotW := w.String(); !assert.Equal(t, gotW, tt.wantW) {
				t.Errorf("WriteLogs() = %v, want %v", gotW, tt.wantW)
			}
		})
	}
}

func TestMemLogger_WriteLogs_Lvl_3(t *testing.T) {
	t.Parallel()

	var (
		ml  = NewMemLogger(nil, nil)
		w   = bytes.NewBuffer(nil)
		cfg = zap.NewDevelopmentConfig()
	)
	cfg.EncoderConfig.LevelKey = "level"
	cfg.EncoderConfig.NameKey = "name"
	cfg.EncoderConfig.MessageKey = "msg"
	cfg.EncoderConfig.CallerKey = "caller"
	cfg.EncoderConfig.StacktraceKey = "stacktrace"

	encoder := zapcore.NewConsoleEncoder(cfg.EncoderConfig)
	logs := ml.GetLogs()

	for idx := 0; idx < len(logs); idx++ {
		if logs[idx] != nil {
			fields := logs[idx].Context
			buf, err := encoder.EncodeEntry(logs[idx].Entry, fields)
			if err != nil {
				return
			}
			w.Write(buf.Bytes())
		}
	}

	type fields struct {
		core *MemCore
	}
	type args struct {
		detailLevel int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		wantW  string
	}{
		{
			name:   "Test_MemLogger_WriteLogs_OK",
			fields: fields{core: ml.core},
			args:   args{detailLevel: 3},
			wantW:  w.String(),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ml := &MemLogger{
				core: tt.fields.core,
			}
			w := &bytes.Buffer{}
			ml.WriteLogs(w, tt.args.detailLevel)
			if gotW := w.String(); !assert.Equal(t, gotW, tt.wantW) {
				t.Errorf("WriteLogs() = %v, want %v", gotW, tt.wantW)
			}
		})
	}
}

func TestMemCore_With(t *testing.T) {
	t.Parallel()

	conf := zap.NewProductionConfig()
	mc := &MemCore{
		LevelEnabler: conf.Level,
		enc:          zapcore.NewConsoleEncoder(conf.EncoderConfig),
		r:            ring.New(BufferSize),
		mu:           &sync.RWMutex{},
	}

	f := make([]zapcore.Field, 1)
	clone := mc.clone()

	for ind := range f {
		f[ind].Type = zapcore.BoolType
		f[ind].AddTo(clone.enc)
	}

	type fields struct {
		LevelEnabler zapcore.LevelEnabler
		enc          zapcore.Encoder
		r            *ring.Ring
		mu           *sync.RWMutex
	}
	type args struct {
		fields []zapcore.Field
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   zapcore.Core
	}{
		{
			name: "Test_MemCore_With_OK",
			fields: fields{
				LevelEnabler: mc.LevelEnabler,
				enc:          mc.enc,
				r:            mc.r,
				mu:           mc.mu,
			},
			args: args{fields: f},
			want: clone,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := &MemCore{
				LevelEnabler: tt.fields.LevelEnabler,
				enc:          tt.fields.enc,
				r:            tt.fields.r,
				mu:           tt.fields.mu,
			}
			if got := mc.With(tt.args.fields); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("With() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMemCore_Check(t *testing.T) {
	t.Parallel()

	mc := &MemCore{
		LevelEnabler: zap.NewAtomicLevelAt(2),
		enc:          zapcore.NewConsoleEncoder(zap.NewProductionEncoderConfig()),
		r:            ring.New(BufferSize),
		mu:           &sync.RWMutex{},
	}
	ce := zapcore.CheckedEntry{}
	ent := zapcore.Entry{Level: 1}

	type fields struct {
		LevelEnabler zapcore.LevelEnabler
		enc          zapcore.Encoder
		r            *ring.Ring
		mu           *sync.RWMutex
	}
	type args struct {
		ent zapcore.Entry
		ce  *zapcore.CheckedEntry
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *zapcore.CheckedEntry
	}{
		{
			name: "Test_MemCore_Check_OK",
			fields: fields{
				LevelEnabler: mc.LevelEnabler,
				enc:          mc.enc,
				r:            mc.r,
				mu:           mc.mu,
			},
			args: args{
				ent: ent,
				ce:  &ce,
			},
			want: ce.AddCore(ent, mc),
		},
		{
			name: "Test_MemCore_Check_Low_Level_OK",
			fields: fields{
				LevelEnabler: zap.NewAtomicLevelAt(0),
				enc:          mc.enc,
				r:            mc.r,
				mu:           mc.mu,
			},
			args: args{
				ent: ent,
				ce:  &ce,
			},
			want: &ce,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := &MemCore{
				LevelEnabler: tt.fields.LevelEnabler,
				enc:          tt.fields.enc,
				r:            tt.fields.r,
				mu:           tt.fields.mu,
			}
			if got := mc.Check(tt.args.ent, tt.args.ce); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Check() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMemCore_Sync(t *testing.T) {
	t.Parallel()

	mc := &MemCore{
		LevelEnabler: zap.NewAtomicLevelAt(2),
		enc:          zapcore.NewConsoleEncoder(zap.NewProductionEncoderConfig()),
		r:            ring.New(BufferSize),
		mu:           &sync.RWMutex{},
	}

	type fields struct {
		LevelEnabler zapcore.LevelEnabler
		enc          zapcore.Encoder
		r            *ring.Ring
		mu           *sync.RWMutex
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Test_MemCore_Sync_OK",
			fields: fields{
				LevelEnabler: mc.LevelEnabler,
				enc:          mc.enc,
				r:            mc.r,
				mu:           mc.mu,
			},
			wantErr: false, // not implemented
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := &MemCore{
				LevelEnabler: tt.fields.LevelEnabler,
				enc:          tt.fields.enc,
				r:            tt.fields.r,
				mu:           tt.fields.mu,
			}
			if err := mc.Sync(); (err != nil) != tt.wantErr {
				t.Errorf("Sync() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMemCore_Write(t *testing.T) {
	t.Parallel()

	mc := &MemCore{
		LevelEnabler: zap.NewAtomicLevelAt(2),
		enc:          zapcore.NewConsoleEncoder(zap.NewProductionEncoderConfig()),
		r:            ring.New(BufferSize),
		mu:           &sync.RWMutex{},
	}

	type fields struct {
		LevelEnabler zapcore.LevelEnabler
		enc          zapcore.Encoder
		r            *ring.Ring
		mu           *sync.RWMutex
	}
	type args struct {
		ent    zapcore.Entry
		fields []zapcore.Field
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Test_MemCore_Write_OK",
			fields: fields{
				LevelEnabler: mc.LevelEnabler,
				enc:          mc.enc,
				r:            mc.r,
				mu:           mc.mu,
			},
			args: args{ent: zapcore.Entry{}, fields: make([]zapcore.Field, 0)},
		},
		{
			name: "Test_MemCore_Write_Nil_r_OK",
			fields: fields{
				LevelEnabler: mc.LevelEnabler,
				enc:          mc.enc,
				r:            mc.r,
				mu:           mc.mu,
			},
			args: args{ent: zapcore.Entry{}, fields: make([]zapcore.Field, 0)},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := &MemCore{
				LevelEnabler: tt.fields.LevelEnabler,
				enc:          tt.fields.enc,
				r:            tt.fields.r,
				mu:           tt.fields.mu,
			}
			if err := mc.Write(tt.args.ent, tt.args.fields); (err != nil) != tt.wantErr {
				t.Errorf("Write() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

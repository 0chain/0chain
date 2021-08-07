package util

import (
	"io"
	"testing"

	"github.com/0chain/errors"
)

type testWriter struct{}

func (t testWriter) Write(_ []byte) (n int, err error) {
	return 0, errors.New("error")
}

var _ io.Writer = (*testWriter)(nil)

func TestOriginTracker_Write(t *testing.T) {
	t.Parallel()

	type fields struct {
		Origin  Sequence
		Version Sequence
	}
	tests := []struct {
		name    string
		arg     io.Writer
		fields  fields
		wantErr bool
	}{
		{
			name:    "Test_OriginTracker_Write_ERR",
			arg:     &testWriter{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			o := &OriginTracker{
				Origin:  tt.fields.Origin,
				Version: tt.fields.Version,
			}

			err := o.Write(tt.arg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Write() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

type testReader struct{}

func (t testReader) Read(_ []byte) (n int, err error) {
	return 0, errors.New("error")
}

var _ io.Reader = (*testReader)(nil)

func TestOriginTracker_Read(t *testing.T) {
	t.Parallel()

	type fields struct {
		Origin  Sequence
		Version Sequence
	}
	type args struct {
		r io.Reader
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "Test_OriginTracker_Write_ERR",
			args:    args{r: &testReader{}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			o := &OriginTracker{
				Origin:  tt.fields.Origin,
				Version: tt.fields.Version,
			}
			if err := o.Read(tt.args.r); (err != nil) != tt.wantErr {
				t.Errorf("Read() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

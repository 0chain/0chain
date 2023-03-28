package datastore

import (
	"reflect"
	"testing"
	"time"

	"0chain.net/core/common"
)

func TestCreationDateField_GetCreationTime(t *testing.T) {
	t.Parallel()

	ts := common.Now()

	type fields struct {
		CreationDate common.Timestamp
	}
	tests := []struct {
		name   string
		fields fields
		want   common.Timestamp
	}{
		{
			name:   "TestCreationDateField_GetCreationTime_OK",
			fields: fields{CreationDate: ts},
			want:   ts,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cd := &CreationDateField{
				CreationDate: tt.fields.CreationDate,
			}
			if got := cd.GetCreationTime(); got != tt.want {
				t.Errorf("GetCreationTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreationDateField_ToTime(t *testing.T) {
	t.Parallel()

	ts := common.Now()

	type fields struct {
		CreationDate common.Timestamp
	}
	tests := []struct {
		name   string
		fields fields
		want   time.Time
	}{
		{
			name:   "TestCreationDateField_ToTime_OK",
			fields: fields{CreationDate: ts},
			want:   time.Unix(int64(ts), 0),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cd := &CreationDateField{
				CreationDate: tt.fields.CreationDate,
			}
			if got := cd.ToTime(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ToTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

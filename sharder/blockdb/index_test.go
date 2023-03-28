package blockdb

import (
	"bytes"
	"io"
	"reflect"
	"testing"
)

func Test_mapIndex_GetOffset(t *testing.T) {
	t.Parallel()

	index := map[Key]int64{
		"1": 1,
		"2": 2,
	}

	type fields struct {
		index map[Key]int64
	}
	type args struct {
		key Key
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int64
		wantErr bool
	}{
		{
			name:    "Test_mapIndex_GetOffset_OK",
			fields:  fields{index: index},
			args:    args{"1"},
			want:    1,
			wantErr: false,
		},
		{
			name:    "Test_mapIndex_GetOffset_Unknown_Key_ERR",
			fields:  fields{index: index},
			args:    args{"11"},
			want:    -1,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mi := &mapIndex{
				index: tt.fields.index,
			}
			got, err := mi.GetOffset(tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetOffset() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetOffset() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_mapIndex_Decode(t *testing.T) {
	t.Parallel()

	mi := &mapIndex{
		index: map[Key]int64{
			"1": 1,
			"2": 2,
		},
	}
	buf := bytes.Buffer{}
	if err := mi.Encode(&buf); err != nil {
		t.Fatal(err)
	}

	type fields struct {
		index map[Key]int64
	}
	type args struct {
		reader io.Reader
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *mapIndex
		wantErr bool
	}{
		{
			name:    "Test_mapIndex_Decode_OK",
			fields:  fields{index: mi.index},
			args:    args{reader: &buf},
			want:    mi,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mi := &mapIndex{
				index: tt.fields.index,
			}
			if err := mi.Decode(tt.args.reader); (err != nil) != tt.wantErr {
				t.Errorf("Decode() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && !reflect.DeepEqual(mi, tt.want) {
				t.Errorf("Decode() got = %v, want %v", mi, tt.want)
			}
		})
	}
}

func Test_mapIndex_GetKeys(t *testing.T) {
	t.Parallel()

	mi := &mapIndex{
		index: map[Key]int64{
			"1": 1,
			"2": 2,
		},
	}
	buf := bytes.Buffer{}
	if err := mi.Encode(&buf); err != nil {
		t.Fatal(err)
	}

	type fields struct {
		index map[Key]int64
	}
	tests := []struct {
		name   string
		fields fields
		want   []Key
	}{
		{
			name:   "Test_mapIndex_GetKeys_OK",
			fields: fields{index: mi.index},
			want: []Key{
				"1",
				"2",
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mi := &mapIndex{
				index: tt.fields.index,
			}
			if got := mi.GetKeys(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetKeys() = %v, want %v", got, tt.want)
			}
		})
	}
}

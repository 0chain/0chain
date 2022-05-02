package common

import (
	"reflect"
	"testing"

	"github.com/golang/snappy"
	"github.com/stretchr/testify/require"
	"github.com/valyala/gozstd"
)

func TestNewSnappyCompDe(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want *SnappyCompDe
	}{
		{
			name: "Test_NewSnappyCompDe_OK",
			want: &SnappyCompDe{},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := NewSnappyCompDe(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewSnappyCompDe() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSnappyCompDe_Compress(t *testing.T) {
	t.Parallel()

	data := []byte("data")

	type args struct {
		data []byte
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "Test_SnappyCompDe_Compress_OK",
			args: args{data: data},
			want: snappy.Encode(nil, data),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			scd := &SnappyCompDe{}
			if got := scd.Compress(tt.args.data); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Compress() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSnappyCompDe_Decompress(t *testing.T) {
	t.Parallel()

	want := []byte("data")
	enc := snappy.Encode(nil, want)

	type args struct {
		data []byte
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name:    "Test_SnappyCompDe_Decompress_OK",
			args:    args{data: enc},
			want:    want,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			scd := &SnappyCompDe{}
			got, err := scd.Decompress(tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Decompress() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Decompress() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSnappyCompDe_Encoding(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want string
	}{
		{
			name: "Test_SnappyCompDe_Encoding_OK",
			want: "snappy",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			scd := &SnappyCompDe{}
			if got := scd.Encoding(); got != tt.want {
				t.Errorf("Encoding() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewZStdCompDe(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want *ZStdCompDe
	}{
		{
			name: "Test_NewZStdCompDe_OK",
			want: &ZStdCompDe{},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := NewZStdCompDe(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewZStdCompDe() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestZStdCompDe_SetLevel(t *testing.T) {
	t.Parallel()

	type fields struct {
		level int
	}
	type args struct {
		level int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   ZStdCompDe
	}{
		{
			name:   "Test_ZStdCompDe_SetLevel_OK",
			fields: fields{level: 1},
			args:   args{level: 2},
			want: ZStdCompDe{
				level: 2,
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			zstd := &ZStdCompDe{
				level: tt.fields.level,
			}

			zstd.SetLevel(tt.args.level)
			if !reflect.DeepEqual(*zstd, tt.want) {
				t.Errorf("SetLevel() got = %v, want = %v", zstd, tt.want)
			}
		})
	}
}

func TestZStdCompDe_Compress(t *testing.T) {
	t.Parallel()

	data := []byte("data")

	type fields struct {
		level int
	}
	type args struct {
		data []byte
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []byte
	}{
		{
			name:   "Test_ZStdCompDe_Compress_Level_0_OK",
			fields: fields{level: 0},
			args:   args{data: data},
			want:   gozstd.Compress(nil, data),
		},
		{
			name:   "Test_ZStdCompDe_Compress_Level_1_OK",
			fields: fields{level: 1},
			args:   args{data: data},
			want:   gozstd.CompressLevel(nil, data, 1),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			zstd := &ZStdCompDe{
				level: tt.fields.level,
			}
			got, err := zstd.Compress(tt.args.data)
			require.NoError(t, err)

			require.Equal(t, tt.want, got)
		})
	}
}

func TestZStdCompDe_Decompress(t *testing.T) {
	t.Parallel()

	comprData := gozstd.Compress(nil, []byte("data"))
	want, err := gozstd.Decompress(nil, comprData)
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		level int
	}
	type args struct {
		data []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name:    "Test_ZStdCompDe_Decompress_OK",
			args:    args{data: comprData},
			want:    want,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			zstd := &ZStdCompDe{
				level: tt.fields.level,
			}
			got, err := zstd.Decompress(tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Decompress() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Decompress() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestZStdCompDe_Encoding(t *testing.T) {
	t.Parallel()

	type fields struct {
		level int
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "Test_ZStdCompDe_Encoding_OK",
			want: "zstd",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			zstd := &ZStdCompDe{
				level: tt.fields.level,
			}
			if got := zstd.Encoding(); got != tt.want {
				t.Errorf("Encoding() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewZStdCompDeWithDict(t *testing.T) {
	t.Parallel()

	dict := []byte("abcde")
	cDict, err := gozstd.NewCDict(dict)
	if err != nil {
		t.Fatal(err)
	}
	dDict, err := gozstd.NewDDict(dict)
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		dict []byte
	}
	tests := []struct {
		name    string
		args    args
		want    *ZStdDictCompDe
		wantErr bool
	}{
		{
			name: "Test_NewZStdCompDeWithDict_OK",
			args: args{dict: dict},
			want: &ZStdDictCompDe{
				cdict: cDict,
				ddict: dDict,
			},
			wantErr: false,
		},
		{
			name:    "Test_NewZStdCompDeWithDict_CDict_ERR",
			args:    args{dict: []byte{}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewZStdCompDeWithDict(tt.args.dict)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewZStdCompDeWithDict() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewZStdCompDeWithDict() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestZStdDictCompDe_Compress(t *testing.T) {
	t.Parallel()

	dict := []byte("abcde")
	cDict, err := gozstd.NewCDict(dict)
	if err != nil {
		t.Fatal(err)
	}
	data := []byte("data")

	type fields struct {
		cdict *gozstd.CDict
		ddict *gozstd.DDict
	}
	type args struct {
		data []byte
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []byte
	}{
		{
			name:   "Test_ZStdDictCompDe_Compress_OK",
			fields: fields{cdict: cDict},
			args:   args{data: data},
			want:   gozstd.CompressDict(nil, data, cDict),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			zstd := &ZStdDictCompDe{
				cdict: tt.fields.cdict,
				ddict: tt.fields.ddict,
			}
			if got := zstd.Compress(tt.args.data); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Compress() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestZStdDictCompDe_Decompress(t *testing.T) {
	t.Parallel()

	dict := []byte("abcde")
	cDict, err := gozstd.NewCDict(dict)
	if err != nil {
		t.Fatal(err)
	}
	dDict, err := gozstd.NewDDict(dict)
	if err != nil {
		t.Fatal(err)
	}

	data := []byte("data")
	compData := gozstd.CompressDict(nil, data, cDict)

	type fields struct {
		cdict *gozstd.CDict
		ddict *gozstd.DDict
	}
	type args struct {
		data []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name:    "Test_ZStdDictCompDe_Decompress_OK",
			fields:  fields{cdict: cDict, ddict: dDict},
			args:    args{data: compData},
			want:    data,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			zstd := &ZStdDictCompDe{
				cdict: tt.fields.cdict,
				ddict: tt.fields.ddict,
			}
			got, err := zstd.Decompress(tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Decompress() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Decompress() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestZStdDictCompDe_Encoding(t *testing.T) {
	t.Parallel()

	type fields struct {
		cdict *gozstd.CDict
		ddict *gozstd.DDict
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "Test_ZStdDictCompDe_Encoding_OK",
			want: "zstddict",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			zstd := &ZStdDictCompDe{
				cdict: tt.fields.cdict,
				ddict: tt.fields.ddict,
			}
			if got := zstd.Encoding(); got != tt.want {
				t.Errorf("Encoding() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewZLibCompDe(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want *ZLibCompDe
	}{
		{
			name: "Test_NewZLibCompDe_OK",
			want: &ZLibCompDe{},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := NewZLibCompDe(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewZLibCompDe() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestZLibCompDe_Decompress(t *testing.T) {
	t.Parallel()

	data := []byte("data")
	cd := &ZLibCompDe{}
	compData, err := cd.Compress(data)
	require.NoError(t, err)

	// compressing with unknown dict
	cDict, err := gozstd.NewCDict([]byte("abcde"))
	if err != nil {
		t.Fatal(err)
	}
	invCompr := gozstd.CompressDict(nil, data, cDict)

	type args struct {
		data []byte
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name:    "Test_ZLibCompDe_Decompress_OK",
			args:    args{data: compData},
			want:    data,
			wantErr: false,
		},
		{
			name:    "Test_ZLibCompDe_Decompress_OK",
			args:    args{data: invCompr},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			zlibcd := &ZLibCompDe{}
			got, err := zlibcd.Decompress(tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Decompress() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Decompress() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestZLibCompDe_Encoding(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want string
	}{
		{
			name: "Test_ZLibCompDe_Encoding_OK",
			want: "zlib",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			zlibcd := &ZLibCompDe{}
			if got := zlibcd.Encoding(); got != tt.want {
				t.Errorf("Encoding() = %v, want %v", got, tt.want)
			}
		})
	}
}

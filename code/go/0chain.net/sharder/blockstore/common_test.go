package blockstore

import (
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"testing"

	"0chain.net/sharder/mocks"
)

func Test_countFiles(t *testing.T) {
	dirPath := t.TempDir()
	dirPrefix := "dirPrefix"
	countTempFiles, _, err := mocks.CommonMock(dirPath, dirPrefix, 10)

	if err != nil {
		log.Fatal(err)
	}
	path := filepath.Join(dirPath, dirPrefix+"0", "1")
	wrongDir := filepath.Join(path, "10")

	tests := [2]struct {
		name  string
		path  string
		want  int
		error bool
	}{
		{
			name:  "OK",
			path:  path,
			want:  int(countTempFiles / 100),
			error: false,
		},
		{
			name:  "Wrong Dir",
			path:  wrongDir,
			want:  0,
			error: true,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := countFiles(test.path)
			if (err != nil) != test.error {
				t.Errorf("countFiles error: %v | want: %v", err, test.error)
			}
			if got != test.want {
				t.Errorf("countFiles got: %v | want: %v", got, test.want)
			}
		})
	}
}

func Benchmark_countFiles(b *testing.B) {
	dirPath := b.TempDir()
	dirPrefix := "dirPrefix"
	_, _, err := mocks.CommonMock(dirPath, dirPrefix, 10)
	if err != nil {
		log.Fatal(err)
	}
	path := filepath.Join(dirPath, dirPrefix+"0", "1")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = countFiles(path)
	}
}

func Test_getAvailableSizeAndInodes(t *testing.T) {
	t.Skip()
}

func Benchmark_getAvailableSizeAndInodes(b *testing.B) {
	dirPath := b.TempDir()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _, _ = getAvailableSizeAndInodes(dirPath)
	}
}

func Test_getCurIndexes(t *testing.T) {
	curKInd := 1
	curDirInd := 10
	dirPath := t.TempDir()

	tempFile := filepath.Join(dirPath, "tempFile")
	f, err := os.Create(tempFile)
	if err != nil {
		log.Fatal(err)
	}
	_, err = f.Write([]byte(fmt.Sprintf("%v\n%v", curKInd, curDirInd)))

	tempFileWithOutCurKInd := filepath.Join(dirPath, "tempFileWithOutCurKInd")
	f, err = os.Create(tempFileWithOutCurKInd)
	if err != nil {
		log.Fatal(err)
	}
	tempFileWithOutCurDirInd := filepath.Join(dirPath, "tempFileWithOutCurDirInd")

	f, err = os.Create(tempFileWithOutCurDirInd)
	if err != nil {
		log.Fatal(err)
	}
	_, err = f.Write([]byte(fmt.Sprintf("%v\n", curDirInd)))

	tests := [4]struct {
		name          string
		fPath         string
		wantCurKInd   int
		wantCurDirInd int
		error         bool
	}{
		{
			name:          "OK",
			fPath:         tempFile,
			wantCurKInd:   curKInd,
			wantCurDirInd: curDirInd,
			error:         false,
		},
		{
			name:          "Not File",
			fPath:         dirPath,
			wantCurKInd:   0,
			wantCurDirInd: 0,
			error:         true,
		},
		{
			name:          "Missing curKInd",
			fPath:         tempFileWithOutCurKInd,
			wantCurKInd:   0,
			wantCurDirInd: 0,
			error:         true,
		},
		{
			name:          "Missing curDirInd",
			fPath:         tempFileWithOutCurDirInd,
			wantCurKInd:   0,
			wantCurDirInd: 0,
			error:         true,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			gotCurKInd, gotCurDirInd, err := getCurIndexes(test.fPath)
			if (err != nil) != test.error {
				t.Errorf("getCurIndexes error %v | want %v", err, test.error)
			}
			if gotCurKInd != test.wantCurKInd {
				t.Errorf("getCurIndexes got %v | want %v", gotCurKInd, test.wantCurKInd)
			}
			if gotCurDirInd != test.wantCurDirInd {
				t.Errorf("getCurIndexes got %v | want %v", gotCurDirInd, test.wantCurDirInd)
			}
		})
	}
}

func Benchmark_getCurIndexes(b *testing.B) {
	dirPath := b.TempDir()
	tempFile := filepath.Join(dirPath, "tempFile")
	_, err := os.Create(tempFile)
	if err != nil {
		log.Fatal(err)
	}

	curKInd := 1
	curDirInd := 10
	_ = updateCurIndexes(tempFile, curKInd, curDirInd)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = getCurIndexes(tempFile)
	}
}

func Test_getCurrentDirBlockNums(t *testing.T) {
	dirEntries := t.TempDir()
	dirPrefix := "dirPrefix"
	countTempFiles, _, err := mocks.CommonMock(dirEntries, dirPrefix, 10)

	if err != nil {
		log.Fatal(err)
	}
	path := filepath.Join(dirEntries, dirPrefix+"0", "1")
	wrongDir := filepath.Join(path, "10")

	tests := [2]struct {
		name  string
		path  string
		want  int
		error bool
	}{
		{
			name:  "OK",
			path:  path,
			want:  int(countTempFiles / 100),
			error: false,
		},
		{
			name:  "Wrong Dir",
			path:  wrongDir,
			want:  0,
			error: true,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := countFiles(test.path)
			if (err != nil) != test.error {
				t.Errorf("countFiles error: %v | want: %v", err, test.error)
			}
			if got != test.want {
				t.Errorf("countFiles got: %v | want: %v", got, test.want)
			}
		})
	}
}

func Benchmark_getCurrentDirBlockNums(b *testing.B) {
	dirPath := b.TempDir()
	dirPrefix := "dirPrefix"
	_, _, err := mocks.CommonMock(dirPath, dirPrefix, 10)
	if err != nil {
		log.Fatal(err)
	}
	path := filepath.Join(dirPath, dirPrefix+"0", "1")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = getCurrentDirBlockNums(path)
	}
}

func Test_updateCurIndexes(t *testing.T) {
	curKInd := 1
	curDirInd := 10
	curBlockNums := 100
	dirPath := t.TempDir()
	tempFile := filepath.Join(dirPath, "tempFile")
	wrongPath := filepath.Join(dirPath, "wrongPath")
	tempWrongFile := filepath.Join(wrongPath, "tempWrongFile")
	_ = os.Mkdir(wrongPath, 0111)

	tests := [2]struct {
		name             string
		dirPath          string
		wantCurKInd      int
		wantCurDirInd    int
		wantCurBlockNums int
		error            bool
	}{
		{
			name:             "OK",
			dirPath:          tempFile,
			wantCurKInd:      curKInd,
			wantCurDirInd:    curDirInd,
			wantCurBlockNums: curBlockNums,
			error:            false,
		},
		{
			name:             "Wrong Path",
			dirPath:          tempWrongFile,
			wantCurKInd:      0,
			wantCurDirInd:    0,
			wantCurBlockNums: 0,
			error:            true,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := updateCurIndexes(test.dirPath, test.wantCurKInd, test.wantCurDirInd)
			if (err != nil) != test.error {
				t.Errorf("updateCurIndexes error %v | want %v", err, test.error)
			}
			gotCurKInd, gotCurDirInd, _ := getCurIndexes(test.dirPath)
			if gotCurKInd != test.wantCurKInd {
				t.Errorf("updateCurIndexes got %v | want %v", gotCurKInd, test.wantCurKInd)
			}
			if gotCurDirInd != test.wantCurDirInd {
				t.Errorf("updateCurIndexes got %v | want %v", gotCurDirInd, test.wantCurDirInd)
			}
		})
	}
}

func Benchmark_updateCurIndexes(b *testing.B) {
	dirPath := b.TempDir()
	tempFile := filepath.Join(dirPath, "tempFile")
	_, err := os.Create(tempFile)
	if err != nil {
		log.Fatal(err)
	}

	curKInd := 1
	curDirInd := 10

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = updateCurIndexes(tempFile, curKInd, curDirInd)
	}
}

func Test_countBlocksInVolumes(t *testing.T) {
	dirPath := t.TempDir()
	dirPrefix := "dirPrefix"
	dcl := 10
	countTempFiles, size, err := mocks.CommonMock(dirPath, dirPrefix, 10)
	if err != nil {
		log.Fatal(err)
	}

	tests := [1]struct {
		name      string
		path      string
		dirPrefix string
		dcl       int
		wantCount uint64
		wantSize  uint64
	}{
		{
			name:      "OK",
			path:      dirPath,
			dirPrefix: dirPrefix,
			dcl:       dcl,
			wantCount: countTempFiles,
			wantSize:  size,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			gotSize, gotCount := countBlocksInVolumes(test.path, test.dirPrefix, test.dcl)
			if gotCount != test.wantCount {
				t.Errorf("countBlocksInVolumes got %v | want %v", gotCount, test.wantCount)
			}
			if gotSize != test.wantSize {
				t.Errorf("countBlocksInVolumes got %v | want %v", gotSize, test.wantSize)
			}
		})
	}
}

func Benchmark_countBlocksInVolumes(b *testing.B) {
	dirPath := b.TempDir()
	dirPrefix := "dirPrefix"
	dcl := 10
	_, _, err := mocks.CommonMock(dirPath, dirPrefix, dcl)
	if err != nil {
		log.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = countBlocksInVolumes(dirPath, dirPrefix, dcl)
	}
}

func Test_getUint64ValueFromYamlConfig(t *testing.T) {
	var (
		intInterface                             interface{} = 123100
		stringWithLidSeparator                   interface{} = "123^100"
		stringWithLidSeparatorAndWithSpace       interface{} = "1 23^100"
		stringWithLidSeparatorAndWithOutNumInLhs interface{} = "abc^100"
		stringWithLidSeparatorAndWithOutNumInRhs interface{} = "123^abc"
		stringWithMultSeparator                  interface{} = "123*100"
		stringWithMultSeparatorWithOutNumInLhs   interface{} = "abc*100"
		stringWithMultSeparatorWithOutNumInRhs   interface{} = "123*abc"
		UnSupportedType                          interface{} = 1.2300
	)

	tests := [9]struct {
		name  string
		value interface{}
		want  uint64
		error bool
	}{
		{
			name:  "OK",
			value: intInterface,
			want:  123100,
			error: false,
		},
		{
			name:  "With '^' Separator",
			value: stringWithLidSeparator,
			want:  uint64(math.Pow(123, 100)),
			error: false,
		},
		{
			name:  "With '^' Separator And Space",
			value: stringWithLidSeparatorAndWithSpace,
			want:  uint64(math.Pow(123, 100)),
			error: false,
		},
		{
			name:  "With '^' Separator And WithOut Number In Lhs And ",
			value: stringWithLidSeparatorAndWithOutNumInLhs,
			want:  0,
			error: true,
		},
		{
			name:  "With '^' Separator And WithOut Number In Rhs",
			value: stringWithLidSeparatorAndWithOutNumInRhs,
			want:  0,
			error: true,
		},
		{
			name:  "With '*' Separator",
			value: stringWithMultSeparator,
			want:  12300,
			error: false,
		},
		{
			name:  "With '*' Separator WithOut Number In Lhs",
			value: stringWithMultSeparatorWithOutNumInLhs,
			want:  0,
			error: true,
		},
		{
			name:  "With '*' Separator WithOut Number In Rhs",
			value: stringWithMultSeparatorWithOutNumInRhs,
			want:  0,
			error: true,
		},
		{
			name:  "Unsupported Type",
			value: UnSupportedType,
			want:  0,
			error: true,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := getUint64ValueFromYamlConfig(test.value)
			if (err != nil) != test.error {
				t.Errorf("getUint64ValueFromYamlConfig error %v | want %v", err, test.error)
			}
			if got != test.want {
				t.Errorf("getUint64ValueFromYamlConfig got %v | want %v", got, test.want)
			}
		})
	}
}

func Benchmark_getUint64ValueFromYamlConfig(b *testing.B) {
	var (
		intInterface            interface{} = 123100
		stringWithLidSeparator  interface{} = "123^100"
		stringWithMultSeparator interface{} = "123*100"
	)

	tests := [3]struct {
		name  string
		value interface{}
	}{
		{
			name:  "Interface Type int",
			value: intInterface,
		},
		{
			name:  "With '^' Separator",
			value: stringWithLidSeparator,
		},
		{
			name:  "With '*' Separator",
			value: stringWithMultSeparator,
		},
	}

	for idx := range tests {
		test := tests[idx]
		b.Run(test.name, func(b *testing.B) {

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = getUint64ValueFromYamlConfig(test.value)
			}
		})
	}
}

func Test_getVolumePathFromBlockPath(t *testing.T) {
	tests := [2]struct {
		name string
		path string
		want string
	}{
		{
			name: "OK Path",
			path: "/path/to/blocks/HK0/199/blockname.dat",
			want: "/path/to/blocks",
		},
		{
			name: "OK Another Path",
			path: "/another/path/to/blocks/HK0/199/blockname.dat",
			want: "/another/path/to/blocks",
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := getVolumePathFromBlockPath(test.path); got != test.want {
				t.Errorf("getVolumePathFromBlockPath got %v | want %v", got, test.want)
			}
		})
	}
}

func Benchmark_getVolumePathFromBlockPath(b *testing.B) {
	tests := [2]struct {
		name string
		path string
		want string
	}{
		{
			name: "OK Path",
			path: "/path/to/blocks/HK0/199/blockname.dat",
			want: "/path/to/blocks",
		},
		{
			name: "OK Another Path",
			path: "/another/path/to/blocks/HK0/199/blockname.dat",
			want: "/another/path/to/blocks",
		},
	}

	for idx := range tests {
		test := tests[idx]
		b.Run(test.name, func(b *testing.B) {
			_ = getVolumePathFromBlockPath(test.path)
		})
	}
}

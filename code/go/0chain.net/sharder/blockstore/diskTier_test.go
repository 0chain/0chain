package blockstore

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"

	"golang.org/x/sys/unix"

	b "0chain.net/chaincore/block"
	"0chain.net/core/logging"
)

func Benchmark_DiskTier_removeSelectedVolume(b *testing.B) {
	// Call it only manually with an adequate number of tests.
	// Example use:  go test -bench=Benchmark_removeSelectedVolume -benchmem -benchtime=5000x  -tags bn256

	b.Skip()
	path := b.TempDir()
	volumes := mockTierVolumes(path, b.N)
	unableVolumes = make(map[string]*volume)

	d := diskTier{
		Volumes:    volumes,
		PrevVolInd: b.N - 1,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		d.removeSelectedVolume()
	}
}

func Test_DiskTier_removeSelectedVolume(t *testing.T) {
	t.Parallel()

	path := t.TempDir()
	volumes := mockTierVolumes(path, 4)
	volumeWithoutMiddle := append(volumes[:1], volumes[2:]...)
	volumeWithoutStart := append(volumeWithoutMiddle[1:])
	volumeWithoutEnd := append(volumeWithoutStart[:1])
	unableVolumes = make(map[string]*volume)
	dTier := diskTier{
		Volumes: volumes,
	}

	tests := [3]struct {
		name       string
		prevVolInd int
		want       diskTier
		wantVolume []*volume
	}{
		{
			name:       "Delete from Middle",
			prevVolInd: 1,
			wantVolume: volumeWithoutMiddle,
		},
		{
			name:       "Delete from start",
			prevVolInd: 0,
			wantVolume: volumeWithoutStart,
		},
		{
			name:       "Delete from End",
			prevVolInd: 1,
			wantVolume: volumeWithoutEnd,
		},
	}

	for idx := range tests {
		test := tests[idx]

		t.Run(test.name, func(t *testing.T) {
			dTier.PrevVolInd = test.prevVolInd
			dTier.removeSelectedVolume()
			if !reflect.DeepEqual(dTier.Volumes, test.wantVolume) {
				t.Errorf("removeSelectedVolume() got %v | want %v", dTier.Volumes, test.wantVolume)
			}
		})
	}
}

func Benchmark_volumeInit(b *testing.B) {
	b.Skip()
	hotCgf := mockHotConfig(b)
	hCfg := hotCgf.Sub("hot")
	logging.InitLogging("")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = volumeInit("hot", hCfg, "start")
	}
}

func Benchmark_volume_delete(b *testing.B) {
	// Call it only manually with an adequate number of tests.
	// Example use:  go test -bench=Benchmark_Volume_delete -benchmem -benchtime=1000x  -tags bn256
	b.Skip()
	var path string
	tmpPath := b.TempDir()
	dirPrefix := "DirPrefix"
	fileName := "fileName"
	dcl := 3
	var files []string

	_, _, _ = mockFileSystem(tmpPath, dirPrefix, fileName, dcl)
	vol := mockVolume(tmpPath, dcl-1, dcl-1, dcl-1)
	dTier := &diskTier{
		DirPrefix: dirPrefix,
	}

	for i := 0; i < b.N; i++ {
		block := mockBlock()
		data, err := json.Marshal(block)
		if err != nil {
			b.Error(err)
		}
		path, err = vol.write(block, data, dTier)
		if err != nil {
			b.Error(err)
		}
		files = append(files, path)
	}

	for i := 0; i < b.N; i++ {
		p := files[i]

		b.ResetTimer()
		if err := vol.delete("", p); err != nil {
			b.Error(err)
		}
	}
}

func Test_volume_delete(t *testing.T) {
	t.Parallel()

	tmpPath := t.TempDir()
	path := filepath.Join(tmpPath, "1")
	if err := os.MkdirAll(path, 0777); err != nil {
		t.Fatal("test volume delete", err)
	}

	var volStat unix.Statfs_t
	if err := unix.Statfs(tmpPath, &volStat); err != nil {
		t.Fatal("test volume delete", err)
	}
	vol := volume{BlocksCount: 1, BlocksSize: uint64(volStat.Bsize)}

	tests := [2]struct {
		name      string
		vol       volume
		path      string
		wantCount uint64
		wantSize  uint64
		error     bool
	}{
		{
			name:      "OK",
			vol:       vol,
			path:      path,
			wantCount: 0,
			wantSize:  0,
			error:     false,
		},
		{
			name:      "Path Not Exist",
			vol:       vol,
			path:      filepath.Join(tmpPath, "2"),
			wantCount: 1,
			wantSize:  uint64(volStat.Bsize),
			error:     true,
		},
	}

	for idx := range tests {
		test := tests[idx]

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if err := test.vol.delete("", test.path); (err != nil) != test.error {
				t.Errorf("delete() error: %v | want: %v", err, test.error)
			}
			if test.vol.BlocksSize != test.wantSize {
				t.Errorf("delete() BlocksSize: %v | want: %v", test.vol.BlocksSize, test.wantSize)
			}
			if test.vol.BlocksCount != test.wantCount {
				t.Errorf("delete() BlocksCount: %v | want: %v", test.vol.BlocksCount, test.wantCount)
			}
		})
	}

}

func Benchmark_volume_isAbleToStoreBlock(b *testing.B) {
	logging.InitLogging("")
	tmpPath := b.TempDir()
	dirPrefix := "DirPrefix"
	fileName := "fileName"
	dcl := 3

	_, _, _ = mockFileSystem(tmpPath, dirPrefix, fileName, dcl-1)
	dTier := &diskTier{
		DCL:       dcl,
		DirPrefix: dirPrefix,
	}
	vol := volume{
		Path:                tmpPath,
		AllowedBlockSize:    2,
		BlocksSize:          1,
		AllowedBlockNumbers: 2,
		BlocksCount:         1,
		InodesToMaintain:    100,

		CurKInd:         dcl - 2,
		CurDirInd:       dcl - 1,
		CurDirBlockNums: dcl,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = vol.isAbleToStoreBlock(dTier)
	}
}

func Test_volume_isAbleToStoreBlock(t *testing.T) {
	// Do not use parallel test execution.
	// Disk space is used.

	logging.InitLogging("")
	tmpPath := t.TempDir()
	dirPrefix := "DirPrefix"
	fileName := "fileName"
	dcl := 3

	var volStat unix.Statfs_t
	if err := unix.Statfs(tmpPath, &volStat); err != nil {
		t.Fatal("test volume delete", err)
	}

	_, _, _ = mockFileSystem(tmpPath, dirPrefix, fileName, dcl)

	tests := [13]struct {
		name  string
		dTier diskTier
		vol   volume
		want  bool
	}{
		{
			name:  "OK",
			dTier: diskTier{DCL: dcl},
			vol: volume{
				Path:             tmpPath,
				AllowedBlockSize: 2,
				BlocksSize:       1,
				BlocksCount:      1,
				InodesToMaintain: volStat.Ffree - 100,
				SizeToMaintain:   1,
				CurDirBlockNums:  1,
			},
			want: true,
		},
		{
			name:  "AllowedBlockSize == 0",
			dTier: diskTier{DCL: dcl},
			vol: volume{
				Path:             tmpPath,
				AllowedBlockSize: 0,
				BlocksSize:       0,
			},
			want: true,
		},
		{
			name:  "BlocksSize == AllowedBlockSize",
			dTier: diskTier{DCL: dcl},
			vol: volume{
				Path:             tmpPath,
				AllowedBlockSize: 1,
				BlocksSize:       1,
			},
			want: false,
		},
		{
			name:  "AllowedBlockNumbers == 0",
			dTier: diskTier{DCL: dcl},
			vol: volume{
				Path:                tmpPath,
				AllowedBlockSize:    0,
				AllowedBlockNumbers: 0,
			},
			want: true,
		},
		{
			name:  "AllowedBlockNumbers == BlocksSize",
			dTier: diskTier{DCL: dcl},
			vol: volume{
				Path:                tmpPath,
				AllowedBlockSize:    2,
				BlocksSize:          1,
				AllowedBlockNumbers: 1,
				BlocksCount:         1,
			},
			want: false,
		},
		{
			name:  "AllowedBlockNumbers > BlocksSize",
			dTier: diskTier{DCL: dcl},
			vol: volume{
				Path:                tmpPath,
				AllowedBlockSize:    2,
				BlocksSize:          1,
				AllowedBlockNumbers: 1,
				BlocksCount:         0,
			},
			want: true,
		},
		{
			name:  "InodesToMaintain == 0",
			dTier: diskTier{DCL: dcl},
			vol: volume{
				Path:                tmpPath,
				AllowedBlockSize:    0,
				AllowedBlockNumbers: 0,
			},
			want: true,
		},
		{
			name:  "InodesToMaintain < volStat.Ffree",
			dTier: diskTier{DCL: dcl},
			vol: volume{
				Path:                tmpPath,
				AllowedBlockSize:    0,
				AllowedBlockNumbers: 0,
				InodesToMaintain:    1,
			},
			want: true,
		},
		{
			name:  "InodesToMaintain > volStat.Ffree",
			dTier: diskTier{DCL: dcl},
			vol: volume{
				Path:                tmpPath,
				AllowedBlockSize:    0,
				AllowedBlockNumbers: 0,
				InodesToMaintain:    volStat.Ffree + 1000,
			},
			want: false,
		},
		{
			name:  "SizeToMaintain == 0",
			dTier: diskTier{DCL: dcl},
			vol: volume{
				Path:                tmpPath,
				AllowedBlockSize:    0,
				AllowedBlockNumbers: 0,
				InodesToMaintain:    0,
				SizeToMaintain:      0,
			},
			want: true,
		},
		{
			name:  "SizeToMaintain != 0",
			dTier: diskTier{DCL: dcl},
			vol: volume{
				Path:                tmpPath,
				AllowedBlockSize:    0,
				AllowedBlockNumbers: 0,
				InodesToMaintain:    0,
				SizeToMaintain:      1,
			},
			want: true,
		},
		{
			name:  "SizeToMaintain > availableSize",
			dTier: diskTier{DCL: dcl},
			vol: volume{
				Path:                tmpPath,
				AllowedBlockSize:    0,
				AllowedBlockNumbers: 0,
				InodesToMaintain:    0,
				SizeToMaintain:      volStat.Bfree*uint64(volStat.Bsize) + 2,
			},
			want: false,
		},
		{
			name:  "selectDir error",
			dTier: diskTier{DCL: dcl, DirPrefix: dirPrefix},
			vol: volume{
				Path:                tmpPath,
				AllowedBlockSize:    0,
				AllowedBlockNumbers: 0,
				InodesToMaintain:    0,
				SizeToMaintain:      0,
				CurDirBlockNums:     dcl,
				CurDirInd:           dcl,
				CurKInd:             dcl,
			},
			want: false,
		},
	}

	for idx := range tests {
		test := tests[idx]

		t.Run(test.name, func(t *testing.T) {

			got := test.vol.isAbleToStoreBlock(&test.dTier)
			if got != test.want {
				t.Errorf("isAbleToStoreBlock() got %v | want %v", got, test.want)
			}
		})
	}
}

func Benchmark_volume_read(b *testing.B) {
	var err error
	tmpPath := b.TempDir()
	dirPrefix := "DirPrefix"
	fileName := "fileName"
	dcl := 3

	_, _, _ = mockFileSystem(tmpPath, dirPrefix, fileName, dcl)
	dTier := &diskTier{
		DirPrefix: dirPrefix,
	}
	vol := mockVolume(tmpPath, dcl-1, dcl-1, dcl-1)
	block := mockBlock()
	data, err := json.Marshal(block)
	if err != nil {
		b.Error(err)
	}
	path, err := vol.write(block, data, dTier)
	if err != nil {
		b.Error(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := vol.read("", path); err != nil {
			b.Error(err)
		}
	}

}

func Test_volume_read(t *testing.T) {
	t.Parallel()

	var err error
	tmpPath := t.TempDir()
	dirPrefix := "DirPrefix"
	fileName := "fileName"
	dcl := 3

	_, _, _ = mockFileSystem(tmpPath, dirPrefix, fileName, dcl)
	dTier := &diskTier{
		DirPrefix: dirPrefix,
	}
	vol := mockVolume(tmpPath, dcl-1, dcl-1, dcl-1)
	block := mockBlock()
	data, err := json.Marshal(block)
	if err != nil {
		t.Error(err)
	}
	blockPath, err := vol.write(block, data, dTier)
	if err != nil {
		t.Error(err)
	}
	nilBlockPath := filepath.Join(tmpPath, fmt.Sprintf("%v%v", dirPrefix, 0), "0", fileName+"_1")
	f, err := os.Create(nilBlockPath)
	if err != nil {
		t.Fatalf("read() %v", err)
	}
	_, _ = f.Write([]byte{})

	tests := [4]struct {
		name      string
		vol       volume
		path      string
		wantBlock *b.Block
		error     bool
	}{
		{
			name:      "OK",
			vol:       vol,
			path:      blockPath,
			wantBlock: block,
			error:     false,
		},
		{
			name:      "File Not Exist",
			vol:       vol,
			path:      filepath.Join(tmpPath, fmt.Sprintf("%v%v", dirPrefix, 0), strconv.Itoa(dcl), "test.dat"),
			wantBlock: nil,
			error:     true,
		},
		{
			name:      "Bad Data Block",
			vol:       vol,
			path:      filepath.Join(tmpPath, fmt.Sprintf("%v%v", dirPrefix, 0), "0", fileName+"_0"),
			wantBlock: nil,
			error:     true,
		},
		{
			name:      "Nil Data Block",
			vol:       vol,
			path:      nilBlockPath,
			wantBlock: nil,
			error:     true,
		},
	}

	for idx := range tests {
		test := tests[idx]

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := test.vol.read("", test.path)
			if (err != nil) != test.error {
				t.Errorf("read() error %v | want %v", err, test.error)
			}
			if !reflect.DeepEqual(got, test.wantBlock) {
				t.Errorf("read() got %v |want %v", got, test.wantBlock)
			}
		})
	}
}

func Benchmark_volume_selectDir(b *testing.B) {
	tmpPath := b.TempDir()
	dirPrefix := "DirPrefix"
	fileName := "fileName"
	dcl := 3

	_, _, _ = mockFileSystem(tmpPath, dirPrefix, fileName, dcl-1)

	tests := [3]struct {
		name   string
		volume volume
		dTier  *diskTier
	}{
		{
			name:   "CurDirBlockNums < DCL",
			volume: mockVolume(tmpPath, dcl-1, 0, dcl-1),
			dTier: &diskTier{
				DCL:       dcl,
				DirPrefix: dirPrefix,
			},
		},
		{
			name:   "CurDirInd < DCL-1",
			volume: mockVolume(tmpPath, dcl-1, dcl-2, dcl),
			dTier: &diskTier{
				DCL:       dcl,
				DirPrefix: dirPrefix,
			},
		},
		{
			name:   "CurKInd < DCL-1",
			volume: mockVolume(tmpPath, dcl-2, dcl-1, dcl),
			dTier: &diskTier{
				DCL:       dcl,
				DirPrefix: dirPrefix,
			},
		},
	}

	for idx := range tests {
		test := tests[idx]

		b.Run(test.name, func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				if err := test.volume.selectDir(test.dTier); err != nil {
					b.Error(err)
				}
			}
		})
	}
}

func Test_volume_selectDir(t *testing.T) {
	t.Parallel()

	tmpPath := t.TempDir()
	dirPrefix := "DirPrefix"
	fileName := "fileName"
	dcl := 3

	_, _, _ = mockFileSystem(tmpPath, dirPrefix, fileName, dcl)

	tests := [9]struct {
		name  string
		dTier diskTier
		vol   volume
		want  bool
	}{
		{
			name:  "OK",
			dTier: diskTier{DCL: dcl + 1, DirPrefix: dirPrefix},
			vol: volume{
				Path:            tmpPath,
				CurDirBlockNums: dcl + 1,
				CurDirInd:       dcl + 1,
				CurKInd:         dcl - 2,
			},
			want: false,
		},
		{
			name:  "CurDirBlockNums <  dTier.DCL",
			dTier: diskTier{DCL: dcl, DirPrefix: dirPrefix},
			vol: volume{
				Path:            tmpPath,
				CurDirBlockNums: 0,
				CurDirInd:       dcl - 1,
				CurKInd:         dcl - 1,
			},
			want: false,
		},
		{
			name:  "CurDirBlockNums <  dTier.DCL Without Path",
			dTier: diskTier{DCL: dcl, DirPrefix: dirPrefix},
			vol: volume{
				Path:            tmpPath,
				CurDirBlockNums: 0,
				CurDirInd:       dcl,
				CurKInd:         0,
			},
			want: false,
		},
		{
			name:  "CurDirInd < dTier.DCL-1 ",
			dTier: diskTier{DCL: dcl + 1, DirPrefix: dirPrefix},
			vol: volume{
				Path:            tmpPath,
				CurDirBlockNums: dcl + 1,
				CurDirInd:       dcl - 2,
				CurKInd:         dcl - 1,
			},
			want: false,
		},
		{
			name:  "CurDirInd < dTier.DCL-1 Without Path",
			dTier: diskTier{DCL: dcl + 1, DirPrefix: dirPrefix},
			vol: volume{
				Path:            tmpPath,
				CurDirBlockNums: dcl + 1,
				CurDirInd:       dcl - 2,
				CurKInd:         dcl,
			},
			want: false,
		},
		{
			name:  "CurDirInd < dTier.DCL-1 && blocksCount >= dTier.DCL ",
			dTier: diskTier{DCL: dcl, DirPrefix: dirPrefix},
			vol: volume{
				Path:            tmpPath,
				CurDirBlockNums: dcl,
				CurDirInd:       dcl - 2,
				CurKInd:         0,
			},
			want: true,
		},
		{
			name:  "CurKInd < DCL-1",
			vol:   mockVolume(tmpPath, dcl-2, dcl+1, dcl+1),
			dTier: diskTier{DCL: dcl + 1, DirPrefix: dirPrefix},
			want:  false,
		},
		{
			name:  "CurKInd < DCL-1 Without Path",
			vol:   mockVolume(tmpPath, dcl-1, dcl, dcl+1),
			dTier: diskTier{DCL: dcl + 1, DirPrefix: dirPrefix},
			want:  false,
		},
		{
			name:  "CurKInd < DCL-1 With a Vacant Place",
			vol:   mockVolume(tmpPath, dcl-1, dcl, dcl+1),
			dTier: diskTier{DCL: dcl + 1, DirPrefix: dirPrefix},
			want:  false,
		},
	}

	for idx := range tests {
		test := tests[idx]

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := test.vol.selectDir(&test.dTier)
			if (err != nil) != test.want {
				t.Errorf("selectDir() error %v | want %v", err, test.want)
			}
		})
	}
}

func Benchmark_volume_updateCount(b *testing.B) {
	vol := volume{BlocksCount: uint64(b.N + 1)}

	tests := [3]struct {
		name   string
		volume volume
		value  int64
	}{
		{
			name:   "Increase Counter",
			volume: vol,
			value:  -1,
		},
		{
			name:   "Decrease Counter",
			volume: vol,
			value:  1,
		},
	}

	for ibx := range tests {
		test := tests[ibx]
		b.ResetTimer()

		b.Run(test.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				vol.updateCount(test.value)
			}
		})
	}
}

func Test_volume_updateCount(t *testing.T) {
	t.Parallel()

	tmpPath := t.TempDir()
	var volStat unix.Statfs_t
	if err := unix.Statfs(tmpPath, &volStat); err != nil {
		t.Fatal("test volume updateCount", err)
	}

	tests := [4]struct {
		name   string
		volume volume
		value  int64
		want   uint64
	}{
		{
			name:   "N < 0",
			volume: volume{BlocksCount: 2},
			value:  -1,
			want:   1,
		},
		{
			name:   "N < 0 && BlocksSize == 0",
			volume: volume{BlocksCount: 0},
			value:  -1,
			want:   0,
		},
		{
			name:   "N > 0",
			volume: volume{BlocksCount: 1},
			value:  1,
			want:   2,
		},
		{
			name:   "N > 0 && BlocksSize > math.MaxUint64",
			volume: volume{BlocksCount: math.MaxUint64},
			want:   uint64(math.MaxUint64),
		},
	}

	for idx := range tests {
		test := tests[idx]

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			test.volume.updateCount(test.value)
			if test.volume.BlocksCount != test.want {
				t.Errorf("updateCount() got %v | want %v", test.volume.BlocksCount, test.want)
			}
		})
	}
}

func Benchmark_volume_updateSize(b *testing.B) {
	vol := volume{BlocksSize: uint64(b.N + 1)}

	tests := [3]struct {
		name   string
		volume volume
		value  int64
	}{
		{
			name:   "Increase Counter",
			volume: vol,
			value:  -1,
		},
		{
			name:   "Decrease Counter",
			volume: vol,
			value:  1,
		},
	}

	for ibx := range tests {
		test := tests[ibx]
		b.ResetTimer()

		b.Run(test.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				vol.updateSize(test.value)
			}
		})
	}
}

func Test_volume_updateSize(t *testing.T) {
	t.Parallel()

	tmpPath := t.TempDir()
	var volStat unix.Statfs_t
	if err := unix.Statfs(tmpPath, &volStat); err != nil {
		t.Fatal("test volume updateSize", err)
	}

	tests := [4]struct {
		name   string
		volume volume
		value  int64
		want   uint64
	}{
		{
			name:   "N < 0",
			volume: volume{BlocksSize: uint64(volStat.Bsize), Path: tmpPath},
			value:  -volStat.Bsize,
			want:   0,
		},
		{
			name:   "N < 0 && BlocksSize < volStat.Bsize",
			volume: volume{BlocksSize: uint64(volStat.Bsize) - 1, Path: tmpPath},
			value:  -volStat.Bsize,
			want:   0,
		},
		{
			name:   "N > 0",
			volume: volume{BlocksSize: uint64(volStat.Bsize), Path: tmpPath},
			value:  volStat.Bsize,
			want:   uint64(volStat.Bsize * 2),
		},
		{
			name:   "N > 0 && BlocksSize > volStat.Bsize",
			volume: volume{BlocksSize: math.MaxUint64 - uint64(volStat.Bsize) + 1, Path: tmpPath},
			value:  volStat.Bsize,
			want:   uint64(math.MaxUint64),
		},
	}

	for idx := range tests {
		test := tests[idx]

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			test.volume.updateSize(test.value)
			if test.volume.BlocksSize != test.want {
				t.Errorf("updateSize() got %v | want %v", test.volume.BlocksSize, test.want)
			}
		})
	}
}

func Benchmark_volume_write(b *testing.B) {
	var err error
	tmpPath := b.TempDir()
	dirPrefix := "DirPrefix"
	fileName := "fileName"
	dcl := 3

	_, _, _ = mockFileSystem(tmpPath, dirPrefix, fileName, dcl)
	dTier := &diskTier{
		DirPrefix: dirPrefix,
	}
	vol := mockVolume(tmpPath, dcl-1, dcl-1, dcl-1)
	block := mockBlock()
	data, err := json.Marshal(block)
	if err != nil {
		b.Error(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := vol.write(block, data, dTier); err != nil {
			b.Error(err)
		}
	}
}

func Test_volume_write(t *testing.T) {
	t.Parallel()

	var err error
	tmpPath := t.TempDir()
	dirPrefix := "DirPrefix"
	fileName := "fileName"
	dcl := 3

	_, _, _ = mockFileSystem(tmpPath, dirPrefix, fileName, dcl)

	block := mockBlock()
	data, err := json.Marshal(block)
	if err != nil {
		t.Fatal(err)
	}

	tests := [2]struct {
		name       string
		volume     volume
		dTier      diskTier
		wantPath   string
		wantVolume volume
		wantError  bool
	}{
		{
			name: "OK",
			volume: volume{
				Path:      tmpPath,
				CurKInd:   0,
				CurDirInd: 0,
			},
			dTier: diskTier{DirPrefix: dirPrefix},
			wantPath: filepath.Join(tmpPath, fmt.Sprintf("%v%v/%v", dirPrefix, 0, 0),
				fmt.Sprintf("%v%v", block.Hash, fileExt)),
			wantVolume: volume{Path: tmpPath, BlocksCount: 1, BlocksSize: uint64(len(data)), CurDirBlockNums: 1},
			wantError:  false,
		},
		{
			name: "Wrong Path",
			volume: volume{
				Path:      tmpPath,
				CurKInd:   5,
				CurDirInd: 0,
			},
			dTier: diskTier{DirPrefix: dirPrefix},
			wantPath: filepath.Join(tmpPath, fmt.Sprintf("%v%v/%v", dirPrefix, 5, 0),
				fmt.Sprintf("%v%v", block.Hash, fileExt)),
			wantVolume: volume{Path: tmpPath, CurKInd: 5},
			wantError:  true,
		},
	}

	for idx := range tests {
		test := tests[idx]

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := test.volume.write(block, data, &test.dTier)
			if (err != nil) != test.wantError {
				t.Errorf("write() error %v | want %v", err, test.wantError)
			}
			if got != test.wantPath {
				t.Errorf("write() got %v | want %v", got, test.wantPath)
			}
			if !reflect.DeepEqual(test.volume, test.wantVolume) {
				t.Errorf("write() got %v | want %v", test.wantVolume, test.volume)
			}
		})
	}
}

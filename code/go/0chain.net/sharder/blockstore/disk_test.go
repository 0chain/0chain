package blockstore

import (
	"bufio"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"0chain.net/chaincore/block"
	"0chain.net/core/datastore"
	"github.com/0chain/common/core/logging"

	"github.com/stretchr/testify/require"
)

func init() {
	logging.InitLogging("debug", ".")
}

func TestIsVolumeAbleToStoreBlock(t *testing.T) {
	p := "./vol"
	err := os.MkdirAll(p, 0777)
	require.Nil(t, err)

	type input struct {
		name        string
		vol         volume
		setup       func(t *testing.T)
		furtherTest func(t *testing.T, in *input)
		isAble      bool
	}

	tests := []input{
		{
			name: "exceeded block numbers",
			vol: volume{
				AllowedBlockNumbers: 10,
				BlocksCount:         10,
			},
			isAble: false,
		},
		{
			name: "exceeded block size",
			vol: volume{
				AllowedBlockSize: 10,
				BlocksSize:       10,
			},
			isAble: false,
		},
		{
			name: "able to store",
			vol: volume{
				Path:            "./vol",
				CountMu:         &sync.Mutex{},
				IndMu:           &sync.Mutex{},
				CurKInd:         10,
				CurDirInd:       100,
				CurDirBlockNums: DirectoryContentLimit,
			},
			isAble: true,

			setup: func(t *testing.T) {
				err := os.MkdirAll("./vol", 0777)
				require.Nil(t, err)
			},

			furtherTest: func(t *testing.T, in *input) {
				require.Equal(t, 10, in.vol.CurKInd)
				require.Equal(t, 101, in.vol.CurDirInd)
				require.Equal(t, 0, in.vol.CurDirBlockNums)
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			if test.setup != nil {
				test.setup(t)
			}

			isAble := test.vol.isAbleToStoreBlock()
			require.Equal(t, test.isAble, isAble)

			if test.furtherTest != nil {
				test.furtherTest(t, &test)
			}

		})
	}
}

func TestVolumeWrite(t *testing.T) {
	type input struct {
		name        string
		block       *block.Block
		vol         volume
		data        []byte
		setup       func(t *testing.T)
		furtherTest func(t *testing.T, in *input)
		tearUp      func(t *testing.T)
	}

	tests := []input{
		{
			name:  "Volume write okay#1",
			block: &block.Block{HashIDField: datastore.HashIDField{Hash: "hash1"}},
			vol: volume{
				Path:    "./vol",
				CountMu: &sync.Mutex{},
				IndMu:   &sync.Mutex{},
			},
			setup: func(t *testing.T) {
				err := os.MkdirAll("./vol/blocks/K0/0", 0777)
				require.Nil(t, err)
			},
			furtherTest: func(t *testing.T, in *input) {
				require.Equal(t, uint64(1), in.vol.BlocksCount)
				require.Equal(t, uint64(len(in.data)), in.vol.BlocksSize)
				require.Equal(t, 1, in.vol.CurDirBlockNums)
			},
			data: generateRandomBytes(t, 1024*10),
			tearUp: func(t *testing.T) {
				err := os.RemoveAll("./vol")
				require.Nil(t, err)
			},
		},
		{
			name:  "Volume write okay#2",
			block: &block.Block{HashIDField: datastore.HashIDField{Hash: "hash2"}},
			vol: volume{
				Path:    "./vol",
				CountMu: &sync.Mutex{},
				IndMu:   &sync.Mutex{},
			},
			setup: func(t *testing.T) {
				err := os.MkdirAll("./vol/blocks/K0/0", 0777)
				require.Nil(t, err)
			},
			furtherTest: func(t *testing.T, in *input) {
				require.Equal(t, uint64(1), in.vol.BlocksCount)
				require.Equal(t, uint64(len(in.data)), in.vol.BlocksSize)
				require.Equal(t, 1, in.vol.CurDirBlockNums)
			},
			data: generateRandomBytes(t, 1024*100),
			tearUp: func(t *testing.T) {
				err := os.RemoveAll("./vol")
				require.Nil(t, err)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer test.tearUp(t)

			if test.setup != nil {
				test.setup(t)
			}

			err := test.vol.selectDir()
			require.Nil(t, err)
			_, err = test.vol.write(test.block, test.data)
			require.Nil(t, err)

			if test.furtherTest != nil {
				test.furtherTest(t, &test)
			}
		})
	}
}

func TestVolumeRead(t *testing.T) {
	type input struct {
		name    string
		dTier   diskTier
		bPath   string
		setup   func(t *testing.T)
		tearup  func(t *testing.T)
		wantErr bool
	}

	tests := []input{
		{
			name:  "Read Okay",
			dTier: diskTier{},
			bPath: "./vol/hash1.dat",
			setup: func(t *testing.T) {
				err := os.MkdirAll("./vol", 0777)
				require.Nil(t, err)

				b := block.Block{HashIDField: datastore.HashIDField{Hash: "hash1"}}
				data, err := json.Marshal(b)
				require.Nil(t, err)

				f, err := os.Create("./vol/hash1.dat")
				require.Nil(t, err)
				defer f.Close()

				bf := bufio.NewWriterSize(f, 64*1024)
				volWriter, err := zlib.NewWriterLevel(f, zlib.BestCompression)
				require.Nil(t, err)

				defer volWriter.Close()
				_, err = volWriter.Write(data)
				require.Nil(t, err)

				err = bf.Flush()
				require.Nil(t, err)

			},
			tearup: func(t *testing.T) {
				err := os.RemoveAll("./vol")
				require.Nil(t, err)
			},
			wantErr: false,
		},
		{
			name:  "Read Fail",
			dTier: diskTier{},
			bPath: "./vol/hash1.dat",
			setup: func(t *testing.T) {
				err := os.MkdirAll("./vol", 0777)
				require.Nil(t, err)

				data := generateRandomBytes(t, 1024)
				require.Nil(t, err)

				f, err := os.Create("./vol/hash1.dat")
				require.Nil(t, err)

				_, err = f.Write(data)
				require.Nil(t, err)
			},
			tearup: func(t *testing.T) {
				err := os.RemoveAll("./vol")
				require.Nil(t, err)
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.tearup != nil {
				defer test.tearup(t)
			}

			if test.setup != nil {
				test.setup(t)
			}

			_, err := test.dTier.read(test.bPath)
			if test.wantErr {
				require.NotNil(t, err)
				return
			}
			require.Nil(t, err)
		})
	}
}

func TestSelectDir(t *testing.T) {
	type input struct {
		name        string
		vol         volume
		setup       func(t *testing.T)
		tearup      func(t *testing.T)
		furtherTest func(t *testing.T, in *input)
		wantErr     bool
		errContains string
	}

	tests := []input{
		{
			name: "Select directory okay#1",
			vol: volume{
				Path:            "./vol",
				CurKInd:         1,
				CurDirInd:       DirectoryContentLimit - 1,
				CurDirBlockNums: DirectoryContentLimit,
				IndMu:           &sync.Mutex{},
			},
			setup: func(t *testing.T) {
				err := os.MkdirAll("./vol", 0777)
				require.Nil(t, err)
			},
			tearup: func(t *testing.T) {
				err := os.RemoveAll("./vol")
				require.Nil(t, err)
			},
			furtherTest: func(t *testing.T, in *input) {
				require.Equal(t, 2, in.vol.CurKInd)
				require.Equal(t, 0, in.vol.CurDirInd)

				blocksPath := filepath.Join(in.vol.Path, fmt.Sprintf("%v%v/%v", DirPrefix, in.vol.CurKInd, in.vol.CurDirInd))
				_, err := os.Stat(blocksPath)
				require.Nil(t, err)

				curKInd, curDirInd, err := getCurIndexes(filepath.Join(in.vol.Path, IndexStateFileName))
				require.Nil(t, err)
				require.Equal(t, 2, curKInd)
				require.Equal(t, 0, curDirInd)
			},
		},
		{
			name: "Volume full",
			vol: volume{
				Path:            "./vol",
				CurKInd:         DirectoryContentLimit - 1,
				CurDirInd:       DirectoryContentLimit - 1,
				CurDirBlockNums: DirectoryContentLimit,
				IndMu:           &sync.Mutex{},
			},
			setup: func(t *testing.T) {
				blocksPath := filepath.Join("./vol", fmt.Sprintf("%v%v/%v", DirPrefix, 0, 0))
				err := os.MkdirAll(blocksPath, 0777)
				require.Nil(t, err)

				wg := &sync.WaitGroup{}
				for i := 0; i < DirectoryContentLimit; i++ {
					wg.Add(1)
					go func(i int) {
						defer wg.Done()
						hash := "hash" + fmt.Sprint(i)
						f, err := os.Create(filepath.Join(blocksPath, hash))
						require.Nil(t, err)
						f.Close()
					}(i)
				}
				wg.Wait()
			},
			tearup: func(t *testing.T) {
				err := os.RemoveAll("./vol")
				require.Nil(t, err)
			},
			wantErr:     true,
			errContains: "is full",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.tearup != nil {
				defer test.tearup(t)
			}

			if test.setup != nil {
				test.setup(t)
			}

			err := test.vol.selectDir()
			if test.wantErr {
				require.NotNil(t, err)
				require.Contains(t, err.Error(), test.errContains)
				return
			}

			require.Nil(t, err)
			if test.furtherTest != nil {
				test.furtherTest(t, &test)
			}
		})
	}
}

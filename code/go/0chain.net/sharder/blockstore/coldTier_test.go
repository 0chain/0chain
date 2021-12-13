package blockstore

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/client"
	"0chain.net/chaincore/config"
	"0chain.net/core/datastore"
	"0chain.net/core/logging"
	"0chain.net/core/memorystore"
	"0chain.net/sharder/blockstore/options"
)

var (
	rootPath = os.TempDir() + "sharder"
)

func TestMain(m *testing.M) {
	movePath := moveBlockPath()

	if err := os.Mkdir(rootPath, 0777); err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(movePath, 0777); err != nil {
		log.Fatal(err)
	}
	if _, err := os.Create(movePath + "/test.dat"); err != nil {
		log.Fatal(err)
	}

	sp := memorystore.GetStorageProvider()
	block.SetupEntity(sp)
	block.SetupBlockSummaryEntity(sp)

	clientEM := datastore.MetadataProvider()
	clientEM.Name = "client"
	clientEM.Provider = client.Provider
	clientEM.Store = sp
	datastore.RegisterEntityMetadata("client", clientEM)

	logging.InitLogging("testing")
	config.SetServerChainID("")

	code := m.Run()

	// clean up
	if err := os.RemoveAll(rootPath); err != nil {
		log.Println("cannot clean up path: " + rootPath + " - please remove it manually")
	}

	os.Exit(code)
}

/*********************************  coldDisk benchmarks *********************************/

func Benchmark_coldDisk_getBlock(b *testing.B) {
	hash := rootPath + "/test.zlib"
	rawbytes := []byte("coldDisk_getBlock test")
	var buf bytes.Buffer
	writer := zlib.NewWriter(&buf)
	_, err := writer.Write(rawbytes)
	err = writer.Close()

	err = ioutil.WriteFile(rootPath+"/test.zlib", buf.Bytes(), 0755)
	if err != nil {
		log.Printf("Unable to write file: %v", err)
	}

	var mockDisk *coldDisk

	for i := 0; i < b.N; i++ {
		_, err := mockDisk.GetBlock(hash)
		require.NoError(b, err)
	}
}

func Benchmark_coldDisk_getBlocks(b *testing.B) {
	hash := rootPath + "/test.zlib"
	rawbytes := []byte("coldDisk_getBlock test")
	var buf bytes.Buffer
	writer := zlib.NewWriter(&buf)
	_, err := writer.Write(rawbytes)
	err = writer.Close()

	err = ioutil.WriteFile(rootPath+"/test.zlib", buf.Bytes(), 0755)
	if err != nil {
		log.Printf("Unable to write file: %v", err)
	}

	var mockDisk *coldDisk
	for i := 0; i < b.N; i++ {
		_, err := mockDisk.GetBlocks(&options.ColdFilterOptions{Prefix: hash})
		require.NoError(b, err)
	}
}

func Benchmark_coldDisk_selectDir(b *testing.B) {
	// cannot be run in parallel

	curDirBlockNums := CDCL
	curDirInd := CDCL
	blocksCount := 0

	mockDisk := &coldDisk{
		Path:            rootPath,
		CurKInd:         0,
		CurDirInd:       curDirInd,
		CurDirBlockNums: curDirBlockNums,
	}

	updateCurIndexes = mockUpdateCurIndexes()
	countFiles = mockCountFiles(blocksCount)

	for i := 0; i < b.N; i++ {
		_ = mockDisk.selectDir()
	}
}

func Benchmark_coldDisk_isAbleToStoreBlock(b *testing.B) {
	mockDisk := &coldDisk{
		Path: rootPath,
	}

	for i := 0; i < b.N; i++ {
		_ = mockDisk.isAbleToStoreBlock()
	}
}

func Benchmark_coldDisk_moveBlock(b *testing.B) {
	mockDisk := newTestColdDisk()

	hash := "test"
	oldBlockPath := fmt.Sprintf("%v/test.dat", moveBlockPath())

	for i := 0; i < b.N; i++ {
		_, _ = mockDisk.MoveBlock(hash, oldBlockPath)
	}
}

/*********************************  coldTier benchmarks *********************************/

func Benchmark_coldTier_moveBlock(b *testing.B) {
	mockTier := &coldTier{
		SelectNextStorage: func(coldStorageProviders []ColdStorageProvider, prevInd int) {},
		PrevInd:           0,
		DeleteLocal:       false,
	}

	sc := selectedColdStorage{
		coldStorage: mockColdStorageProvider(),
	}

	for i := 0; i < b.N; i++ {
		c := make(chan selectedColdStorage)

		go func() {
			c <- sc
		}()

		mockTier.SelectedStorageChan = c
		if _, err := mockTier.moveBlock("", ""); err != nil {
			b.Fatal(err)
		}
	}
}

func Benchmark_coldTier_read(b *testing.B) {
	mockTier := &coldTier{}

	mockTier.StorageType = "disk"
	coldPath := rootPath + "/test.zlib"

	err := ioutil.WriteFile(coldPath, []byte("Hello"), 0755)
	if err != nil {
		log.Printf("Unable to write file: %v", err)
	}

	for i := 0; i < b.N; i++ {
		r, err := mockTier.read(coldPath, "")
		if err != nil {
			log.Fatal(err)
		}
		err = r.Close()
		require.NoError(b, err)
	}
}

func Benchmark_coldTier_removeSelectedColdStorage(b *testing.B) {
	for i := 0; i < b.N; i++ {
		mockTier := &coldTier{
			ColdStorages: []ColdStorageProvider{
				&coldDisk{},
				&coldDisk{},
			},
			PrevInd: 0,
		}

		mockTier.removeSelectedColdStorage()
	}
}

func Benchmark_coldTier_write(b *testing.B) {
	mockTier := &coldTier{
		SelectNextStorage: func(coldStorageProviders []ColdStorageProvider, prevInd int) {},
	}
	sc := selectedColdStorage{
		coldStorage: mockColdStorageProvider(),
	}

	temp := datastore.GetEntityMetadata("block")
	bl, ok := temp.Instance().(*block.Block)
	if !ok {
		log.Fatal(ok)
	}
	bl.Round = 1
	bl.ChainID = ""

	for i := 0; i < b.N; i++ {
		c := make(chan selectedColdStorage)
		go func() {
			c <- sc
		}()

		mockTier.SelectedStorageChan = c

		_, err := mockTier.write(bl, []byte("coldTier_write test"))
		require.NoError(b, err)
	}
}

/*********************************  MinioClient benchmarks *********************************/

func Benchmark_minioClient_initialize(b *testing.B) {
	for i := 0; i < b.N; i++ {

	}
}

func Benchmark_minioClient_GetBlock(b *testing.B) {
	// TODO: code refactor required for testing
	for i := 0; i < b.N; i++ {
		// mockClient := MinioClient{
		//	Client: mockMinioClient(),
		// }
		//
		// _, err := mockClient.GetBlock("test")
		// require.NoError(b, err)
	}
}

func Benchmark_minioClient_MoveBlock(b *testing.B) {
	mockClient := MinioClient{
		Client: mockMinioClient(),
	}

	for i := 0; i < b.N; i++ {
		if _, err := mockClient.MoveBlock("", ""); err != nil {
			b.Fatal(err)
		}
	}
}

func Benchmark_minioClient_WriteBlock(b *testing.B) {
	mockClient := MinioClient{
		Client: mockMinioClient(),
	}

	for i := 0; i < b.N; i++ {
		_, err := mockClient.WriteBlock(&block.Block{}, []byte("test"))
		require.NoError(b, err)
	}
}

func Benchmark_startcoldVolumes(b *testing.B) {
	mVolumes := []map[string]interface{}{
		{
			"path":                  os.TempDir() + "test_dir",
			"size_to_maintain":      uint64(10),
			"inodes_to_maintain":    uint64(1000),
			"allowed_block_numbers": uint64(10),
			"allowed_block_size":    uint64(10),
		},
	}
	coldTier := &coldTier{
		ColdStorages: []ColdStorageProvider{
			&coldDisk{},
		},
	}
	shouldDelete := true

	// mock functions
	countBlocksInVolumes = mockCountBlocksInVolumes()
	getAvailableSizeAndInodes = mockGetAvailableSizeAndInodes()
	getCurIndexes = mockGetCurIndexes()
	getCurrentDirBlockNums = mockGetCurrentDirBlockNums()
	getUint64ValueFromYamlConfig = mockGetUint64ValueFromYamlConfig()

	for i := 0; i < b.N; i++ {
		startcoldVolumes(mVolumes, coldTier, shouldDelete)
	}
}

func Benchmark_startColdVolumes(b *testing.B) {
	mVolumes := []map[string]interface{}{
		{
			"path":                  os.TempDir() + "test_dir",
			"size_to_maintain":      uint64(10),
			"inodes_to_maintain":    uint64(1000),
			"allowed_block_numbers": uint64(10),
			"allowed_block_size":    uint64(10),
		},
	}
	coldTier := &coldTier{
		ColdStorages: []ColdStorageProvider{
			&coldDisk{},
		},
	}

	// mock functions
	countBlocksInVolumes = mockCountBlocksInVolumes()
	getAvailableSizeAndInodes = mockGetAvailableSizeAndInodes()
	getCurIndexes = mockGetCurIndexes()
	getCurrentDirBlockNums = mockGetCurrentDirBlockNums()
	getUint64ValueFromYamlConfig = mockGetUint64ValueFromYamlConfig()

	for i := 0; i < b.N; i++ {
		startColdVolumes(mVolumes, coldTier)
	}
}

func Benchmark_restartColdVolumes(b *testing.B) {
	mVolumes := []map[string]interface{}{
		{
			"path":                  os.TempDir() + "test_dir",
			"size_to_maintain":      uint64(10),
			"inodes_to_maintain":    uint64(1000),
			"allowed_block_numbers": uint64(10),
			"allowed_block_size":    uint64(10),
		},
	}
	coldTier := &coldTier{
		ColdStorages: []ColdStorageProvider{
			&coldDisk{},
		},
	}

	// mock functions
	countBlocksInVolumes = mockCountBlocksInVolumes()
	getAvailableSizeAndInodes = mockGetAvailableSizeAndInodes()
	getCurIndexes = mockGetCurIndexes()
	getCurrentDirBlockNums = mockGetCurrentDirBlockNums()
	getUint64ValueFromYamlConfig = mockGetUint64ValueFromYamlConfig()

	for i := 0; i < b.N; i++ {
		restartColdVolumes(mVolumes, coldTier)
	}
}

/******************************** coldTier unit tests *********************************/

func Test_coldTier_write(t *testing.T) {
	t.Parallel()

	mockTier := &coldTier{
		SelectNextStorage: func(coldStorageProviders []ColdStorageProvider, prevInd int) {},
	}
	temp := datastore.GetEntityMetadata("block")
	bl, ok := temp.Instance().(*block.Block)
	if !ok {
		log.Fatal(ok)
	}
	bl.Round = 1
	bl.ChainID = ""

	type test struct {
		name       string
		block      *block.Block
		data       []byte
		preprocess func()
		want       string
		wantErr    bool
	}

	tests := [2]test{
		{
			name:  "Test_coldTier_writer_OK",
			block: &block.Block{},
			data:  []byte("coldTier_write test"),
			preprocess: func() {
				sc := selectedColdStorage{
					coldStorage: mockColdStorageProvider(),
				}

				c := make(chan selectedColdStorage)
				go func() {
					c <- sc
				}()

				mockTier.SelectedStorageChan = c
			},
			want:    "",
			wantErr: false,
		},
		{
			name:  "Test_coldTier_writer_ERR",
			block: &block.Block{},
			data:  []byte("coldTier_write test"),
			preprocess: func() {
				sc := selectedColdStorage{
					coldStorage: mockColdStorageProvider(),
					err:         errors.New("test error"),
				}

				c := make(chan selectedColdStorage)
				go func() {
					c <- sc
				}()

				mockTier.SelectedStorageChan = c
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.preprocess()

			got, err := mockTier.write(tt.block, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("coldTier.write() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("coldTier.write() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_coldTier_read(t *testing.T) {
	coldPath := rootPath + "/test.zlib"
	data := []byte("Hello")

	err := ioutil.WriteFile(coldPath, data, 0755)
	if err != nil {
		log.Printf("Unable to write file: %v", err)
	}

	if coldStoragesMap == nil {
		coldStoragesMap = map[string]ColdStorageProvider{}
	}

	coldStoragesMap[coldPath] = mockColdStorageProvider()

	type test struct {
		name        string
		storageType string
		hash        string
		want        io.ReadCloser
		wantErr     bool
	}

	tests := [5]test{
		{
			name:        "Test_coldTier_read_disk_OK",
			storageType: "disk",
			hash:        "test",
			want:        &os.File{},
			wantErr:     false,
		},
		{
			name:        "Test_coldTier_read_minio_OK",
			storageType: "minio",
			hash:        "test",
			want:        ioutil.NopCloser(bytes.NewReader(data)),
			wantErr:     false,
		},
		{
			name:    "Test_coldTier_read_noType_OK",
			hash:    "test",
			want:    nil,
			wantErr: false,
		},
		{
			name:        "Test_coldTier_read_minio_ERR",
			storageType: "minio",
			hash:        "err",
			want:        nil,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTier := &coldTier{
				StorageType: tt.storageType,
			}

			got, err := mockTier.read(coldPath, tt.hash)
			if (err != nil) != tt.wantErr {
				t.Errorf("coldTier.read() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !(reflect.TypeOf(got) == reflect.TypeOf(tt.want)) {
				t.Errorf("coldTier.read() got = %v, want %v", got, tt.want)
			}

		})
	}
}

func Test_coldTier_moveBlock(t *testing.T) {
	mockTier := &coldTier{
		SelectNextStorage: func(coldStorageProviders []ColdStorageProvider, prevInd int) {},
		PrevInd:           0,
		DeleteLocal:       false,
	}
	temp := datastore.GetEntityMetadata("block")
	bl, ok := temp.Instance().(*block.Block)
	if !ok {
		log.Fatal(ok)
	}
	bl.Round = 1
	bl.ChainID = ""

	sc := selectedColdStorage{
		coldStorage: mockColdStorageProvider(),
	}

	type test struct {
		name    string
		want    string
		wantErr bool
	}

	tests := [2]test{
		{
			name:    "Test_coldTier_moveBlock_OK",
			want:    "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := make(chan selectedColdStorage)
			go func() {
				c <- sc
			}()
			mockTier.SelectedStorageChan = c
			got, err := mockTier.moveBlock("", "")
			if (err != nil) != tt.wantErr {
				t.Errorf("coldTier.moveBlock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("coldTier.moveBlock() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_coldTier_removeSelectedColdStorage(t *testing.T) {
	type test struct {
		name       string
		coldTier   *coldTier
		oldPrevInd int
	}

	tests := [1]test{
		{
			name: "Test_coldTier_removeSelectedColdStorage_OK",
			coldTier: &coldTier{
				ColdStorages: []ColdStorageProvider{
					&coldDisk{},
					&coldDisk{},
				},
				PrevInd: 0,
			},
			oldPrevInd: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.coldTier.removeSelectedColdStorage()
			if tt.coldTier.PrevInd != tt.oldPrevInd-1 {
				t.Errorf("coldTier.removeSelectedColdStorage() got = %v, want = %v", tt.coldTier.PrevInd, tt.oldPrevInd-1)
			}
		})
	}

}

/*********************************  coldDisk unit tests *********************************/

func Test_coldDisk_getBlock(t *testing.T) {
	t.Parallel()

	type test struct {
		name       string
		preprocess func()
		data       []byte
		hash       string
		want       []byte
		wantErr    bool
	}

	tests := []test{
		{
			name: "Test_coldDisk_GetBlock_OK",
			preprocess: func() {
				text := []byte("test")

				var buf bytes.Buffer
				writer := zlib.NewWriter(&buf)
				_, err := writer.Write(text)
				err = writer.Close()

				err = ioutil.WriteFile(rootPath+"/test.zlib", buf.Bytes(), 0755)
				if err != nil {
					log.Printf("Unable to write file: %v", err)
				}
			},
			hash:    rootPath + "/test.zlib",
			want:    []byte("test"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.preprocess()

			var mockDisk *coldDisk
			got, err := mockDisk.GetBlock(tt.hash)
			if (err != nil) != tt.wantErr {
				t.Errorf("coldDisk.GetBlock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("coldDisk.GetBlock() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_coldDisk_isAbleToStoreBlock(t *testing.T) {
	mockDisk := &coldDisk{
		Path: rootPath,
	}

	type test struct {
		name string
		want bool
	}

	tests := []test{
		{
			name: "Test_coldDisk_isAbleToStoreBlock_OK",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDisk.isAbleToStoreBlock()
		})
	}

}

func Test_coldDisk_selectDir(t *testing.T) {
	// cannot be run in parallel

	type test struct {
		name            string
		curDirBlockNums int
		curDirInd       int
		blocksCount     int
		wantErr         bool
	}

	tests := []test{
		{
			name:            "Test_coldDisk_selectDir_OK",
			curDirBlockNums: CDCL,
			curDirInd:       CDCL,
			blocksCount:     0,
			wantErr:         false,
		},
		{
			name:            "Test_coldDisk_selectDir_curDirBlockNums_OK",
			curDirBlockNums: CDCL - 2,
			curDirInd:       CDCL,
			blocksCount:     0,
			wantErr:         false,
		},
		{
			name:            "Test_coldDisk_selectDir_curDirInd_OK",
			curDirBlockNums: CDCL,
			curDirInd:       CDCL - 2,
			blocksCount:     0,
			wantErr:         false,
		},
		{
			name:            "Test_coldDisk_selectDir_ERR",
			curDirBlockNums: CDCL,
			curDirInd:       CDCL,
			blocksCount:     CDCL,
			wantErr:         true,
		},
		{
			name:            "Test_coldDisk_selectDir_curDirInd_ERR",
			curDirBlockNums: CDCL,
			curDirInd:       CDCL - 2,
			blocksCount:     CDCL,
			wantErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDisk := &coldDisk{
				Path:            rootPath,
				CurKInd:         0,
				CurDirInd:       tt.curDirInd,
				CurDirBlockNums: tt.curDirBlockNums,
			}

			updateCurIndexes = mockUpdateCurIndexes()
			countFiles = mockCountFiles(tt.blocksCount)

			err := mockDisk.selectDir()
			if (err != nil) != tt.wantErr {
				t.Errorf("coldDisk.selectDir() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
		})
	}
}

func Test_coldDisk_MoveBlock(t *testing.T) {
	mockDisk := newTestColdDisk()

	type test struct {
		name         string
		hash         string
		oldBlockPath string
		want         string
		wantErr      bool
	}

	tests := []test{
		{
			name:         "Test_coldDisk_MoveBlock_OK",
			hash:         "test",
			oldBlockPath: fmt.Sprintf("%v/test.dat", moveBlockPath()),
			want:         fmt.Sprintf("%v/test.dat", moveBlockPath()),
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mockDisk.MoveBlock(tt.hash, tt.oldBlockPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("coldDisk.MoveBlock() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && (got != tt.want) {
				t.Errorf("coldDisk.MoveBlock() got = %v, want = %v", got, tt.want)
			}
		})
	}
}

func Test_coldDisk_updateBlocksCount(t *testing.T) {
	t.Skip("overflow when subtracting uint64(i)")
	blocksCount := int64(10)

	type test struct {
		name string
		i    int64
		want uint64
	}

	tests := [2]test{
		{
			name: "Test_coldTier_updateBlocksCount_add_OK",
			i:    1,
			want: uint64(blocksCount) + 1,
		},
		{
			name: "Test_coldTier_updateBlocksCount_sub_OK",
			i:    -1,
			want: uint64(blocksCount) - 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDisk := &coldDisk{
				BlocksCount: uint64(blocksCount),
			}

			mockDisk.updateBlocksCount(tt.i)
			if mockDisk.BlocksCount != tt.want {
				t.Errorf("coldTier.updateBlocksCount() got = %v, want = %v", mockDisk.BlocksCount, tt.want)
			}
		})
	}

}

func Test_coldDisk_updateBlocksSize(t *testing.T) {
	t.Skip("overflow when subtracting uint64(i)")
	blocksSize := int64(10)

	type test struct {
		name string
		i    int64
		want uint64
	}

	tests := [2]test{
		{
			name: "Test_coldTier_updateBlocksSize_add_OK",
			i:    1,
			want: uint64(blocksSize) + 1,
		},
		{
			name: "Test_coldTier_updateBlocksSize_sub_OK",
			i:    -1,
			want: uint64(blocksSize) - 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDisk := &coldDisk{
				BlocksSize: uint64(blocksSize),
			}

			mockDisk.updateBlocksSize(tt.i)
			if mockDisk.BlocksSize != tt.want {
				t.Errorf("coldTier.updateBlocksSize() got = %v, want = %v", mockDisk.BlocksSize, tt.want)
			}
		})
	}

}

func Test_minioClient_GetBlock(t *testing.T) {
	t.Skip("obj.Read hanging cannot mock")
	mockMinio := mockMinioClient()

	type test struct {
		name    string
		hash    string
		want    []byte
		wantErr bool
	}

	tests := []test{
		{
			name:    "Test_minioClient_GetBlock_OK",
			hash:    "",
			want:    []byte("test"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := MinioClient{
				Client: mockMinio,
			}

			got, err := mc.GetBlock(tt.hash)
			if (err != nil) != tt.wantErr {
				t.Errorf("minioClient.GetBlock() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && reflect.DeepEqual(got, tt.want) {
				t.Errorf("minioClient.GetBlock() got = %v, want = %v", got, tt.want)
			}
		})
	}
}

func Test_minioClient_initialize(t *testing.T) {
	type test struct {
		name string
	}

	tests := []test{
		{
			name: "Test_minioClient_initialize_OK",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

		})
	}
}

func Test_minioClient_MoveBlock(t *testing.T) {
	mockMinio := mockMinioClient()

	type test struct {
		name              string
		data              []byte
		storageServiceURL string
		bucketName        string
		wantErr           bool
	}

	tests := []test{
		{
			name:              "Test_minioClient_MoveBlock_OK",
			data:              []byte("test"),
			storageServiceURL: "testStorageServiceURL",
			bucketName:        "testBucketName",
			wantErr:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := MinioClient{
				Client:            mockMinio,
				storageServiceURL: tt.storageServiceURL,
				bucketName:        tt.bucketName,
			}

			want := fmt.Sprintf("%v:%v", mc.storageServiceURL, mc.bucketName)

			got, err := mc.MoveBlock("", "")
			if (err != nil) != tt.wantErr {
				t.Errorf("minioClient.MoveBlock() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && (got != want) {
				t.Errorf("minioClient.MoveBlock() got = %v, want = %v", got, want)
			}

		})
	}
}

func Test_minioClient_WriteBlock(t *testing.T) {
	mockMinio := mockMinioClient()

	type test struct {
		name              string
		data              []byte
		storageServiceURL string
		bucketName        string
		wantErr           bool
	}

	tests := []test{
		{
			name:              "Test_minioClient_WriteBlock_OK",
			data:              []byte("test"),
			storageServiceURL: "testStorageServiceURL",
			bucketName:        "testBucketName",
			wantErr:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := MinioClient{
				Client:            mockMinio,
				storageServiceURL: tt.storageServiceURL,
				bucketName:        tt.bucketName,
			}

			want := fmt.Sprintf("%v:%v", mc.storageServiceURL, mc.bucketName)

			got, err := mc.WriteBlock(&block.Block{}, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("minioClient.WriteBlock() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && (got != want) {
				t.Errorf("minioClient.WriteBlock() got = %v, want = %v", got, want)
			}

		})
	}
}

func Test_startcloudstorages(t *testing.T) {
	type test struct {
		name          string
		cloudStorages []map[string]interface{}
		coldTier      *coldTier
		shouldDelete  bool
	}

	tests := [2]test{
		{
			name: "Test_startcloudstorages_OK",
			cloudStorages: []map[string]interface{}{
				{
					"storage_service_url":   "test_storage_service_url",
					"access_id":             "test_access_id",
					"secret_access_key":     "test_secret_access_key",
					"bucket_name":           "test_bucket_name",
					"allowed_block_numbers": uint64(10),
					"allowed_block_size":    uint64(10),
					"use_ssl":               false,
				},
			},
			coldTier: &coldTier{
				ColdStorages: []ColdStorageProvider{
					&coldDisk{},
				},
			},
			shouldDelete: false,
		},
		{
			name: "Test_restartCloudStorages_OK",
			cloudStorages: []map[string]interface{}{
				{
					"storage_service_url":   "test_storage_service_url",
					"access_id":             "test_access_id",
					"secret_access_key":     "test_secret_access_key",
					"bucket_name":           "test_bucket_name",
					"allowed_block_numbers": uint64(10),
					"allowed_block_size":    uint64(10),
					"use_ssl":               false,
				},
			},
			coldTier: &coldTier{
				ColdStorages: []ColdStorageProvider{
					&coldDisk{},
					&coldDisk{},
				},
			},
			shouldDelete: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("startcloudstorages() panic occured, recovered in %v", r)
				}
			}()

			// mock functions
			getUint64ValueFromYamlConfig = mockGetUint64ValueFromYamlConfig()

			startcloudstorages(tt.cloudStorages, tt.coldTier, tt.shouldDelete)
		})
	}
}

func Test_startcoldVolumes(t *testing.T) {
	type test struct {
		name         string
		mVolumes     []map[string]interface{}
		coldTier     *coldTier
		shouldDelete bool
	}

	tests := [2]test{
		{
			name: "Test_restartColdVolumes_OK",
			mVolumes: []map[string]interface{}{
				{
					"path":                  os.TempDir() + "test_dir",
					"size_to_maintain":      uint64(10),
					"inodes_to_maintain":    uint64(1000),
					"allowed_block_numbers": uint64(10),
					"allowed_block_size":    uint64(10),
				},
			},
			coldTier: &coldTier{
				ColdStorages: []ColdStorageProvider{
					&coldDisk{},
				},
			},
			shouldDelete: false,
		},
		{
			name: "Test_startColdVolumes_OK",
			mVolumes: []map[string]interface{}{
				{
					"path":                  os.TempDir() + "test_dir",
					"size_to_maintain":      uint64(10),
					"inodes_to_maintain":    uint64(1000),
					"allowed_block_numbers": uint64(10),
					"allowed_block_size":    uint64(10),
				},
			},
			coldTier: &coldTier{
				ColdStorages: []ColdStorageProvider{
					&coldDisk{},
				},
			},
			shouldDelete: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("startcoldVolumes() panic occured, recovered in %v", r)
				}
			}()

			// mock functions
			countBlocksInVolumes = mockCountBlocksInVolumes()
			getAvailableSizeAndInodes = mockGetAvailableSizeAndInodes()
			getCurIndexes = mockGetCurIndexes()
			getCurrentDirBlockNums = mockGetCurrentDirBlockNums()
			getUint64ValueFromYamlConfig = mockGetUint64ValueFromYamlConfig()

			startcoldVolumes(tt.mVolumes, tt.coldTier, tt.shouldDelete)
		})
	}
}

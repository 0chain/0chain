package blockstore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sync"
	"unsafe"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/mock"

	"0chain.net/sharder/mocks"
)

type coldStorageProvider struct {
	mocks.ColdStorageProvider
}

type minioClient struct {
	mocks.MinioClientI
}

func mockColdStorageProvider() ColdStorageProvider {
	c := coldStorageProvider{}

	c.On("GetBlock",
		mock.AnythingOfType("string"),
	).Return(func(hash string) []byte {
		if hash == "err" {
			return nil
		}

		return []byte("")
	}, func(hash string) error {
		if hash == "err" {
			return errors.New("error")
		}

		return nil
	})

	c.On("MoveBlock",
		mock.AnythingOfType("string"),
		mock.AnythingOfType("string"),
	).Return("", nil)

	c.On("WriteBlock",
		mock.AnythingOfType("*block.Block"),
		mock.AnythingOfType("[]uint8"),
	).Return("", nil)

	return &c
}

func mockCountBlocksInVolumes() CountBlocksInVolumes {
	u := mocks.CountBlocksInVolumes{}

	u.On("Execute",
		mock.AnythingOfType("string"),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("int"),
	).Return(uint64(10), uint64(10))

	return u.Execute
}

func mockCountFiles(count int) CountFiles {
	c := mocks.CountFiles{}

	c.On("Execute",
		mock.AnythingOfType("string"),
	).Return(
		func(dirPath string) int { return count },
		func(dirPath string) error { return nil },
	)

	return c.Execute
}

func mockGetAvailableSizeAndInodes() GetAvailableSizeAndInodes {
	u := mocks.GetAvailableSizeAndInodes{}

	u.On("Execute",
		mock.AnythingOfType("string"),
	).Return(uint64(10), uint64(10), uint64(10), nil)

	return u.Execute
}

func mockGetCurIndexes() GetCurIndexes {
	u := mocks.GetCurIndexes{}

	u.On("Execute",
		mock.AnythingOfType("string"),
	).Return(10, 10, nil)

	return u.Execute
}

func mockGetCurrentDirBlockNums() GetCurrentDirBlockNums {
	u := mocks.GetCurrentDirBlockNums{}

	u.On("Execute",
		mock.AnythingOfType("string"),
	).Return(10, nil)

	return u.Execute
}

func mockGetUint64ValueFromYamlConfig() GetUint64ValueFromYamlConfig {
	u := mocks.GetUint64ValueFromYamlConfig{}

	u.On("Execute",
		mock.Anything,
	).Return(func(v interface{}) uint64 {
		return v.(uint64)
	}, nil)

	return u.Execute
}

func mockUpdateCurIndexes() UpdateCurIndexes {
	u := mocks.UpdateCurIndexes{}

	u.On("Execute",
		mock.AnythingOfType("string"),
		mock.AnythingOfType("int"),
		mock.AnythingOfType("int"),
	).Return(nil)

	return u.Execute
}

func newTestColdDisk() *coldDisk {
	updateCurIndexes = mockUpdateCurIndexes()
	countFiles = mockCountFiles(0)

	return &coldDisk{
		Path:            rootPath,
		CurKInd:         0,
		CurDirInd:       CDCL,
		CurDirBlockNums: CDCL,
	}

}

func mockMinioClient() MinioClientI {
	mc := minioClient{}
	AnyReader := mock.MatchedBy(func(r io.Reader) bool {
		return true
	})
	AnyContext := mock.MatchedBy(func(c context.Context) bool {
		return true
	})

	mc.On("FPutObject",
		AnyContext,
		mock.AnythingOfType("string"),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("minio.PutObjectOptions"),
	).Return(minio.UploadInfo{}, nil)

	mc.On("GetObject",
		AnyContext,
		mock.AnythingOfType("string"),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("minio.GetObjectOptions"),
	).Return(func(ctx context.Context, bucketName, objectName string, opts minio.GetObjectOptions) *minio.Object {
		mo := &minio.Object{}
		// t := reflect.ValueOf(mo)
		// f := t.FieldByName("mutex")
		// f.Set(reflect.ValueOf(&sync.Mutex{}))
		// v := reflect.ValueOf(mo)
		// wv := reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem()

		ptrTof := unsafe.Pointer(mo)
		// ptrTof = unsafe.Pointer(uintptr(ptrTof)) // move according to types

		ptrToy := (**sync.Mutex)(ptrTof)

		*ptrToy = &sync.Mutex{} // or *ptrToy = &Foo{} or whatever you want

		return mo
	}, nil)

	mc.On("ListObjects",
		AnyContext,
		mock.AnythingOfType("string"),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("minio.ListObjectsOptions"),
	).Return(make(chan minio.ObjectInfo))

	mc.On("PutObject",
		AnyContext,
		mock.AnythingOfType("string"),
		mock.AnythingOfType("string"),
		AnyReader,
		mock.AnythingOfType("int64"),
		mock.AnythingOfType("minio.PutObjectOptions"),
	).Return(minio.UploadInfo{}, nil)

	mc.On("RemoveBucket",
		AnyContext,
		mock.AnythingOfType("string"),
	).Return(nil)

	mc.On("StatObject",
		AnyContext,
		mock.AnythingOfType("string"),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("minio.GetObjectOptions"),
	).Return(minio.ObjectInfo{
		Size: 1,
	}, nil)

	return &mc
}

func moveBlockPath() string {
	mockDisk := newTestColdDisk()

	return filepath.Join(rootPath, fmt.Sprintf("%v%v/%v", CK, mockDisk.CurKInd+1, 0))
}

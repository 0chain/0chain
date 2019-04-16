package roundstore

import (
	"encoding/binary"
	"os"
	"path/filepath"
)

type FSRoundStore struct {
	RootDirectory string
	fileName      string
	bytes         int
}

func NewFSRoundStore(rootDir string) *FSRoundStore {
	store := &FSRoundStore{RootDirectory: rootDir}
	store.fileName = "round"
	store.bytes = 0
	return store
}

func (frs *FSRoundStore) getFile() string {
	return frs.RootDirectory + string(os.PathSeparator) + frs.fileName + ".bin"
}

func (frs *FSRoundStore) Write(roundNum int64) error {
	file := frs.getFile()
	dir := filepath.Dir(file)
	os.MkdirAll(dir, 0755)
	data := make([]byte, 8)
	binary.LittleEndian.PutUint64(data, uint64(roundNum))
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	frs.bytes, err = f.Write(data)
	if err != nil {
		return err
	}
	return nil
}

func (frs *FSRoundStore) Read() (int64, error) {
	file := frs.getFile()
	f, err := os.Open(file)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	data := make([]byte, 8)
	_, err = f.Read(data)
	if err != nil {
		return 0, err
	}
	value := int64(binary.LittleEndian.Uint64(data))
	return value, nil
}

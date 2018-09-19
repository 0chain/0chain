package blockdb

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/golang/snappy"
)

/*BlockDB - a simple database that supports write and read of an immutable block on the blockchain where the
records of the database are the transactions in the block*/
type BlockDB struct {
	file      string
	compress  bool
	dbHeader  DBHeader
	index     Index
	keyLength int8
	dataFile  *os.File
}

/*NewBlockDB - create a new block db
-- file name is of the form directory/where/to/store/dbfile. The actual files will be dbfile.idx and dbfile.dat
-- create - create a new one or only try to open an existing one
-- compress - compress the records being saved
*/
func NewBlockDB(file string, keyLength int8, compress bool) (*BlockDB, error) {
	db := &BlockDB{file: file, keyLength: keyLength, compress: compress}
	return db, nil
}

//SetDBHeader - set the db header
func (bdb *BlockDB) SetDBHeader(dbHeader DBHeader) {
	bdb.dbHeader = dbHeader
}

//SetIndex - set the index object
func (bdb *BlockDB) SetIndex(index Index) {
	bdb.index = index
}

//Create - create the database
func (bdb *BlockDB) Create() error {
	dir := filepath.Dir(bdb.file)
	err := os.MkdirAll(dir, 0755)
	if bdb.index == nil {
		bdb.SetIndex(newMapIndex())
	}
	bdb.dataFile, err = os.OpenFile(bdb.getDataFileName(), os.O_RDWR|os.O_CREATE, 0644)
	return err
}

//Open - open an existing database
func (bdb *BlockDB) Open() error {
	f, err := os.OpenFile(bdb.getHeaderFileName(), os.O_RDONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	if bdb.index == nil {
		bdb.SetIndex(newFixedKeyArrayIndex(bdb.keyLength))
	}
	err = bdb.readHeader(f)
	if err != nil {
		return err
	}
	bdb.dataFile, err = os.OpenFile(bdb.getDataFileName(), os.O_RDONLY, 0644)
	return err
}

//Read - read an individual record
func (bdb *BlockDB) Read(key Key, record Record) error {
	offset, err := bdb.index.GetOffset(key)
	if err != nil {
		return err
	}
	_, err = bdb.dataFile.Seek(offset, 0)
	if err != nil {
		return err
	}
	var dlen int32
	err = binary.Read(bdb.dataFile, binary.LittleEndian, &dlen)
	if err != nil {
		return err
	}
	data := make([]byte, dlen, dlen)
	n, err := bdb.dataFile.Read(data)
	if err != nil {
		return err
	}
	if int32(n) != dlen {
		return errors.New("read data length doesnot match expected data length")
	}
	if bdb.compress {
		data, err = snappy.Decode(nil, data)
		if err != nil {
			return err
		}
	}
	buffer := bytes.NewBuffer(data)
	err = record.Decode(buffer)
	return err
}

//ReadAll - read all the records
func (bdb *BlockDB) ReadAll(rp RecordProvider) ([]Record, error) {
	keys := bdb.index.GetKeys()
	records := make([]Record, 0, len(keys))
	for _, key := range keys {
		record := rp.NewRecord()
		err := bdb.Read(key, record)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

//WriteData - write the data
func (bdb *BlockDB) WriteData(record Record) error {
	offset, err := bdb.dataFile.Seek(0, 1)
	if err != nil {
		return err
	}
	bdb.index.SetOffset(record.GetKey(), offset)
	buffer := bytes.NewBuffer(nil)
	err = record.Encode(buffer)
	if err != nil {
		return err
	}
	if bdb.compress {
		cbytes := snappy.Encode(nil, buffer.Bytes())
		buffer = bytes.NewBuffer(cbytes)
	}
	data := buffer.Bytes()
	dlen := int32(len(data))
	err = binary.Write(bdb.dataFile, binary.LittleEndian, dlen)
	if err != nil {
		return err
	}
	n, err := bdb.dataFile.Write(data)
	if err != nil {
		return err
	}
	if int32(n) != dlen {
		return errors.New("written data length doesn't match computed length")
	}
	return err
}

//Iterate - implement the interface
func (bdb *BlockDB) Iterate(ctx context.Context, handler DBIteratorHandler, rp RecordProvider) error {
	records, err := bdb.ReadAll(rp)
	if err != nil {
		return err
	}
	for _, record := range records {
		err = handler(ctx, record)
		if err != nil {
			return err
		}
	}
	return nil
}

//Save - implement interface
func (bdb *BlockDB) Save() error {
	bdb.saveHeader()
	return bdb.Close()
}

//Close - implement interface
func (bdb *BlockDB) Close() error {
	if bdb.dataFile != nil {
		return bdb.dataFile.Close()
	}
	return nil
}

//Delete - implement interface
func (bdb *BlockDB) Delete() error {
	err := os.Remove(bdb.getHeaderFileName())
	if err != nil {
		return err
	}
	return os.Remove(bdb.getDataFileName())
}

func (bdb *BlockDB) saveHeader() error {
	headerFile, err := os.OpenFile(bdb.getHeaderFileName(), os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer headerFile.Close()
	err = bdb.writeIndex(headerFile)
	if err != nil {
		return err
	}
	if bdb.dbHeader != nil {
		buffer := bytes.NewBuffer(nil)
		err = bdb.dbHeader.Encode(buffer)
		if err != nil {
			return err
		}
		if bdb.compress {
			cbytes := snappy.Encode(nil, buffer.Bytes())
			buffer = bytes.NewBuffer(cbytes)
		}
		_, err = buffer.WriteTo(headerFile)
	}
	return err
}

func (bdb *BlockDB) readHeader(file *os.File) error {
	err := bdb.readIndex(file)
	if err != nil {
		return err
	}
	if bdb.dbHeader != nil {
		var data []byte
		data, err = ioutil.ReadAll(file)
		if err != nil {
			return err
		}
		if bdb.compress {
			data, err = snappy.Decode(nil, data)
			if err != nil {
				return err
			}
		}
		buffer := bytes.NewBuffer(data)
		err = bdb.dbHeader.Decode(buffer)
	}
	return err
}

func (bdb *BlockDB) writeIndex(file *os.File) error {
	return bdb.index.Encode(file)
}

func (bdb *BlockDB) readIndex(file *os.File) error {
	return bdb.index.Decode(file)
}

const (
	FileExtHeader = "idx"
	FileExtData   = "dat"
)

func (bdb *BlockDB) getHeaderFileName() string {
	return bdb.file + "." + FileExtHeader
}

func (bdb *BlockDB) getDataFileName() string {
	return bdb.file + "." + FileExtData
}

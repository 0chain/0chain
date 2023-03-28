package blockdb

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"sort"
	"sync"
)

type keyo struct {
	k Key
	o int64
}

//mapIndex - an implementation of the db index
type mapIndex struct {
	index map[Key]int64
	mutex sync.RWMutex
}

func newMapIndex() *mapIndex {
	idx := &mapIndex{}
	idx.index = make(map[Key]int64)
	return idx
}

//SetOffset - set the offset of the given record
func (mi *mapIndex) SetOffset(key Key, offset int64) error {
	mi.mutex.Lock()
	defer mi.mutex.Unlock()
	mi.index[key] = offset
	return nil
}

//GetOffset - get the offset of the given record */
func (mi *mapIndex) GetOffset(key Key) (int64, error) {
	mi.mutex.RLock()
	defer mi.mutex.RUnlock()
	offset, ok := mi.index[key]
	if !ok {
		return -1, ErrKeyNotFound
	}
	return offset, nil
}

func (mi *mapIndex) Encode(writer io.Writer) error {
	mi.mutex.RLock()
	defer mi.mutex.RUnlock()
	numKeys := int32(len(mi.index))
	err := binary.Write(writer, binary.LittleEndian, numKeys)
	if err != nil {
		return err
	}

	keyos := make([]keyo, 0, len(mi.index))
	for k, o := range mi.index {
		keyos = append(keyos, keyo{k: k, o: o})
	}
	sort.SliceStable(keyos, func(i, j int) bool { return keyos[i].k < keyos[j].k })
	buffer := bytes.NewBuffer(nil)
	for _, ko := range keyos {
		var key = ko.k
		var offset = ko.o
		var klen = int8(len(key))
		err = binary.Write(buffer, binary.LittleEndian, klen)
		if err != nil {
			return err
		}
		n, err := buffer.Write([]byte(key))
		if err != nil {
			return err
		}
		if n != len(key) {
			return errors.New("written bytes length doesn't match the key length")
		}
		err = binary.Write(buffer, binary.LittleEndian, offset)
		if err != nil {
			return err
		}
	}
	_, err = buffer.WriteTo(writer)
	return err
}

func (mi *mapIndex) Decode(reader io.Reader) error {
	mi.mutex.Lock()
	defer mi.mutex.Unlock()
	var numKeys int32
	err := binary.Read(reader, binary.LittleEndian, &numKeys)
	if err != nil {
		return err
	}
	mi.index = make(map[Key]int64, numKeys)
	for i := int32(0); i < numKeys; i++ {
		var klen int8
		err := binary.Read(reader, binary.LittleEndian, &klen)
		if err != nil {
			return err
		}
		buf := make([]byte, klen)
		n, err := reader.Read(buf)
		if err != nil {
			return err
		}
		if int8(n) != klen {
			return errors.New("coudld not read the required number of bytes")
		}
		var key = Key(buf)
		var offset int64
		err = binary.Read(reader, binary.LittleEndian, &offset)
		if err != nil {
			return err
		}
		mi.index[key] = offset
	}
	return nil
}

func (mi *mapIndex) GetKeys() []Key {
	mi.mutex.RLock()
	defer mi.mutex.RUnlock()
	keyos := make([]keyo, 0, len(mi.index))
	for k, o := range mi.index {
		keyos = append(keyos, keyo{k: k, o: o})
	}
	sort.SliceStable(keyos, func(i, j int) bool { return keyos[i].o < keyos[j].o })
	keys := make([]Key, len(keyos))
	for idx, key := range keyos {
		keys[idx] = key.k
	}
	return keys
}

//fixedKeyArrayIndex
type fixedKeyArrayIndex struct {
	buffer []byte
	keylen int8
}

func newFixedKeyArrayIndex(keyLength int8) *fixedKeyArrayIndex {
	fkai := &fixedKeyArrayIndex{}
	fkai.keylen = keyLength
	return fkai
}

func (fkai *fixedKeyArrayIndex) getKeySize() int8 {
	return int8(fkai.keylen) + 1 + 8
}

//SetOffset - set the offset of the given record
func (fkai *fixedKeyArrayIndex) SetOffset(_ Key, _ int64) error {
	return errors.New("method not supported for this implementation")
}

//GetOffset - get the offset of the given record */
func (fkai *fixedKeyArrayIndex) GetOffset(key Key) (int64, error) {
	klen := int(fkai.keylen)
	ksz := fkai.getKeySize()
	numKeys := len(fkai.buffer) / int(ksz)
	bkey := []byte(key)
	var offset int64
	for lo, hi := 0, numKeys-1; lo <= hi; {
		mid := (lo + hi) / 2
		start := int(ksz) * mid
		switch bytes.Compare(fkai.buffer[start+1:start+1+klen], bkey) {
		case 0:
			err := binary.Read(bytes.NewBuffer(fkai.buffer[start+1+klen:]), binary.LittleEndian, &offset)
			if err != nil {
				return -1, err
			}
			return offset, nil
		case -1:
			if lo == hi {
				break
			}
			lo = mid + 1
		case 1:
			if lo == hi {
				break
			}
			hi = mid - 1
		}
	}
	return -1, ErrKeyNotFound
}

func (fkai *fixedKeyArrayIndex) GetKeys() []Key {
	klen := int(fkai.keylen)
	ksz := fkai.getKeySize()
	numKeys := len(fkai.buffer) / int(ksz)
	keys := make([]Key, 0, numKeys)
	for i := 0; i < numKeys; i++ {
		start := i * int(ksz)
		key := Key(fkai.buffer[start+1 : start+1+klen])
		keys = append(keys, key)
	}
	return keys
}

func (fkai *fixedKeyArrayIndex) Encode(_ io.Writer) error {
	//TODO
	return nil
}

func (fkai *fixedKeyArrayIndex) Decode(reader io.Reader) error {
	var numKeys int32
	err := binary.Read(reader, binary.LittleEndian, &numKeys)
	if err != nil {
		return err
	}
	sz := int(numKeys * int32(fkai.getKeySize()))
	fkai.buffer = make([]byte, sz)
	n, err := reader.Read(fkai.buffer)
	if err != nil {
		return err
	}
	if n != sz {
		return errors.New("couldn't read the entire index")
	}
	/*
		for i := int32(0); i < numKeys; i++ {
			start := int(i * int32(fkai.getKeySize()))
			var offset int64
			klen := int(fkai.keylen)
			err := binary.Read(bytes.NewBuffer(fkai.buffer[start+1+klen:]), binary.LittleEndian, &offset)
			if err != nil {
				return err
			}
			fmt.Printf("DEBUG: %v %v\n", string(fkai.buffer[start+1:start+1+klen]), offset)
		}*/
	return nil
}

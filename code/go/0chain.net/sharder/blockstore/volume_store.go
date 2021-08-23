//Tiering is done to achieve large storage capacity, disk failures and performance as cache disk(SSD) will be used for latest and
// frequently used blocks.
//Hot tiering: Block data is in the cache disk
//Warm tiering: Block data in in HDD
//Cold tiering: Block data is in minio/s3/blobber server

package blockstore

import (
	"bufio"
	"compress/zlib"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"0chain.net/chaincore/block"
)

type Volume struct {
	rootPath                string
	blocksSize, blocksCount uint64
	availableSize           uint64
}

func (v *Volume) Write(b *block.Block, data []byte, subDir string) (bPath string, err error) {
	bPath = path.Join(v.rootPath, subDir, fmt.Sprintf("%v.%v", b.Hash, fileExt))
	var f *os.File
	f, err = os.Create(bPath)
	if err != nil {
		return
	}
	bf := bufio.NewWriterSize(f, 64*1024)
	volumeWriter, err := zlib.NewWriterLevel(f, zlib.BestCompression)

	if err != nil {
		return
	}
	var n int
	n, err = volumeWriter.Write(data)
	if err != nil {
		volumeWriter.Close()
		os.Remove(bPath)
		return
	}

	if err = volumeWriter.Close(); err != nil {
		f.Close()
		os.Remove(bPath)
		return
	}
	if err = bf.Flush(); err != nil {
		f.Close()
		os.Remove(bPath)
		return
	}
	if err = f.Close(); err != nil {
		os.Remove(bPath)
		return
	}
	v.updateCount(1)
	v.updateSize(int64(n))
	return
}

func (v *Volume) updateSize(n int64) {
	if n < 0 {
		v.blocksSize -= uint64(n)
		v.availableSize += uint64(n)
	} else {
		v.blocksSize += uint64(n)
		v.availableSize -= uint64(n)
	}
}

func (v *Volume) updateCount(n int64) {
	if n < 0 {
		v.blocksCount -= uint64(n)
	} else {
		v.blocksCount += uint64(n)
	}
}

func volumeStrategy(strategy string) interface{} {
	switch strategy {
	case Random:
		r := func(volumes []Volume) Volume { //return volume path
			return volumes[0]
		}
		return r
	case RoundRobin:
		r := func(volumes []Volume) Volume { //return volume path
			return volumes[0]
		}
		return r
	case MinCountFirst:
		//
		return nil
	case MinSizeFirst:
		//
		return nil
	default:
		//
		panic(fmt.Errorf("Stragegy %v not defined", strategy))
	}
}

func checkVolumes(volumesPath []string) (volumes []Volume) {
	for _, v := range volumesPath {
		_, err := os.Stat(v)
		if err != nil {
			panic(fmt.Errorf("Volume %v is not accessible: Got error \"%v\"", v, err)) //check to mount unmounted volumes
		}
		blocksPath := path.Join(v, "blocks")
		var size int64
		var count uint64
		err = filepath.Walk(blocksPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				count++
			}
			size += info.Size()
			return nil

		})
		volume := Volume{rootPath: blocksPath, blocksSize: uint64(size), blocksCount: count}
		volumes = append(volumes, volume)
	}
	return
}

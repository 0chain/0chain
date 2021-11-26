package mocks

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
)

func CommonMock(path, dirPrefix string, dcl int) (countFiles, size uint64, err error) {
	for i := 0; i < dcl; i++ {
		subPath := filepath.Join(path, dirPrefix+fmt.Sprint(i))
		_ = os.Mkdir(subPath, 0777)
		for j := 0; j < dcl; j++ {
			sPath := filepath.Join(subPath, fmt.Sprint(j))
			_ = os.Mkdir(sPath, 0777)
			for x := 0; x < dcl; x++ {
				fileName := "fileForCount" + "_" + strconv.Itoa(x)
				fileForCount := filepath.Join(sPath, fileName)
				fTemp, err := os.Create(fileForCount)
				if err != nil {
					log.Fatal(err)
				}
				countFiles++
				for j := 0; j < 100; j++ {
					_, _ = fTemp.WriteString("Hello, Bench\n")
				}

				info, _ := fTemp.Stat()
				size += uint64(info.Size())
				_ = fTemp.Close()
			}
		}
	}

	return countFiles, size, err
}

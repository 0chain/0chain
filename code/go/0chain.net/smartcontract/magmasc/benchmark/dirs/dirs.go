package dirs

import (
	"os"
)

const (
	SciDir    = "/tmp/0chain/bench/execute-stress/sci"
	SciLogDir = "/tmp/0chain/bench/execute-stress/sci-log"
	DbDir     = "/tmp/0chain/bench/execute-stress/db"
)

// CleanDirs cleans SciDir, SciLogDir, DbDir.
func CleanDirs() error {
	dirs := []string{SciDir, SciLogDir, DbDir}
	for _, dir := range dirs {
		err := os.RemoveAll(dir)
		if err != nil {
			return err
		}
	}
	return nil
}

// CreateDirs creates SciDir, SciLogDir, DbDir.
func CreateDirs() error {
	dirs := []string{SciDir, SciLogDir, DbDir}
	for _, dir := range dirs {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
	}
	return nil
}

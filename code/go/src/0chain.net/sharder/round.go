package sharder

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"strconv"

	"0chain.net/ememorystore"
	"0chain.net/round"
)

func getFileName() string {
	return "data/round/round.txt"
}

/*WriteRound - writes round number to file system*/
func WriteRound(roundNum int64) error {
	fileName := getFileName()
	dir := filepath.Dir(fileName)
	os.MkdirAll(dir, 0755)
	f, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	bf := bufio.NewWriter(f)
	bf.WriteString(strconv.FormatInt(roundNum, 10))
	bf.Flush()
	return nil
}

/*ReadRound - read round from file system*/
func ReadRound() (int64, error) {
	fileName := getFileName()
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		err := WriteRound(0)
		return 0, err
	}
	f, err := os.Open(fileName)
	if err != nil {
		return -1, err
	}
	defer f.Close()
	reader := bufio.NewReader(f)
	roundNum, err := reader.ReadString('\n')
	return strconv.ParseInt(roundNum, 10, 64)
}

/*StoreRound - persists given round to ememory(rocksdb)*/
func (sc *Chain) StoreRound(ctx context.Context, r *round.Round) error {
	roundEntityMetadata := r.GetEntityMetadata()
	rctx := ememorystore.WithEntityConnection(ctx, roundEntityMetadata)
	defer ememorystore.Close(rctx)
	err := r.Write(rctx)
	if err != nil {
		return err
	}
	con := ememorystore.GetEntityCon(rctx, roundEntityMetadata)
	err = con.Commit()
	if err != nil {
		return err
	}
	return nil
}

package cmd

import (
	"context"
	"fmt"
	"os"
	"path"
	"testing"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
	"0chain.net/smartcontract/benchmark"
	bk "0chain.net/smartcontract/benchmark"
	"0chain.net/smartcontract/benchmark/main/cmd/log"
	"github.com/0chain/common/core/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

var (
	mpt    *util.MerklePatriciaTrie
	root   util.Key
	data   *benchmark.BenchData
	suites []bk.TestSuite
)

func prepare(b *testing.B) func() {
	loadPath := viper.GetString("load")
	log.Println("load path", loadPath)
	configPath := viper.GetString("config")
	if loadPath != "" {
		configPath = path.Join(loadPath, "benchmark.yaml")
	}
	log.Println("config path", configPath)

	GetViper(loadPath)
	log.PrintSimSettings()
	common.SetupRootContext(context.Background())

	tests, omittedTests := suitesOmits()
	log.Println("read in command line options")

	executor := common.NewWithContextFunc(viper.GetInt(bk.OptionsLoadConcurrency))
	var mptDir string
	mpt, root, data, mptDir = getMpt(loadPath, configPath, executor)
	log.Println("finished setting up blockchain", "root", string(root))

	savePath := viper.GetString(bk.OptionSavePath)
	if len(savePath) > 0 && loadPath != savePath {
		if err := viper.WriteConfigAs(path.Join(savePath, "benchmark.yaml")); err != nil {
			log.Fatal("cannot copy config file to", savePath)
		}
	}
	//testsTimer := time.Now()
	suites = getTestSuites(data, tests, omittedTests)
	return func() { // clean up function
		err := os.RemoveAll(mptDir)
		fmt.Println("rm: ", mptDir, "err:", err)
	}
}

func BenchmarkTest(b *testing.B) {
	//totalTimer := time.Now()
	// path to config file can only come from command line options
	clean := prepare(b)
	defer clean()

	b.Run(fmt.Sprintf("test-%s", suites[0].Benchmarks[0].Name()), func(b *testing.B) {
		bm := suites[0].Benchmarks[0]
		for i := 0; i < b.N; i++ {
			cloneMPT := util.CloneMPT(mpt)
			_, balances := getBalances(
				bm.Transaction(),
				extractMpt(cloneMPT, root),
				data,
			)
			timedBalance := cstate.NewTimedQueryStateContext(balances, func() common.Timestamp {
				return data.Now
			})

			b.StartTimer()
			err := bm.Run(timedBalance, b)
			b.StopTimer()

			require.NoError(b, err)
		}
	})

}

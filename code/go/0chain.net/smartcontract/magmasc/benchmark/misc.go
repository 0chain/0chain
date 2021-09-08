package benchmark

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	zmc "github.com/0chain/gosdk/zmagmacore/magmasc"

	"0chain.net/chaincore/block"
	chain "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
	"0chain.net/smartcontract/benchmark"
	"0chain.net/smartcontract/magmasc"
	"0chain.net/smartcontract/magmasc/benchmark/dirs"
	"0chain.net/smartcontract/magmasc/benchmark/rand"
	"0chain.net/smartcontract/magmasc/benchmark/sessions"
	"0chain.net/smartcontract/minersc"
)

func mkTempDir() string {
	d := filepath.Join(os.TempDir(), rand.String(6))
	err := os.MkdirAll(d, 0755)
	panicIfErr(err)
	return d
}

func createTempDirsForStress() (dbDir, sciDbDir, sciLogDir string) {
	dbDir, sciDbDir, sciLogDir = mkTempDir(), mkTempDir(), mkTempDir()
	copyDir(dirs.SciDir, sciDbDir)
	copyDir(dirs.DbDir, dbDir)
	copyDir(dirs.SciLogDir, sciLogDir)

	return dbDir, sciDbDir, sciLogDir
}

func copyDir(source, destination string) {
	var err = filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		var relPath = strings.Replace(path, source, "", 1)
		if relPath == "" {
			return nil
		}
		if info.IsDir() {
			return os.Mkdir(filepath.Join(destination, relPath), 0755)
		} else {
			var (
				data, err = ioutil.ReadFile(filepath.Join(source, relPath))
			)
			if err != nil {
				return err
			}
			return ioutil.WriteFile(filepath.Join(destination, relPath), data, 0777)
		}
	})
	panicIfErr(err)
}

func getSourcePostfix(sc *magmasc.MagmaSmartContract, sci chain.StateContextI) string {
	actSess, inactSess, err := sessions.Count(sc, sci)
	panicIfErr(err)

	consumers, providers, err := countNodes(sc, sci)
	panicIfErr(err)

	return del + strconv.Itoa(actSess) + "as" + del + strconv.Itoa(inactSess) + del + "is" +
		del + strconv.Itoa(consumers) + "c" + del + strconv.Itoa(providers) + "p"
}

func countNodes(sc *magmasc.MagmaSmartContract, sci chain.StateContextI) (consumers, providers int, err error) {
	allConsHandl := sc.RestHandlers["/allConsumers"]
	output, err := allConsHandl(nil, nil, sci)
	if err != nil {
		return 0, 0, err
	}

	cList := output.([]*zmc.Consumer)
	consumers = len(cList)

	allProvidersHandl := sc.RestHandlers["/allProviders"]
	output, err = allProvidersHandl(nil, nil, sci)
	if err != nil {
		return 0, 0, err
	}

	pList := output.([]*zmc.Provider)
	providers = len(pList)

	return consumers, providers, nil
}

func getBalances(txn *transaction.Transaction, mpt *util.MerklePatriciaTrie, data benchmark.BenchData) (*util.MerklePatriciaTrie, chain.StateContextI) {
	bk := &block.Block{
		MagicBlock: &block.MagicBlock{
			StartingRound: 0,
		},
		PrevBlock: &block.Block{},
	}
	bk.Round = 2
	bk.MinerID = minersc.GetMockNodeId(0, minersc.NodeTypeMiner)
	node.Self.Underlying().SetKey(minersc.GetMockNodeId(0, minersc.NodeTypeMiner))
	magicBlock := &block.MagicBlock{}
	signatureScheme := &encryption.BLS0ChainScheme{}
	return mpt, chain.NewStateContext(
		bk,
		mpt,
		&state.Deserializer{},
		txn,
		func(*block.Block) []string { return data.Sharders },
		func() *block.Block { return bk },
		func() *block.MagicBlock { return magicBlock },
		func() encryption.SignatureScheme { return signatureScheme },
	)
}

func extractMpt(mpt *util.MerklePatriciaTrie, root util.Key) *util.MerklePatriciaTrie {
	pNode := mpt.GetNodeDB()
	memNode := util.NewMemoryNodeDB()
	levelNode := util.NewLevelNodeDB(
		memNode,
		pNode,
		false,
	)
	return util.NewMerklePatriciaTrie(levelNode, 1, root)
}

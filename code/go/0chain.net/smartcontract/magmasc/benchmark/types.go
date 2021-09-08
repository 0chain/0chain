package benchmark

import (
	"os"
	"sync"

	"0chain.net/chaincore/block"
	chain "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	store "0chain.net/core/ememorystore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
	"0chain.net/smartcontract/magmasc"
)

type (
	SC struct {
		magma *magmasc.MagmaSmartContract
		dbDir string

		mu sync.Mutex
	}
)

const (
	// storeName describes the magma smart contract's store name.
	storeName = "magmadb"
)

func makeSC(dbDir string) *SC {
	db, err := store.CreateDB(dbDir)
	panicIfErr(err)

	store.AddPool(storeName, db)

	sc := magmasc.NewMagmaSmartContract()
	sc.SetDB(db)
	return &SC{
		magma: sc,
		dbDir: dbDir,
	}
}

func (sc *SC) Clean() {
	sc.magma.GetDB().Close()

	err := os.RemoveAll(sc.dbDir)
	panicIfErr(err)
}

type (
	SCI struct {
		sci chain.StateContextI

		mpt *util.MerklePatriciaTrie

		dir    string
		logDir string

		mu sync.Mutex
	}
)

func makeSCI(dbDir, logDir string, mptRoot []byte) *SCI {
	pNodeDB, err := util.NewPNodeDB(dbDir, logDir)
	panicIfErr(err)

	mpt := util.NewMerklePatriciaTrie(pNodeDB, 1, mptRoot)

	return &SCI{
		sci: chain.NewStateContext(
			&block.Block{},
			mpt,
			&state.Deserializer{},
			&transaction.Transaction{},
			func(*block.Block) []string { return []string{} },
			func() *block.Block { return &block.Block{} },
			func() *block.MagicBlock { return &block.MagicBlock{} },
			func() encryption.SignatureScheme { return &encryption.BLS0ChainScheme{} },
		),
		mpt:    mpt,
		dir:    dbDir,
		logDir: logDir,
	}
}

func (sci *SCI) Clean() {
	pNodeDB, ok := sci.sci.GetState().GetNodeDB().(*util.PNodeDB)
	if !ok {
		panic("must be pNodeDB type")
	}
	pNodeDB.Close()

	err := os.RemoveAll(sci.dir)
	panicIfErr(err)

	err = os.RemoveAll(sci.logDir)
	panicIfErr(err)
}

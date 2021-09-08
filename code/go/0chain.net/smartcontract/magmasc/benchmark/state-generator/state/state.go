package state

import (
	"errors"

	"github.com/0chain/gorocksdb"

	"0chain.net/chaincore/block"
	chain "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	store "0chain.net/core/ememorystore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
	"0chain.net/smartcontract/magmasc/benchmark/mpt"
)

func CreateStateContextAndDB(sciDbDir, logDir, dbDir string, txn *transaction.Transaction) (
	chain.StateContextI, *gorocksdb.TransactionDB, error) {

	db, err := createDB(dbDir)
	if err != nil {
		return nil, nil, err
	}

	root, err := mpt.GetRoot(db)
	if err != nil {
		return nil, nil, err
	}

	sci, err := createSCI(sciDbDir, logDir, txn, root)
	if err != nil {
		return nil, nil, err
	}

	return sci, db, nil
}

// createSCI creates state.StateContextI with only util.NewMerklePatriciaTrie initialized,
// and provided transaction.
//
// For util.NewMerklePatriciaTrie util.PNodeDB is used.
func createSCI(dbDir, logDir string, txn *transaction.Transaction, mptRoot []byte) (chain.StateContextI, error) {
	pNodeDB, err := util.NewPNodeDB(dbDir, logDir)
	if err != nil {
		return nil, err
	}

	return chain.NewStateContext(
		&block.Block{},
		util.NewMerklePatriciaTrie(pNodeDB, 1, mptRoot),
		&state.Deserializer{},
		txn,
		func(*block.Block) []string { return []string{} },
		func() *block.Block { return &block.Block{} },
		func() *block.MagicBlock { return &block.MagicBlock{} },
		func() encryption.SignatureScheme { return &encryption.BLS0ChainScheme{} },
	), err
}

const (
	// storeName describes the magma smart contract's store name.
	storeName = "magmadb"
)

// createDB opens gorocksdb.TransactionDB  on provided path.
func createDB(path string) (*gorocksdb.TransactionDB, error) {
	db, err := store.CreateDB(path)
	if err != nil {
		return nil, err
	}

	store.AddPool(storeName, db)

	return db, nil
}

// CloseSciAndDB closes provided state context and db.
//
// NOTE: it panics if util.NodeDB of the provided context is not implemented by util.PNodeDB.
func CloseSciAndDB(sci chain.StateContextI, db *gorocksdb.TransactionDB) error {
	if err := mpt.SaveRoot(sci.GetState().GetRoot(), db); err != nil {
		return err
	}

	db.Close()

	pNodeDB, ok := sci.GetState().GetNodeDB().(*util.PNodeDB)
	if !ok {
		return errors.New("must be PNodeDB type")
	}

	pNodeDB.Close()

	return nil
}

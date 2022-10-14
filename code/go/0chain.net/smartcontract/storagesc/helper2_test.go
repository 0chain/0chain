package storagesc

import (
	"0chain.net/chaincore/currency"

	"0chain.net/smartcontract/dbs/event"

	"0chain.net/chaincore/block"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"github.com/0chain/common/core/util"
)

type mockStateContext struct {
	cstate.StateContext
	clientBalance currency.Coin
	store         map[datastore.Key]util.MPTSerializable
	config        *cstate.SCConfig
}

// GetConfig implements state.StateContextI
func (sc *mockStateContext) GetConfig(smartcontract string) (*cstate.SCConfig, error) {
	return sc.config, nil
}

// SetConfig implements state.StateContextI
func (sc *mockStateContext) SetConfig(smartcontract string, config cstate.SCConfig) error {
	sc.config = &config
	return nil
}

type mockBlobberYaml struct {
	serviceCharge    float64
	readPrice        float64
	writePrice       float64
	MaxOfferDuration int64
	minLockDemand    float64
}

var (
	scYaml          = &Config{}
	creationDate    = common.Timestamp(100)
	approvedMinters = []string{
		"6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9", // miner SC
		"cf8d0df9bd8cc637a4ff4e792ffe3686da6220c45f0e1103baa609f3f1751ef4", // interest SC
		"6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7", // storage SC
	}
	storageScId = approvedMinters[2]
)

func (sc *mockStateContext) SetMagicBlock(_ *block.MagicBlock)           {}
func (sc *mockStateContext) GetState() util.MerklePatriciaTrieI          { return nil }
func (sc *mockStateContext) GetTransaction() *transaction.Transaction    { return nil }
func (sc *mockStateContext) GetSignedTransfers() []*state.SignedTransfer { return nil }
func (sc *mockStateContext) Validate() error                             { return nil }
func (sc *mockStateContext) GetSignatureScheme() encryption.SignatureScheme {
	return encryption.NewBLS0ChainScheme()
}

func (tb *mockStateContext) EmitEvent(event.EventType, event.EventTag, string, interface{}, ...cstate.Appender) {
}
func (sc *mockStateContext) EmitError(error)                                       {}
func (sc *mockStateContext) GetEvents() []event.Event                              { return nil }
func (tb *mockStateContext) GetEventDB() *event.EventDb                            { return nil }
func (sc *mockStateContext) AddSignedTransfer(_ *state.SignedTransfer)             {}
func (sc *mockStateContext) DeleteTrieNode(_ datastore.Key) (datastore.Key, error) { return "", nil }
func (sc *mockStateContext) GetChainCurrentMagicBlock() *block.MagicBlock          { return nil }
func (sc *mockStateContext) GetLatestFinalizedBlock() *block.Block                 { return nil }
func (sc *mockStateContext) GetClientBalance(_ datastore.Key) (currency.Coin, error) {
	return sc.clientBalance, nil
}

func (sc *mockStateContext) GetLastestFinalizedMagicBlock() *block.Block {
	return nil
}

func (sc *mockStateContext) SetStateContext(_ *state.State) error { return nil }

func (sc *mockStateContext) GetTrieNode(key datastore.Key, v util.MPTSerializable) error {
	var val, ok = sc.store[key]
	if !ok {
		return util.ErrValueNotPresent
	}
	d, err := val.MarshalMsg(nil)
	if err != nil {
		return err
	}

	_, err = v.UnmarshalMsg(d)
	return err
}

func (sc *mockStateContext) InsertTrieNode(key datastore.Key, node util.MPTSerializable) (datastore.Key, error) {
	sc.store[key] = node
	return key, nil
}

func zcnToInt64(token float64) int64 {
	return int64(token * float64(x10))
}

func zcnToBalance(token float64) currency.Coin {
	return currency.Coin(token * float64(x10))
}

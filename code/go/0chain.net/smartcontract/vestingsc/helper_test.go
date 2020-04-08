package vestingsc

import (
	"context"
	"net/url"
	"testing"
	"time"

	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/util"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestVestingSC() (vsc *VestingSmartContract) {
	vsc = new(StorageSmartContract)
	vsc.SmartContract = new(smartcontractinterface.SmartContract)
	vsc.ID = ADDRESS
	return
}

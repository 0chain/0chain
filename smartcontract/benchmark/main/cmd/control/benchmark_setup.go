package control

import (
	"encoding/json"
	"strconv"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	bk "0chain.net/smartcontract/benchmark"
	"github.com/spf13/viper"
)

//go:generate msgp -io=false -tests=false -unexported=true -v

var (
	controlMKey = datastore.Key("6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7" + encryption.Hash("control_all"))
)

func getControlNKey(index int) datastore.Key {
	return datastore.Key("6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7" +
		encryption.Hash("control_n"+strconv.Itoa(index)))
}

type item struct {
	Field int64 `json:"field"`
}

func (i *item) Encode() []byte {
	var b, err = json.Marshal(i)
	if err != nil {
		panic(err)
	}
	return b
}

func (i *item) Decode(p []byte) error {
	return json.Unmarshal(p, i)
}

type itemArray struct {
	Fields []int64 `json:"fields"`
}

func (is *itemArray) Encode() []byte {
	var b, err = json.Marshal(is)
	if err != nil {
		panic(err)
	}
	return b
}

func (is *itemArray) Decode(p []byte) error {
	return json.Unmarshal(p, is)
}

func AddControlObjects(
	balances cstate.StateContextI,
) {
	m := viper.GetInt(bk.ControlM)
	n := viper.GetInt(bk.ControlN)
	if m == 0 || n > m {
		return
	}
	fields := make([]int64, m)
	for i := 0; i < m; i++ {
		fields[i] = int64(i)
	}

	_, err := balances.InsertTrieNode(controlMKey, &itemArray{
		Fields: fields,
	})
	if err != nil {
		panic(err)
	}

	for i := 0; i < n; i++ {
		_, err := balances.InsertTrieNode(getControlNKey(i), &item{
			Field: int64(i),
		})
		if err != nil {
			panic(err)
		}
	}

	for i := 0; i < n; i++ {
		_, err := balances.InsertTrieNode(getControlNKey(i), &item{
			Field: int64(i),
		})
		if err != nil {
			panic(err)
		}
	}
}

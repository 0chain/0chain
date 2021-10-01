package magmasc

import (
	"reflect"
	"testing"

	zmc "github.com/0chain/gosdk/zmagmacore/magmasc"

	chain "0chain.net/chaincore/chain/state"
)

func Test_Consumers_add(t *testing.T) {
	t.Parallel()

	list, msc, sci := mockConsumers(), mockMagmaSmartContract(), mockStateContextI()
	cons, _ := list.getByIndex(0)
	if _, err := sci.InsertTrieNode(nodeUID(msc.ID, consumerType, cons.Host), cons); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	tests := [3]struct {
		name  string
		cons  *zmc.Consumer
		msc   *MagmaSmartContract
		sci   chain.StateContextI
		list  *Consumers
		error bool
	}{
		{
			name:  "OK",
			cons:  mockConsumer(),
			msc:   msc,
			sci:   sci,
			list:  list,
			error: false,
		},
		{
			name:  "nil_Pointer_ERR",
			cons:  nil,
			msc:   msc,
			sci:   nil,
			list:  list,
			error: true,
		},
		{
			name:  "Consumer_Insert_Trie_Node_ERR",
			cons:  &zmc.Consumer{ExtID: "cannot_insert_id"},
			msc:   msc,
			sci:   sci,
			list:  list,
			error: true,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			// do not use parallel running to avoid detect race conditions because of
			// everything is happening in a single smart contract so there is only one thread
			err := test.list.add(test.msc.ID, test.cons, msc.db, test.sci)
			if (err != nil) != test.error {
				t.Errorf("add() error: %v | want: %v", err, test.error)
			}
		})
	}
}

func Test_Consumers_copy(t *testing.T) {
	t.Parallel()

	list, want := mockConsumers(), &Consumers{}
	if list.Sorted != nil {
		want.Sorted = make([]*zmc.Consumer, len(list.Sorted))
		copy(want.Sorted, list.Sorted)
	}

	tests := [1]struct {
		name string
		list *Consumers
		want *Consumers
	}{
		{
			name: "OK",
			list: list,
			want: want,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.list.copy(); !reflect.DeepEqual(got, test.want) {
				t.Errorf("copy() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_Consumers_del(t *testing.T) {
	t.Parallel()

	msc, sci := mockMagmaSmartContract(), mockStateContextI()

	cons, list := mockConsumer(), &Consumers{}
	if err := list.add(msc.ID, cons, msc.db, sci); err != nil {
		t.Fatalf("add() error: %v | want: %v", err, nil)
	}

	tests := [2]struct {
		name  string
		cons  *zmc.Consumer
		msc   *MagmaSmartContract
		list  *Consumers
		want  *Consumers
		error bool
	}{
		{
			name:  "TRUE",
			cons:  cons,
			msc:   msc,
			list:  list,
			want:  &Consumers{Sorted: make([]*zmc.Consumer, 0)},
			error: false,
		},
		{
			name:  "FALSE",
			cons:  &zmc.Consumer{ExtID: "not_present_id"},
			msc:   msc,
			list:  list,
			want:  &Consumers{Sorted: make([]*zmc.Consumer, 0)},
			error: true,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			// do not use parallel running to avoid detect race conditions because of
			// everything is happening in a single smart contract so there is only one thread
			got, err := test.list.del(test.cons.ExtID, msc.db)
			if (err != nil) != test.error {
				t.Errorf("del() error: %v | want: %v", err, test.error)
				return
			}
			if err == nil && !reflect.DeepEqual(got, test.cons) {
				t.Errorf("del() got: %#v | want: %#v", got, test.cons)
			}
			if !reflect.DeepEqual(test.list, test.want) {
				t.Errorf("del() got: %#v | want: %#v", test.list, test.want)
			}
		})
	}
}

func Test_Consumers_delByIndex(t *testing.T) {
	t.Parallel()

	list, msc := mockConsumers(), mockMagmaSmartContract()

	cons0, _ := list.getByIndex(0)
	cons1, _ := list.getByIndex(1)
	cons2, _ := list.getByIndex(2)
	cons3, _ := list.getByIndex(3)

	tests := [5]struct {
		name  string
		idx   int
		msc   *MagmaSmartContract
		list  *Consumers
		want  *zmc.Consumer
		error bool
	}{
		{
			name:  cons2.ExtID + "_del_OK",
			idx:   2,
			msc:   msc,
			list:  list,
			want:  cons2,
			error: false,
		},
		{
			name:  cons3.ExtID + "_del_OK",
			idx:   2,
			msc:   msc,
			list:  list,
			want:  cons3,
			error: false,
		},
		{
			name:  cons0.ExtID + "_del_OK",
			idx:   0,
			msc:   msc,
			list:  list,
			want:  cons0,
			error: false,
		},
		{
			name:  cons1.ExtID + "_del_OK",
			idx:   0,
			msc:   msc,
			list:  list,
			want:  cons1,
			error: false,
		},
		{
			name:  "nil_Pointer_ERR",
			idx:   len(list.Sorted),
			msc:   msc,
			list:  list,
			want:  nil,
			error: true,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			// do not use parallel running
			// the particular order of tests is important
			got, err := test.list.delByIndex(test.idx, msc.db)
			if (err != nil) != test.error {
				t.Errorf("delByIndex() error: %v | want: %v", err, test.error)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("delByIndex() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_Consumers_get(t *testing.T) {
	t.Parallel()

	idx, list := 0, mockConsumers()
	tests := [2]struct {
		name string
		id   string
		list *Consumers
		want *zmc.Consumer
		ret  bool
	}{
		{
			name: "TRUE",
			id:   list.Sorted[idx].ExtID,
			list: list,
			want: list.Sorted[idx],
			ret:  true,
		},
		{
			name: "FALSE",
			id:   "not_present_id",
			list: list,
			want: nil,
			ret:  false,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, ret := test.list.get(test.id)
			if ret != test.ret {
				t.Errorf("get() return: %v | want: %v", got, test.ret)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("get() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_Consumers_getByHost(t *testing.T) {
	t.Parallel()

	idx, list := 0, mockConsumers()
	tests := [2]struct {
		name string
		host string
		list *Consumers
		want *zmc.Consumer
		ret  bool
	}{
		{
			name: "TRUE",
			host: list.Sorted[idx].Host,
			list: list,
			want: list.Sorted[idx],
			ret:  true,
		},
		{
			name: "FALSE",
			host: "not_present_host",
			list: list,
			want: nil,
			ret:  false,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, ret := test.list.getByHost(test.host)
			if ret != test.ret {
				t.Errorf("get() return: %v | want: %v", got, test.ret)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("get() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_Consumers_getByIndex(t *testing.T) {
	t.Parallel()

	idx, list := 0, mockConsumers()
	tests := [2]struct {
		name string
		idx  int
		list *Consumers
		want *zmc.Consumer
		ret  bool
	}{
		{
			name: "TRUE",
			idx:  idx,
			list: list,
			want: list.Sorted[idx],
			ret:  true,
		},
		{
			name: "FALSE",
			idx:  len(list.Sorted),
			list: list,
			want: nil,
			ret:  false,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, ret := test.list.getByIndex(test.idx)
			if ret != test.ret {
				t.Errorf("get() return: %v | want: %v", got, test.ret)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("get() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_Consumers_getIndex(t *testing.T) {
	t.Parallel()

	idx, list := 0, mockConsumers()
	tests := [2]struct {
		name string
		id   string
		list *Consumers
		want int
		ret  bool
	}{
		{
			name: "TRUE",
			id:   list.Sorted[idx].ExtID,
			list: list,
			want: idx,
			ret:  true,
		},
		{
			name: "FALSE",
			id:   "not_present_id",
			list: &Consumers{},
			want: -1,
			ret:  false,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, ret := test.list.getIndex(test.id)
			if ret != test.ret {
				t.Errorf("getIndex() return: %v | want: %v", got, test.ret)
			}
			if got != test.want {
				t.Errorf("getIndex() got: %v | want: %v", got, test.want)
			}
		})
	}
}

func Test_Consumers_put(t *testing.T) {
	t.Parallel()

	list := Consumers{}
	cons0 := zmc.Consumer{ExtID: "0"}
	cons1 := zmc.Consumer{ExtID: "1"}
	cons2 := zmc.Consumer{ExtID: "2"}
	cons3 := zmc.Consumer{ExtID: "3"}

	tests := [6]struct {
		name string
		cons *zmc.Consumer
		list *Consumers
		want []*zmc.Consumer
		ret  bool
	}{
		{
			name: "nil_Pointer_ERR",
			cons: nil,
			list: &list,
			want: nil,
			ret:  false,
		},
		{
			name: "TRUE", // appended
			cons: &cons2,
			list: &list,
			want: []*zmc.Consumer{&cons2},
			ret:  true,
		},
		{
			name: "APPEND", // appended
			cons: &cons3,
			list: &list,
			want: []*zmc.Consumer{&cons2, &cons3},
			ret:  true,
		},
		{
			name: "PREPEND", // inserted
			cons: &cons0,
			list: &list,
			want: []*zmc.Consumer{&cons0, &cons2, &cons3},
			ret:  true,
		},
		{
			name: "INSERT", // inserted
			cons: &cons1,
			list: &list,
			want: []*zmc.Consumer{&cons0, &cons1, &cons2, &cons3},
			ret:  true,
		},
		{
			name: "FALSE", // already have
			cons: &cons3,
			list: &list,
			want: []*zmc.Consumer{&cons0, &cons1, &cons2, &cons3},
			ret:  false,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			// do not use parallel running
			// the particular order of tests is important
			if _, got := test.list.put(test.cons); got != test.ret {
				t.Errorf("add() return: %v | want: %v", got, test.ret)
			}
			if !reflect.DeepEqual(test.list.Sorted, test.want) {
				t.Errorf("add() sorted: %#v | want: %#v", test.list.Sorted, test.want)
			}
		})
	}
}

func Test_Consumers_write(t *testing.T) {
	t.Parallel()

	cons, msc, sci := mockConsumer(), mockMagmaSmartContract(), mockStateContextI()

	list := &Consumers{}
	tests := [2]struct {
		name  string
		cons  *zmc.Consumer
		msc   *MagmaSmartContract
		sci   chain.StateContextI
		list  *Consumers
		error bool
	}{
		{
			name:  "OK",
			cons:  cons,
			msc:   msc,
			sci:   sci,
			list:  list,
			error: false,
		},
		{
			name:  "nil_Pointer_ERR",
			cons:  nil,
			msc:   msc,
			sci:   nil,
			list:  list,
			error: true,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			// do not use parallel running to avoid detect race conditions because of
			// everything is happening in a single smart contract so there is only one thread
			err := test.list.write(test.msc.ID, test.cons, msc.db, test.sci)
			if (err != nil) != test.error {
				t.Errorf("write() error: %v | want: %v", err, test.error)
			}
		})
	}
}

func Test_consumersFetch(t *testing.T) {
	t.Parallel()

	list, msc, sci := mockConsumers(), mockMagmaSmartContract(), mockStateContextI()
	if err := list.add(msc.ID, mockConsumer(), msc.db, sci); err != nil {
		t.Fatalf("add() error: %v | want: %v", err, nil)
	}

	tests := [2]struct {
		name  string
		id    string
		msc   *MagmaSmartContract
		want  *Consumers
		error bool
	}{
		{
			name:  "OK",
			id:    AllConsumersKey,
			msc:   msc,
			want:  list,
			error: false,
		},
		{
			name:  "Not_Present_OK",
			id:    "not_present_id",
			msc:   msc,
			want:  &Consumers{},
			error: false,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := consumersFetch(test.id, msc.db)
			if err == nil && !reflect.DeepEqual(got, test.want) {
				t.Errorf("consumersFetch() got: %#v | want: %#v", got, test.want)
				return
			}
			if (err != nil) != test.error {
				t.Errorf("consumersFetch() error: %v | want: %v", err, test.error)
			}
		})
	}
}

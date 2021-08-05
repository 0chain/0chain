package magmasc

import (
	"reflect"
	"testing"

	bmp "github.com/0chain/bandwidth_marketplace/code/core/magmasc"

	chain "0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
	store "0chain.net/core/ememorystore"
)

func Test_Providers_add(t *testing.T) {
	t.Parallel()

	list, msc, sci := mockProviders(), mockMagmaSmartContract(), mockStateContextI()
	provRegistered, _ := list.getByIndex(0)

	tests := [3]struct {
		name  string
		prov  *bmp.Provider
		msc   *MagmaSmartContract
		sci   chain.StateContextI
		list  *Providers
		error bool
	}{
		{
			name:  "OK",
			prov:  mockProvider(),
			msc:   msc,
			sci:   sci,
			list:  list,
			error: false,
		},
		{
			name:  "Provider_Host_Already_Registered_ERR",
			prov:  provRegistered,
			msc:   msc,
			sci:   sci,
			list:  list,
			error: true,
		},
		{
			name:  "Provider_Insert_Trie_Node_ERR",
			prov:  &bmp.Provider{ExtID: "cannot_insert_id"},
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
			err := test.list.add(test.msc.ID, test.prov, store.GetTransaction(test.msc.db), test.sci)
			if (err != nil) != test.error {
				t.Errorf("add() error: %v | want: %v", err, test.error)
			}
		})
	}
}

func Test_Providers_del(t *testing.T) {
	t.Parallel()

	msc, sci := mockMagmaSmartContract(), mockStateContextI()

	prov, list := mockProvider(), &Providers{}
	if err := list.add(msc.ID, prov, store.GetTransaction(msc.db), sci); err != nil {
		t.Fatalf("add() error: %v | want: %v", err, nil)
	}

	tests := [2]struct {
		name  string
		prov  *bmp.Provider
		msc   *MagmaSmartContract
		list  *Providers
		want  *Providers
		error bool
	}{
		{
			name:  "TRUE",
			prov:  prov,
			msc:   msc,
			list:  list,
			want:  &Providers{Sorted: make([]*bmp.Provider, 0)},
			error: false,
		},
		{
			name:  "FALSE",
			prov:  &bmp.Provider{ExtID: "not_present_id"},
			msc:   msc,
			list:  list,
			want:  &Providers{Sorted: make([]*bmp.Provider, 0)},
			error: true,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			// do not use parallel running to avoid detect race conditions because of
			// everything is happening in a single smart contract so there is only one thread
			got, err := test.list.del(test.prov.ExtID, store.GetTransaction(test.msc.db))
			if (err != nil) != test.error {
				t.Errorf("del() error: %v | want: %v", err, test.error)
				return
			}
			if err == nil && !reflect.DeepEqual(got, test.prov) {
				t.Errorf("del() got: %#v | want: %#v", got, test.prov)
			}
			if !reflect.DeepEqual(test.list, test.want) {
				t.Errorf("del() got: %#v | want: %#v", test.list, test.want)
			}
		})
	}
}

func Test_Providers_delByIndex(t *testing.T) {
	t.Parallel()

	list, msc := mockProviders(), mockMagmaSmartContract()

	prov0, _ := list.getByIndex(0)
	prov1, _ := list.getByIndex(1)
	prov2, _ := list.getByIndex(2)
	prov3, _ := list.getByIndex(3)

	tests := [4]struct {
		name  string
		idx   int
		msc   *MagmaSmartContract
		list  *Providers
		want  *bmp.Provider
		error bool
	}{
		{
			name:  prov2.ExtID + "_del_OK",
			idx:   2,
			msc:   msc,
			list:  list,
			want:  prov2,
			error: false,
		},
		{
			name:  prov3.ExtID + "_del_OK",
			idx:   2,
			msc:   msc,
			list:  list,
			want:  prov3,
			error: false,
		},
		{
			name:  prov0.ExtID + "_del_OK",
			idx:   0,
			msc:   msc,
			list:  list,
			want:  prov0,
			error: false,
		},
		{
			name:  prov1.ExtID + "_del_OK",
			idx:   0,
			msc:   msc,
			list:  list,
			want:  prov1,
			error: false,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			// do not use parallel running
			// the particular order of tests is important
			got, err := test.list.delByIndex(test.idx, store.GetTransaction(test.msc.db))
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

func Test_Providers_get(t *testing.T) {
	t.Parallel()

	list := mockProviders()
	pros := list.Sorted[0]

	tests := [2]struct {
		name string
		id   string
		list *Providers
		want *bmp.Provider
		ret  bool
	}{
		{
			name: "TRUE",
			id:   pros.ExtID,
			list: list,
			want: pros,
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

func Test_Providers_getIndex(t *testing.T) {
	t.Parallel()

	idx, list := 0, mockProviders()
	id := list.Sorted[idx].ExtID

	tests := [2]struct {
		name string
		id   string
		list *Providers
		want int
		ret  bool
	}{
		{
			name: "TRUE",
			id:   id,
			list: list,
			want: idx,
			ret:  true,
		},
		{
			name: "FALSE",
			id:   "not_present_id",
			list: &Providers{},
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

func Test_Providers_put(t *testing.T) {
	t.Parallel()

	list := Providers{}
	prov0 := bmp.Provider{ExtID: "0"}
	prov1 := bmp.Provider{ExtID: "1"}
	prov2 := bmp.Provider{ExtID: "2"}
	prov3 := bmp.Provider{ExtID: "3"}

	tests := [5]struct {
		name string
		pros *bmp.Provider
		list *Providers
		want []*bmp.Provider
		ret  bool
	}{
		{
			name: "TRUE", // appended
			pros: &prov2,
			list: &list,
			want: []*bmp.Provider{&prov2},
			ret:  true,
		},
		{
			name: "APPEND", // appended
			pros: &prov3,
			list: &list,
			want: []*bmp.Provider{&prov2, &prov3},
			ret:  true,
		},
		{
			name: "PREPEND", // inserted
			pros: &prov0,
			list: &list,
			want: []*bmp.Provider{&prov0, &prov2, &prov3},
			ret:  true,
		},
		{
			name: "INSERT", // inserted
			pros: &prov1,
			list: &list,
			want: []*bmp.Provider{&prov0, &prov1, &prov2, &prov3},
			ret:  true,
		},
		{
			name: "FALSE", // already have
			pros: &prov3,
			list: &list,
			want: []*bmp.Provider{&prov0, &prov1, &prov2, &prov3},
			ret:  false,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			// do not use parallel running
			// the particular order of tests is important
			if _, got := test.list.put(test.pros); got != test.ret {
				t.Errorf("add() return: %v | want: %v", got, test.ret)
			}
			if !reflect.DeepEqual(test.list.Sorted, test.want) {
				t.Errorf("add() sorted: %#v | want: %#v", test.list.Sorted, test.want)
			}
		})
	}
}

func Test_Providers_write(t *testing.T) {
	t.Parallel()

	prov, msc, sci := mockProvider(), mockMagmaSmartContract(), mockStateContextI()

	list := &Providers{}
	if err := list.add(msc.ID, prov, store.GetTransaction(msc.db), sci); err != nil {
		t.Fatalf("add() error: %v | want: %v", err, nil)
	}

	tests := [1]struct {
		name  string
		prov  *bmp.Provider
		msc   *MagmaSmartContract
		sci   chain.StateContextI
		list  *Providers
		error bool
	}{
		{
			name:  "OK",
			prov:  prov,
			msc:   msc,
			sci:   sci,
			list:  list,
			error: false,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			// do not use parallel running to avoid detect race conditions because of
			// everything is happening in a single smart contract so there is only one thread
			err := test.list.write(test.msc.ID, test.prov, store.GetTransaction(test.msc.db), test.sci)
			if (err != nil) != test.error {
				t.Errorf("write() error: %v | want: %v", err, test.error)
			}
		})
	}
}

func Test_fetchProviders(t *testing.T) {
	t.Parallel()

	list, msc, sci := mockProviders(), mockMagmaSmartContract(), mockStateContextI()
	if err := list.add(msc.ID, mockProvider(), store.GetTransaction(msc.db), sci); err != nil {
		t.Fatalf("add() error: %v | want: %v", err, nil)
	}

	tests := [2]struct {
		name  string
		id    datastore.Key
		msc   *MagmaSmartContract
		want  *Providers
		error bool
	}{
		{
			name:  "OK",
			id:    AllProvidersKey,
			msc:   msc,
			want:  list,
			error: false,
		},
		{
			name:  "Not_Present_OK",
			id:    "not_present_id",
			msc:   msc,
			want:  &Providers{},
			error: false,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := fetchProviders(test.id, store.GetTransaction(test.msc.db))
			if err == nil && !reflect.DeepEqual(got, test.want) {
				t.Errorf("fetchProviders() got: %#v | want: %#v", got, test.want)
				return
			}
			if (err != nil) != test.error {
				t.Errorf("fetchProviders() error: %v | want: %v", err, test.error)
			}
		})
	}
}

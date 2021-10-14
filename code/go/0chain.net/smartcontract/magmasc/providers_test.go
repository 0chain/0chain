package magmasc

import (
	"encoding/json"
	"math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/0chain/gosdk/zmagmacore/magmasc/pb"

	zmc "github.com/0chain/gosdk/zmagmacore/magmasc"

	chain "0chain.net/chaincore/chain/state"
)

func Test_Providers_add(t *testing.T) {
	t.Parallel()

	list, msc, sci := mockProviders(), mockMagmaSmartContract(), mockStateContextI()
	prov, _ := list.getByIndex(0)
	if _, err := sci.InsertTrieNode(nodeUID(msc.ID, providerType, prov.Host), prov); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	tests := [3]struct {
		name  string
		prov  *zmc.Provider
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
			name:  "nil_Pointer_ERR",
			prov:  nil,
			msc:   msc,
			sci:   nil,
			list:  list,
			error: true,
		},
		{
			name: "Provider_Insert_Trie_Node_ERR",
			prov: &zmc.Provider{
				Provider: &pb.Provider{
					ExtId: "cannot_insert_id",
				},
			},
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
			err := test.list.add(test.msc.ID, test.prov, msc.db, test.sci)
			if (err != nil) != test.error {
				t.Errorf("add() error: %v | want: %v", err, test.error)
			}
		})
	}
}

func Test_Providers_copy(t *testing.T) {
	t.Parallel()

	list, want := mockProviders(), &Providers{}
	if list.Sorted != nil {
		want.Sorted = make([]*zmc.Provider, len(list.Sorted))
		copy(want.Sorted, list.Sorted)
	}

	tests := [1]struct {
		name string
		list *Providers
		want *Providers
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

func Test_Providers_del(t *testing.T) {
	t.Parallel()

	msc, sci := mockMagmaSmartContract(), mockStateContextI()

	prov, list := mockProvider(), &Providers{}
	if err := list.add(msc.ID, prov, msc.db, sci); err != nil {
		t.Fatalf("add() error: %v | want: %v", err, nil)
	}

	tests := [2]struct {
		name  string
		prov  *zmc.Provider
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
			want:  &Providers{Sorted: make([]*zmc.Provider, 0)},
			error: false,
		},
		{
			name: "FALSE",
			prov: &zmc.Provider{
				Provider: &pb.Provider{
					ExtId: "not_present_id",
				},
			},
			msc:   msc,
			list:  list,
			want:  &Providers{Sorted: make([]*zmc.Provider, 0)},
			error: true,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			// do not use parallel running to avoid detect race conditions because of
			// everything is happening in a single smart contract so there is only one thread
			got, err := test.list.del(test.prov.ExtId, msc.db)
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

	tests := [5]struct {
		name  string
		idx   int
		msc   *MagmaSmartContract
		list  *Providers
		want  *zmc.Provider
		error bool
	}{
		{
			name:  prov2.ExtId + "_del_OK",
			idx:   2,
			msc:   msc,
			list:  list,
			want:  prov2,
			error: false,
		},
		{
			name:  prov3.ExtId + "_del_OK",
			idx:   2,
			msc:   msc,
			list:  list,
			want:  prov3,
			error: false,
		},
		{
			name:  prov0.ExtId + "_del_OK",
			idx:   0,
			msc:   msc,
			list:  list,
			want:  prov0,
			error: false,
		},
		{
			name:  prov1.ExtId + "_del_OK",
			idx:   0,
			msc:   msc,
			list:  list,
			want:  prov1,
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

func Test_Providers_get(t *testing.T) {
	t.Parallel()

	idx, list := 0, mockProviders()
	tests := [2]struct {
		name string
		id   string
		list *Providers
		want *zmc.Provider
		ret  bool
	}{
		{
			name: "TRUE",
			id:   list.Sorted[idx].ExtId,
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

func Test_Providers_getByHost(t *testing.T) {
	t.Parallel()

	idx, list := 0, mockProviders()
	tests := [2]struct {
		name string
		host string
		list *Providers
		want *zmc.Provider
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

func Test_Providers_getByIndex(t *testing.T) {
	t.Parallel()

	idx, list := 0, mockProviders()
	tests := [2]struct {
		name string
		idx  int
		list *Providers
		want *zmc.Provider
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

func Test_Providers_getIndex(t *testing.T) {
	t.Parallel()

	idx, list := 0, mockProviders()
	tests := [2]struct {
		name string
		id   string
		list *Providers
		want int
		ret  bool
	}{
		{
			name: "TRUE",
			id:   list.Sorted[idx].ExtId,
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
	prov0 := zmc.Provider{
		Provider: &pb.Provider{
			ExtId: "0",
		},
	}
	prov1 := zmc.Provider{
		Provider: &pb.Provider{
			ExtId: "1",
		},
	}
	prov2 := zmc.Provider{
		Provider: &pb.Provider{
			ExtId: "2",
		},
	}
	prov3 := zmc.Provider{
		Provider: &pb.Provider{
			ExtId: "3",
		},
	}

	tests := [6]struct {
		name string
		prov *zmc.Provider
		list *Providers
		want []*zmc.Provider
		ret  bool
	}{
		{
			name: "nil_Pointer_ERR",
			prov: nil,
			list: &list,
			want: nil,
			ret:  false,
		},
		{
			name: "TRUE", // appended
			prov: &prov2,
			list: &list,
			want: []*zmc.Provider{&prov2},
			ret:  true,
		},
		{
			name: "APPEND", // appended
			prov: &prov3,
			list: &list,
			want: []*zmc.Provider{&prov2, &prov3},
			ret:  true,
		},
		{
			name: "PREPEND", // inserted
			prov: &prov0,
			list: &list,
			want: []*zmc.Provider{&prov0, &prov2, &prov3},
			ret:  true,
		},
		{
			name: "INSERT", // inserted
			prov: &prov1,
			list: &list,
			want: []*zmc.Provider{&prov0, &prov1, &prov2, &prov3},
			ret:  true,
		},
		{
			name: "FALSE", // already have
			prov: &prov3,
			list: &list,
			want: []*zmc.Provider{&prov0, &prov1, &prov2, &prov3},
			ret:  false,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			// do not use parallel running
			// the particular order of tests is important
			if _, got := test.list.put(test.prov); got != test.ret {
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

	list, msc, sci := &Providers{}, mockMagmaSmartContract(), mockStateContextI()
	tests := [2]struct {
		name  string
		prov  *zmc.Provider
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
			name:  "nil_Pointer_ERR",
			prov:  nil,
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
			err := test.list.write(test.msc.ID, test.prov, msc.db, test.sci)
			if (err != nil) != test.error {
				t.Errorf("write() error: %v | want: %v", err, test.error)
			}
		})
	}
}

func Test_providersFetch(t *testing.T) {
	t.Parallel()

	list, msc, sci := mockProviders(), mockMagmaSmartContract(), mockStateContextI()
	if err := list.add(msc.ID, mockProvider(), msc.db, sci); err != nil {
		t.Fatalf("add() error: %v | want: %v", err, nil)
	}

	tests := [2]struct {
		name  string
		id    string
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

			got, err := providersFetch(test.id, msc.db)
			resp, _ := json.Marshal(got)
			want, _ := json.Marshal(test.want)

			if err == nil && !reflect.DeepEqual(resp, want) {
				t.Errorf("providersFetch() got: %#v | want: %#v", got, test.want)
				return
			}
			if (err != nil) != test.error {
				t.Errorf("providersFetch() error: %v | want: %v", err, test.error)
			}
		})
	}
}

func TestProviders_random(t *testing.T) {
	t.Parallel()

	list := mockProviders()

	seed := time.Now().Unix()
	rand.Seed(seed)
	randInd := rand.Intn(len(list.Sorted))

	tests := []struct {
		name      string
		providers *Providers
		seed      int64
		want      *zmc.Provider
		wantErr   bool
	}{
		{
			name:      "OK",
			providers: list,
			seed:      seed,
			want:      list.Sorted[randInd],
			wantErr:   false,
		},
		{
			name:      "Empty_List_ERR",
			providers: &Providers{},
			seed:      seed,
			want:      nil,
			wantErr:   true,
		},
	}
	for ind := range tests {
		test := tests[ind]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := test.providers.random(test.seed)
			if (err != nil) != test.wantErr {
				t.Errorf("random() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("random() got = %v, want %v", got, test.want)
			}
		})
	}
}

package magmasc

import (
	"encoding/json"
	"reflect"
	"testing"

	zmc "github.com/0chain/gosdk/zmagmacore/magmasc"
	"github.com/0chain/gosdk/zmagmacore/magmasc/pb"

	chain "0chain.net/chaincore/chain/state"
)

func Test_Users_add(t *testing.T) {

	list, msc, sci := mockUsers(), mockMagmaSmartContract(), mockStateContextI()
	user, _ := list.getByIndex(0)
	if _, err := sci.InsertTrieNode(nodeUID(msc.ID, userType, user.Id), user); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	tests := [4]struct {
		name  string
		user  *zmc.User
		msc   *MagmaSmartContract
		sci   chain.StateContextI
		list  *Users
		error bool
	}{
		{
			name:  "OK",
			user:  mockUser(),
			msc:   msc,
			sci:   sci,
			list:  list,
			error: false,
		},
		{
			name:  "nil_Pointer_ERR",
			user:  nil,
			msc:   msc,
			sci:   nil,
			list:  list,
			error: true,
		},
		{
			name:  "Adding_An_Existing_User_ERR",
			user:  user,
			msc:   msc,
			sci:   sci,
			list:  list,
			error: true,
		},
		{
			name:  "User_Insert_ERR",
			user:  &zmc.User{User: &pb.User{Id: "cannot_insert_id"}},
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
			err := test.list.add(test.msc.ID, test.user, msc.db, test.sci)
			if (err != nil) != test.error {
				t.Errorf("add() error: %v | want: %v", err, test.error)
			}
		})
	}
}

func Test_Users_copy(t *testing.T) {
	t.Parallel()

	list, want := mockUsers(), &Users{}
	if list.Sorted != nil {
		want.Sorted = make([]*zmc.User, len(list.Sorted))
		copy(want.Sorted, list.Sorted)
	}

	tests := [1]struct {
		name string
		list *Users
		want *Users
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

func Test_Users_del(t *testing.T) {
	t.Parallel()

	msc, sci := mockMagmaSmartContract(), mockStateContextI()

	user, list := mockUser(), &Users{}
	if err := list.add(msc.ID, user, msc.db, sci); err != nil {
		t.Fatalf("add() error: %v | want: %v", err, nil)
	}

	tests := [2]struct {
		name  string
		user  *zmc.User
		msc   *MagmaSmartContract
		list  *Users
		want  *Users
		error bool
	}{
		{
			name:  "OK",
			user:  user,
			msc:   msc,
			list:  list,
			want:  &Users{Sorted: make([]*zmc.User, 0)},
			error: false,
		},
		{
			name:  "Delete_Not_Present_ID_ERR",
			user:  &zmc.User{User: &pb.User{Id: "not_present_id"}},
			msc:   msc,
			list:  list,
			want:  &Users{Sorted: make([]*zmc.User, 0)},
			error: true,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			// do not use parallel running to avoid detect race conditions because of
			// everything is happening in a single smart contract so there is only one thread
			got, err := test.list.del(test.user.Id, msc.db)
			if (err != nil) != test.error {
				t.Errorf("del() error: %v | want: %v", err, test.error)
				return
			}
			if err == nil && !reflect.DeepEqual(got, test.user) {
				t.Errorf("del() got: %#v | want: %#v", got, test.user)
			}
			if !reflect.DeepEqual(test.list, test.want) {
				t.Errorf("del() got: %#v | want: %#v", test.list, test.want)
			}
		})
	}
}

func Test_Users_delByIndex(t *testing.T) {
	t.Parallel()

	list, msc := mockUsers(), mockMagmaSmartContract()

	user0, _ := list.getByIndex(0)
	user1, _ := list.getByIndex(1)
	user2, _ := list.getByIndex(2)
	user3, _ := list.getByIndex(3)

	tests := [6]struct {
		name  string
		idx   int
		msc   *MagmaSmartContract
		list  *Users
		want  *zmc.User
		error bool
	}{
		{
			name:  user2.Id + "_del_OK",
			idx:   2,
			msc:   msc,
			list:  list,
			want:  user2,
			error: false,
		},
		{
			name:  user3.Id + "_del_OK",
			idx:   2,
			msc:   msc,
			list:  list,
			want:  user3,
			error: false,
		},
		{
			name:  user0.Id + "_del_OK",
			idx:   0,
			msc:   msc,
			list:  list,
			want:  user0,
			error: false,
		},
		{
			name:  user1.Id + "_del_OK",
			idx:   0,
			msc:   msc,
			list:  list,
			want:  user1,
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
		{
			name:  "Index_Out_Of Range_ERR",
			idx:   -1,
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

func Test_Users_get(t *testing.T) {
	t.Parallel()

	idx, list := 0, mockUsers()
	tests := [2]struct {
		name string
		id   string
		list *Users
		want *zmc.User
		ret  bool
	}{
		{
			name: "TRUE",
			id:   list.Sorted[idx].Id,
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

func Test_Users_getByConsumer(t *testing.T) {
	t.Parallel()

	idx, list := 0, mockUsers()
	tests := [2]struct {
		name string
		cons string
		list *Users
		want *zmc.User
		ret  bool
	}{
		{
			name: "TRUE",
			cons: list.Sorted[idx].ConsumerId,
			list: list,
			want: list.Sorted[idx],
			ret:  true,
		},
		{
			name: "FALSE",
			cons: "not_present_consumer",
			list: list,
			want: nil,
			ret:  false,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, ret := test.list.getByConsumer(test.cons)
			if ret != test.ret {
				t.Errorf("get() return: %v | want: %v", got, test.ret)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("get() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_Users_getByIndex(t *testing.T) {
	t.Parallel()

	idx, list := 0, mockUsers()
	tests := [3]struct {
		name string
		idx  int
		list *Users
		want *zmc.User
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
		{
			name: "FALSE",
			idx:  -1,
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

func Test_Users_getIndex(t *testing.T) {
	t.Parallel()

	idx, list := 0, mockUsers()
	tests := [2]struct {
		name string
		id   string
		list *Users
		want int
		ret  bool
	}{
		{
			name: "TRUE",
			id:   list.Sorted[idx].Id,
			list: list,
			want: idx,
			ret:  true,
		},
		{
			name: "FALSE",
			id:   "not_present_id",
			list: &Users{},
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

func Test_Users_put(t *testing.T) {
	t.Parallel()

	list := Users{}
	user0 := zmc.User{User: &pb.User{Id: "0"}}
	user1 := zmc.User{User: &pb.User{Id: "1"}}
	user2 := zmc.User{User: &pb.User{Id: "2"}}
	user3 := zmc.User{User: &pb.User{Id: "3"}}

	tests := [6]struct {
		name string
		user *zmc.User
		list *Users
		want []*zmc.User
		ret  bool
	}{
		{
			name: "nil_Pointer_ERR",
			user: nil,
			list: &list,
			want: nil,
			ret:  false,
		},
		{
			name: "TRUE", // appended
			user: &user2,
			list: &list,
			want: []*zmc.User{&user2},
			ret:  true,
		},
		{
			name: "APPEND", // appended
			user: &user3,
			list: &list,
			want: []*zmc.User{&user2, &user3},
			ret:  true,
		},
		{
			name: "PREPEND", // inserted
			user: &user0,
			list: &list,
			want: []*zmc.User{&user0, &user2, &user3},
			ret:  true,
		},
		{
			name: "INSERT", // inserted
			user: &user1,
			list: &list,
			want: []*zmc.User{&user0, &user1, &user2, &user3},
			ret:  true,
		},
		{
			name: "FALSE", // already have
			user: &user3,
			list: &list,
			want: []*zmc.User{&user0, &user1, &user2, &user3},
			ret:  false,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			// do not use parallel running
			// the particular order of tests is important
			if _, got := test.list.put(test.user); got != test.ret {
				t.Errorf("add() return: %v | want: %v", got, test.ret)
			}
			if !reflect.DeepEqual(test.list.Sorted, test.want) {
				t.Errorf("add() sorted: %#v | want: %#v", test.list.Sorted, test.want)
			}
		})
	}
}

func Test_Users_write(t *testing.T) {
	t.Parallel()

	list, msc, sci := &Users{}, mockMagmaSmartContract(), mockStateContextI()
	tests := [2]struct {
		name  string
		user  *zmc.User
		msc   *MagmaSmartContract
		sci   chain.StateContextI
		list  *Users
		error bool
	}{
		{
			name:  "OK",
			user:  mockUser(),
			msc:   msc,
			sci:   sci,
			list:  list,
			error: false,
		},
		{
			name:  "nil_Pointer_ERR",
			user:  nil,
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
			err := test.list.write(test.msc.ID, test.user, msc.db, test.sci)
			if (err != nil) != test.error {
				t.Errorf("write() error: %v | want: %v", err, test.error)
			}
		})
	}
}

func Test_usersFetch(t *testing.T) {
	t.Parallel()

	list, msc, sci := mockUsers(), mockMagmaSmartContract(), mockStateContextI()
	if err := list.add(msc.ID, mockUser(), msc.db, sci); err != nil {
		t.Fatalf("add() error: %v | want: %v", err, nil)
	}

	tests := [2]struct {
		name  string
		id    string
		msc   *MagmaSmartContract
		want  *Users
		error bool
	}{
		{
			name:  "OK",
			id:    allUsersKey,
			msc:   msc,
			want:  list,
			error: false,
		},
		{
			name:  "Not_Present_OK",
			id:    "not_present_id",
			msc:   msc,
			want:  &Users{},
			error: false,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			users, err := usersFetch(test.id, msc.db)
			got, _ := json.Marshal(users)
			want, _ := json.Marshal(test.want)

			if err == nil && !reflect.DeepEqual(got, want) {
				t.Errorf("usersFetch() got: %#v | want: %#v", got, want)
				return
			}
			if (err != nil) != test.error {
				t.Errorf("usersFetch() error: %v | want: %v", err, test.error)
			}
		})
	}
}

package entitywrapper

import (
	"errors"
	"fmt"
)

//msgp:ignore Foo stateInMemory Wrapper
//go:generate msgp -v -tests=false -io=false -unexported

type EntityI interface {
	MarshalMsg([]byte) ([]byte, error)
	UnmarshalMsg([]byte) ([]byte, error)
}

type Wrapper struct {
	version        string
	key            string
	v              EntityI
	newWntityFuncs map[string]func() EntityI
}

func NewWrapper(key string) *Wrapper {
	return &Wrapper{
		key:            key,
		newWntityFuncs: make(map[string]func() EntityI),
	}
}

func (w *Wrapper) Register(version string, f func() EntityI) {
	if _, ok := w.newWntityFuncs[version]; ok {
		panic(fmt.Sprintf("entity version already registered: %v", version))
	}
	w.newWntityFuncs[version] = f
}

func (w *Wrapper) NewEntity(version string) (EntityI, bool) {
	f, ok := w.newWntityFuncs[version]
	if ok {
		return f(), true
	}
	return nil, false
}

func (w *Wrapper) MarshalMsg(b []byte) ([]byte, error) {
	if w.v == nil {
		return nil, errors.New("entity not set")
	}
	return w.v.MarshalMsg(nil)
}

func (w *Wrapper) UnmarshalMsg(b []byte) ([]byte, error) {
	newEntity, ok := w.newWntityFuncs[w.version]
	if !ok {
		return nil, fmt.Errorf("unknown version: %s", w.version)
	}

	e := newEntity()
	v, err := e.UnmarshalMsg(b)
	if err != nil {
		return nil, err
	}

	w.v = e
	return v, err
}

// type fooActivatorMap map[string]func() EntityI

// func (fa fooActivatorMap) NewEntity(name string) (EntityI, bool) {
// 	f, ok := fa[name]
// 	if ok {
// 		return f(), true
// 	}
// 	return nil, false
// }

// var fooActivator = fooActivatorMap{
// 	"origin": func() EntityI {
// 		return &foo{}
// 	},
// 	"hf_one": func() EntityI {
// 		return &fooV2{}
// 	},
// 	"hf_two": func() EntityI {
// 		return &fooV3{}
// 	},
// }

// func WithActivator(name string, round int, before, after func()) {
// 	if round < hardforks[name] {
// 		before()
// 	} else {
// 		after()
// 	}
// }

// type Foo struct {
// 	Name string
// 	Fork string
// 	V    EntityI
// }

// type foo struct {
// 	ID string `msg:"id"`
// }

// type fooV2 struct {
// 	ID   string `msg:"id"`
// 	Name string `msg:"name"`
// }

// type fooV3 struct {
// 	Name string `msg:"name"`
// }

// func (f *Foo) MarshalMsg(b []byte) ([]byte, error) {
// 	return f.V.MarshalMsg(nil)
// }

// func (f *Foo) UnmarshalMsg(b []byte) ([]byte, error) {
// 	e, ok := fooActivator.NewEntity(f.Fork)
// 	if !ok {
// 		return nil, fmt.Errorf("unknown hardfork name: %s", f.Fork)
// 	}

// 	v, err := e.UnmarshalMsg(b)
// 	if err != nil {
// 		return nil, err
// 	}

// 	f.V = e
// 	return v, err
// }

// type stateInMemory struct {
// 	VS map[string][]byte
// }

// func (s *stateInMemory) Insert(key string, value []byte) error {
// 	s.VS[key] = value
// 	saveState(s)
// 	return nil
// }

// func (s *stateInMemory) Get(key string) ([]byte, error) {
// 	return s.VS[key], nil
// }

// func newState() (*stateInMemory, error) {
// 	state := &stateInMemory{
// 		VS: make(map[string][]byte),
// 	}
// 	// Check if the file exists
// 	_, err := os.Stat("state.json")
// 	if os.IsNotExist(err) {
// 		// Create a new empty state
// 		// Save the state to a new JSON file
// 		err := saveState(state)
// 		if err != nil {
// 			return nil, err
// 		}
// 		return state, nil
// 	}

// 	// Load the state from the JSON file
// 	data, err := ioutil.ReadFile("state.json")
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Unmarshal the JSON data into a stateInMemory struct
// 	err = json.Unmarshal(data, state)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return state, nil
// }

// func saveState(state *stateInMemory) error {
// 	// Marshal the state to JSON
// 	data, err := json.Marshal(state)
// 	if err != nil {
// 		return err
// 	}

// 	// Save the JSON data to the file
// 	err = ioutil.WriteFile("state.json", data, 0644)
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

// func main() {
// 	state, err := newState()
// 	if err != nil {
// 		panic(err)
// 	}

// 	f := Foo{
// 		Name: "foo",
// 		Fork: "origin",
// 		V:    &foo{ID: "123"},
// 	}

// 	v, err := f.MarshalMsg(nil)
// 	if err != nil {
// 		panic(err)
// 	}

// 	state.Insert(f.Name, v)

// 	for i := 0; i < 10; i++ {
// 		WithActivator("hf_two", i, func() {
// 			WithActivator("hf_one", i,
// 				beforeHardfork(state, i),
// 				hardforkOne(state, i))
// 		}, func() {
// 			ff := Foo{
// 				Name: "foo",
// 				Fork: "hf_two",
// 			}
// 			v, err := state.Get(ff.Name)
// 			if err != nil {
// 				panic(err)
// 			}

// 			_, err = ff.UnmarshalMsg(v)
// 			if err != nil {
// 				panic(err)
// 			}

// 			v2, err := ff.MarshalMsg(nil)
// 			if err != nil {
// 				panic(err)
// 			}

// 			state.Insert(ff.Name, v2)
// 			fmt.Println("fooV3:", ff.V.(*fooV3), "round:", i)
// 		})
// 	}
// }

// func beforeHardfork(state *stateInMemory, i int) func() {
// 	return func() {
// 		ff := Foo{
// 			Name: "foo",
// 			Fork: "origin",
// 		}
// 		v, err := state.Get(ff.Name)
// 		if err != nil {
// 			panic(err)
// 		}

// 		_, err = ff.UnmarshalMsg(v)
// 		if err != nil {
// 			panic(err)
// 		}

// 		fmt.Println("foo:", ff.V.(*foo), "round:", i)
// 	}
// }

// func hardforkOne(state *stateInMemory, i int) func() {
// 	return func() {
// 		ff := Foo{
// 			Name: "foo",
// 			Fork: "hf_one",
// 		}
// 		v, err := state.Get(ff.Name)
// 		if err != nil {
// 			panic(err)
// 		}

// 		_, err = ff.UnmarshalMsg(v)
// 		if err != nil {
// 			panic(err)
// 		}

// 		ev := ff.V.(*fooV2)
// 		if ev.Name == "" {
// 			ev.Name = "bar"
// 		}

// 		v2, err := ff.MarshalMsg(nil)
// 		if err != nil {
// 			panic(err)
// 		}

// 		state.Insert(ff.Name, v2)
// 		fmt.Println("fooV2:", ff.V.(*fooV2), "round:", i)
// 	}
// }

package entitywrapper

import (
	"errors"
	"fmt"
	"reflect"
)

//msgp:ignore Foo stateInMemory Wrapper
//go:generate msgp -v -tests=false -io=false -unexported

// DefaultOriginVersion is the default version for entity, used for old entity that not changed structs yet.
const DefaultOriginVersion = "v1"

// ErrEntityNotRegistered is returned when entity is not registered in wrapper.
var ErrEntityNotRegistered = errors.New("entity not registered in wrapper")

// gWrapperFuncs is global registered entity with wrappers,
// key as the entity type name, value as the version and related entity create functions
var gWrapperFuncs = make(map[string]entityCreateFuncs)

type entityCreateFuncs map[string]func() EntityI

// EntityI is the interface for entity.
type EntityI interface {
	MarshalMsg([]byte) ([]byte, error)
	UnmarshalMsg([]byte) ([]byte, error)
}

// Wrapper is a wrapper for entity.
type Wrapper struct {
	v EntityI
}

// GetEntityVersionFuncs returns the entity version creators for the given entity name.
func GetEntityVersionFuncs(typeName string) (map[string]func() EntityI, bool) {
	fs, ok := gWrapperFuncs[typeName]
	return fs, ok
}

type EntityWithName interface {
	TypeName() string
	EntityI
}

// RegisterWrapper registers a wrapper with the entity name and entity version creators.
func RegisterWrapper(entity EntityWithName, entityVersionCreators map[string]EntityI) {
	fs := make(map[string]func() EntityI, len(entityVersionCreators))
	for k, v := range entityVersionCreators {
		func(key string, e EntityI) {
			fs[key] = func() EntityI {
				entityType := reflect.TypeOf(e).Elem()
				newEntity := reflect.New(entityType).Interface().(EntityI)
				return newEntity
			}
		}(k, v)
	}

	gWrapperFuncs[entity.TypeName()] = fs
}

func (w *Wrapper) MarshalMsg(b []byte) ([]byte, error) {
	if w.v == nil {
		return nil, errors.New("entity not set")
	}
	return w.v.MarshalMsg(nil)
}

func (w *Wrapper) UnmarshalMsgType(b []byte, typeName string) ([]byte, error) {
	// load version field from data []byte
	ev := &entityVersion{}
	_, err := ev.UnmarshalMsg(b)
	if err != nil {
		return nil, err
	}

	if ev.Version == "" {
		ev.Version = DefaultOriginVersion
	}
	if typeName == "" {
		return nil, errors.New("wrapper name not set")
	}

	fs, ok := gWrapperFuncs[typeName]
	if !ok {
		return nil, fmt.Errorf("entity %v not registered in wrapper", typeName)
	}

	newEntity, ok := fs[ev.Version]
	if !ok {
		return nil, fmt.Errorf("unknown version: %s", ev.Version)
	}

	e := newEntity()
	v, err := e.UnmarshalMsg(b)
	if err != nil {
		return nil, err
	}

	w.v = e
	return v, err
}

func (w *Wrapper) Entity() EntityI {
	return w.v
}

func (w *Wrapper) SetEntity(v EntityI) {
	w.v = v
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

// foo fooV2 and fooV3 are for testing purpose, we need them to generate msgp code.
type foo struct {
	ID string `msg:"id"`
}

type fooV2 struct {
	Version string `msg:"version"`
	ID      string `msg:"id"`
	Name    string `msg:"name"`
}

type fooV3 struct {
	Version string `msg:"version"`
	Name    string `msg:"name"`
	Age     int    `msg:"age"`
}

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

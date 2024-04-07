package entitywrapper

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

//msgp:ignore Foo stateInMemory Wrapper EntityI EntityBaseI WrapperEntity MsgEncodeDecoder MsgEncodeDecoderSize
//go:generate msgp -v -tests=false -io=false -unexported

// DefaultOriginVersion is the default version for entity, used for old entity that not changed structs yet.
const DefaultOriginVersion = "v1"

// ErrEntityNotRegistered is returned when entity is not registered in wrapper.
var ErrEntityNotRegistered = errors.New("entity not registered in wrapper")

// gWrapperFuncs is global registered entity with wrappers,
// key as the entity type name, value as the version and related entity create functions
var gWrapperFuncs = make(map[string]entityCreateFuncs)

type entityCreateFuncs map[string]func() EntityI

type MsgEncodeDecoder interface {
	MarshalMsg([]byte) ([]byte, error)
	UnmarshalMsg([]byte) ([]byte, error)
}

type MsgEncodeDecoderSize interface {
	MsgEncodeDecoder
	Msgsize() int
}

// EntityI is the interface for entity.
type EntityI interface {
	MsgEncodeDecoderSize
	InitVersion()
	GetVersion() string
	GetBase() EntityBaseI
	MigrateFrom(prior EntityI) error
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

type WrapperEntity interface {
	MsgEncodeDecoder
	TypeName() string
}

// RegisterWrapper registers a wrapper with the entity name and entity version creators.
func RegisterWrapper(entity WrapperEntity, entityVersionCreators map[string]EntityI) {
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

type EntityBaseI interface {
	CommitChangesTo(e EntityI)
}

func (w *Wrapper) Base() EntityBaseI {
	return w.v.GetBase()
}

func (w *Wrapper) UpdateBase(f func(EntityBaseI) error) error {
	b := w.Base()
	if err := f(b); err != nil {
		return err
	}
	b.CommitChangesTo(w.v)

	// sn.SetEntity(csn.origin)
	return nil
}

// Update will migrate the wrapper.v entity to the given entity type of `e` if it is
// one version prior to `e`. Otherwise use the wrapper.v entity direct and pass it to the callback function.
// The wrapper.v will be updated after the callback function is executed.
func (w *Wrapper) Update(e EntityI, f func(EntityI) error) error {
	v := w.Entity()
	if v.GetVersion() != e.GetVersion() {
		// migrate the entity when see version not match
		// TODO: do version check, and migrate only if the v.GetVersion() is one version behind e.GetVersion()
		if err := e.MigrateFrom(v); err != nil {
			return err
		}

		v = e
	}

	if err := f(v); err != nil {
		return err
	}
	w.SetEntity(v)
	return nil
}

func (w *Wrapper) MarshalMsg(b []byte) ([]byte, error) {
	if w.v == nil {
		return nil, errors.New("entity not set")
	}
	w.v.InitVersion()
	return w.v.MarshalMsg(nil)
}

func (w *Wrapper) UnmarshalMsgType(b []byte, typeName string) ([]byte, error) {
	// load version field from data []byte
	ev := &entityVersion{}
	_, err := ev.UnmarshalMsg(b)
	if err != nil {
		return nil, err
	}

	fmt.Println("unmarshal version:", ev.Version)
	if ev.Version == "" {
		ev.Version = DefaultOriginVersion
	}

	if typeName == "" {
		return nil, errors.New("wrapper name not set")
	}

	fs, ok := gWrapperFuncs[typeName]
	if !ok {
		return nil, ErrEntityNotRegistered
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

// MarshalJSON implements the json.Marshaler interface.
func (w *Wrapper) MarshalJSON() ([]byte, error) {
	return json.Marshal(w.Entity())
}

func (w *Wrapper) UnmarshalJSONType(data []byte, typeName string) error {
	var ev entityVersion
	if err := json.Unmarshal(data, &ev); err != nil {
		return err
	}

	if ev.Version == "" {
		ev.Version = DefaultOriginVersion
	}

	fs, ok := gWrapperFuncs[typeName]
	if !ok {
		return ErrEntityNotRegistered
	}

	newEntity, ok := fs[ev.Version]
	if !ok {
		return fmt.Errorf("unknown version: %s", ev.Version)
	}

	e := newEntity()
	if err := json.Unmarshal(data, e); err != nil {
		return err
	}

	w.v = e
	return nil
}

func (w *Wrapper) Msgsize() int {
	return w.v.Msgsize()
}

func (w *Wrapper) Entity() EntityI {
	return w.v
}

// func (w *Wrapper) EntityConvert(v EntityI, f func(v EntityI) error) error {
// 	if reflect.TypeOf(v).Name() == reflect.TypeOf(w.v).Name() {
// 		return f(w.v)
// 	}
// 	// convert the entity
// 	return w.v
// }

func (w *Wrapper) SetEntity(v EntityI) {
	w.v = v
}

// func (w *Wrapper) SetEntityType(typeName string) {
// }

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

func (f *foo) GetVersion() string {
	return DefaultOriginVersion
}

func (f *foo) InitVersion() {

}

func (f *foo) TypeName() string {
	return "Foo"
}

func (f *foo) GetBase() EntityBaseI {
	b := fooBase(*f)
	return &b
}

func (f *foo) MigrateFrom(e EntityI) error {
	return nil
}

func (f *fooBase) CommitChangesTo(e EntityI) {
	switch v := e.(type) {
	case *foo:
		*v = foo(*f)
	case *fooV2:
		v.ID = f.ID
	case *fooV3:
		v.ID = f.ID
	}
}

type fooBase foo

type fooV2 struct {
	Version string `msg:"version"`
	ID      string `msg:"id"`
	Name    string `msg:"name"`
}

func (f *fooV2) GetVersion() string {
	return "v2"
}

func (f *fooV2) InitVersion() {
	f.Version = "v2"
}

func (f *fooV2) TypeName() string {
	return "Foo"
}

func (f *fooV2) GetBase() EntityBaseI {
	return &fooBase{ID: f.ID}
}

func (f *fooV2) MigrateFrom(e EntityI) error {
	v1, ok := e.(*foo)
	if !ok {
		return errors.New("must migrate from prior version")
	}

	f.Version = "v2"
	f.ID = v1.ID
	return nil
}

type fooV3 struct {
	Version string `msg:"version"`
	ID      string `msg:"id"`
	Age     int    `msg:"age"`
}

func (f *fooV3) GetVersion() string {
	return "v3"
}

func (f *fooV3) InitVersion() {
	f.Version = "v3"
}

func (f *fooV3) TypeName() string {
	return "Foo"
}

func (f *fooV3) GetBase() EntityBaseI {
	return &fooBase{ID: f.ID}
}

func (f *fooV3) MigrateFrom(e EntityI) error {
	v2, ok := e.(*fooV2)
	if !ok {
		return errors.New("must migrate from prior version")
	}

	f.Version = "v3"
	f.ID = v2.ID
	return nil
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

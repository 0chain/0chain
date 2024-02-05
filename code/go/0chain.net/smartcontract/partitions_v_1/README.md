# Partitions

## Add new partition type

A new partition type must meet the `PartitionItem` interface:

```go
type PartitionItem interface {
	util.MPTSerializableSize
	GetID() string
}
```

To implement the `PartitionItem` interface is simple, what we need to do is add the `GetID()`
method manually and use `msgp` to generate methods for `util.MPTSerializableSize` interface.

Fo example:

```go
//go:generate msgp -io=false -tests=false

type Foo struct {
	ID string
	Name string
	Addr string
	Age int
}

func (f *Foo) GetID() string {
	return f.ID
}
```

Then run `go generate`, and we are done for the new partition type.

## Create a new partitions and add items to it

```go
// create a foo partitions with partition size of 50 and insert 100 items to it
//
// state is the state.StateContextI
parts, err := partitions.CreateIfNotExists(state, "foo_partitions", 50)
if err != nil {
	return err
}

// Add 100 items to the partitions
for i := 0; i < 100; i++ {
	err := parts.Add(state, &Foo{ID: fmt.Sprintf("f%v",i)})
	if err != nil {
		return err
	}

	if err := parts.Save(state); err != nil {
		return err
	}
}
```

## Get random items from partitions
```go
// get the partitions
parts, err := partitions.GetPartitions(state, "foo_partition")
if err != nil {
	return err
}

var foos []Foo
if err := parts.GetRandomItems(state, rand, &foos); err != nil {
	return err
}

```

## Get item of by id from partition
```go
parts, err := partitions.GetPartitions(state, "foo_partition")
if err != nil {
	return err
}

var foo Foo
if err := parts.Get(state, "f1", &foo); err != nil {
	return err
}

```


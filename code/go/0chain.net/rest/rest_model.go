package rest

//go:generate msgp -io=false -tests=false -unexported=true -v

// swagger:model int64Map
type int64Map map[string]int64

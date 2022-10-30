package common

/*ContextKey - type for key used to store values into context */
type ContextKey string

//go:generate msgp -io=false -tests=false -v
type FileOp int

const (
	Upload FileOp = iota
	Update
	Rename
	Copy
	Move
	Delete
	NewDir
)

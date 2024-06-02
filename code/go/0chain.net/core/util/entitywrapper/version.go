package entitywrapper

//go:generate msgp -v -tests=false -io=false -unexported
type entityVersion struct {
	Version string `msg:"version" json:"version"`
}

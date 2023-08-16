package event

type GooseDbVersion struct {
	ID        int
	VersionId int64
	IsApplied bool
	Tstamp    int64
}

type Tabler interface {
	TableName() string
}

// TableName overrides the table name used by GooseDbVersion to `profiles`
func (GooseDbVersion) TableName() string {
	return "goose_db_version"
}

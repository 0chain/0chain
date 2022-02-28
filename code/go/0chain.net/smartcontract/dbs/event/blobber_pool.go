package event

import (
	"github.com/guregu/null"
	"gorm.io/gorm"
)

type BlobberPool struct {
	gorm.Model
	ReadAllocationPoolID  null.String
	WriteAllocationPoolID null.String
	BlobberID             string
	Balance               int64
}

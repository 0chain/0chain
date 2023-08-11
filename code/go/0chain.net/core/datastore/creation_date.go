package datastore

import (
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
	"time"

	"0chain.net/core/common"
)

/*CreationTrackable - an interface that supports tracking the creation time */
type CreationTrackable interface {
	GetCreationTime() common.Timestamp
}

//go:generate msgp -io=false -tests=false -v
/*CreationDateField - Can be used to add a creation date functionality to an entity */
type CreationDateField struct {
	CreationDate common.Timestamp `json:"creation_date"`
}

/*InitializeCreationDate sets the creation date to current time */
func (cd *CreationDateField) InitializeCreationDate() {

	logging.Logger.Info("Jayash InitializeCreationDate",
		zap.Any("TIME NOW", time.Now()),
		zap.Any("COMMON NOW", common.Now()),
	)

	cd.CreationDate = common.Now()
}

/*GetCreationTime - Get the creation time */
func (cd *CreationDateField) GetCreationTime() common.Timestamp {
	return cd.CreationDate
}

/*ToTime - convert the common.Timestamp to time.Time */
func (cd *CreationDateField) ToTime() time.Time {
	logging.Logger.Info("Jayash TOTime",
		zap.Any("cd.CreationDate", cd.CreationDate),
		zap.Any("int64(cd.CreationDate)", int64(cd.CreationDate)),
		zap.Any("time.Unix(int64(cd.CreationDate), 0)", time.Unix(int64(cd.CreationDate), 0)),
	)
	return time.Unix(int64(cd.CreationDate), 0)
}

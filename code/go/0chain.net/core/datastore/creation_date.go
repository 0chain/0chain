package datastore

import (
	"github.com/0chain/common/core/logging"
	"github.com/beevik/ntp"
	"go.uber.org/zap"
	"log"
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
	// Specify the NTP server you want to query.
	ntpServer := "pool.ntp.org"

	// Get the current time from the NTP server.
	ntpTime, err := ntp.Time(ntpServer)
	if err != nil {
		log.Fatalf("Error fetching NTP time: %v", err)
	}

	logNow := time.Now()

	logging.Logger.Info("Jayash InitializeCreationDate",
		zap.Any("now", logNow),
		zap.Any("ntpTime", ntpTime))

	cd.CreationDate = common.Now()
}

/*GetCreationTime - Get the creation time */
func (cd *CreationDateField) GetCreationTime() common.Timestamp {
	return cd.CreationDate
}

/*ToTime - convert the common.Timestamp to time.Time */
func (cd *CreationDateField) ToTime() time.Time {
	return time.Unix(int64(cd.CreationDate), 0)
}

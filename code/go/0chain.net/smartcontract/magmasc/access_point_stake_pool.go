package magmasc

import (
	"strconv"

	"github.com/0chain/gosdk/zmagmacore/errors"
	zmc "github.com/0chain/gosdk/zmagmacore/magmasc"

	"0chain.net/core/viper"
)

type (
	// accessPointStakeReq represents access point's stake pool implementation.
	accessPointStakeReq struct {
		AccessPoint *zmc.AccessPoint `json:"access_point"`
		MinStake    int64            `json:"min_stake"`
	}
)

var (
	// Make sure tokenPoolRequest implements PoolConfigurator interface.
	_ zmc.PoolConfigurator = (*accessPointStakeReq)(nil)
)

// newAccessPointStakeReq returns a new constructed access point's stake pool.
func newAccessPointStakeReq(ap *zmc.AccessPoint, cfg *viper.Viper) *accessPointStakeReq {
	minStake := int64(cfg.GetFloat64(accessPointMinStake) * billion)
	if minStake < 0 {
		minStake = 0
	}

	return &accessPointStakeReq{
		AccessPoint: ap,
		MinStake:    minStake,
	}
}

// PoolBalance implements PoolConfigurator interface.
func (m *accessPointStakeReq) PoolBalance() int64 {
	return m.AccessPoint.MinStake
}

// PoolID implements PoolConfigurator interface.
func (m *accessPointStakeReq) PoolID() string {
	return m.AccessPoint.ID
}

// PoolHolderID implements PoolConfigurator interface.
func (m *accessPointStakeReq) PoolHolderID() string {
	return Address
}

// PoolPayerID implements PoolConfigurator interface.
func (m *accessPointStakeReq) PoolPayerID() string {
	return m.AccessPoint.ID
}

// PoolPayeeID implements PoolConfigurator interface.
func (m *accessPointStakeReq) PoolPayeeID() string {
	return m.AccessPoint.ID
}

// Validate checks accessPointStakeReq for correctness.
func (m *accessPointStakeReq) Validate() (err error) {
	switch { // is invalid
	case m.AccessPoint == nil:
		err = errors.New(errCodeInternal, "access point is required")

	case m.AccessPoint.ID == "":
		err = errors.New(errCodeBadRequest, "provider id is required")

	case m.AccessPoint.MinStake < m.MinStake:
		err = errors.New(errCodeInternal, "min stake value must be no less than: "+strconv.Itoa(int(m.MinStake)))
	}

	return err
}

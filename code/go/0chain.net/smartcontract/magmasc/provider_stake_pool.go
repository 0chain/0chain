package magmasc

import (
	"github.com/0chain/gosdk/zmagmacore/errors"
	zmc "github.com/0chain/gosdk/zmagmacore/magmasc"

	"0chain.net/core/viper"
)

type (
	// providerStakeReq represents provider's stake pool implementation.
	providerStakeReq struct {
		Provider *zmc.Provider `json:"provider"`
		MinStake int64         `json:"min_stake"`
	}
)

var (
	// Make sure tokenPoolRequest implements PoolConfigurator interface.
	_ zmc.PoolConfigurator = (*providerStakeReq)(nil)
)

// newProviderStakeReq returns a new constructed provider's stake pool.
func newProviderStakeReq(provider *zmc.Provider, cfg *viper.Viper) *providerStakeReq {
	minStake := int64(cfg.GetFloat64(providerMinStake) * zmc.Billion)
	if minStake < 0 {
		minStake = 0
	}

	return &providerStakeReq{
		Provider: provider,
		MinStake: minStake,
	}
}

// PoolBalance implements PoolConfigurator interface.
func (m *providerStakeReq) PoolBalance() int64 {
	return m.MinStake
}

// PoolID implements PoolConfigurator interface.
func (m *providerStakeReq) PoolID() string {
	return m.Provider.Id
}

// PoolHolderID implements PoolConfigurator interface.
func (m *providerStakeReq) PoolHolderID() string {
	return zmc.Address
}

// PoolPayerID implements PoolConfigurator interface.
func (m *providerStakeReq) PoolPayerID() string {
	return m.Provider.Id
}

// PoolPayeeID implements PoolConfigurator interface.
func (m *providerStakeReq) PoolPayeeID() string {
	return m.Provider.Id
}

// Validate checks providerStakeReq for correctness.
func (m *providerStakeReq) Validate() (err error) {
	switch { // is invalid
	case m.Provider == nil:
		err = errors.New(zmc.ErrCodeInternal, "provider is required")

	case m.Provider.Id == "":
		err = errors.New(zmc.ErrCodeBadRequest, "provider id is required")

	case m.Provider.ExtId == "":
		err = errors.New(zmc.ErrCodeBadRequest, "provider external id is required")
	}

	return err
}

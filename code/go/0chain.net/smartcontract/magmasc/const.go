package magmasc

import (
	zmc "github.com/0chain/gosdk/zmagmacore/magmasc"
)

const (
	// Address is a SHA3-256 hex encoded hash of "magma" string.
	// Represents address of MagmaSmartContract.
	Address = zmc.Address

	// Name contents the smart contract name.
	Name = "magma"

	// one billion (Giga) is a unit prefix in metric systems
	// of units denoting a factor of one billion (1e9 or 1_000_000_000).
	billion = 1e9

	// colon represents values separator.
	colon = ":"

	// providerMinStake represents the key of a provider min stake config.
	providerMinStake = "provider.min_stake"

	// serviceCharge represents the key of a service charge config.
	serviceCharge = "service_charge"

	// rootPath describes the magma smart contract's root path.
	rootPath = ".0chain.net"

	// storeName describes the magma smart contract's store name.
	storeName = "magmadb"

	// storePath describes the magma smart contract's store path.
	storePath = "data/rocksdb/magmasc"
)

const (
	// session contents a value of session string type.
	session = "session"

	// allRewardPoolsKey is a concatenated Address
	// and SHA3-256 hex encoded hash of "all_reward_pools" string.
	allRewardPoolsKey = Address + "59864241d642b4b6b5e5998b70bd201ca4d48926de8934e02e300950c778c7c2"

	// rewardPoolLock represents the name of MagmaSmartContract function.
	// When function is called it means that wallet creates a new locked token pool.
	rewardPoolLock = "reward_pool_lock"

	// rewardPoolUnlock represents the name of MagmaSmartContract function.
	// When function is called it means that wallet refunds a locked token pool.
	rewardPoolUnlock = "reward_pool_unlock"

	// rewardTokenPool contents a value of reward token pool string type.
	rewardTokenPool = "reward_token_pool"
)

// These constants used to identify smart contract functions by Consumer.
const (
	// AllConsumersKey is a concatenated Address
	// and SHA3-256 hex encoded hash of "all_consumers" string.
	AllConsumersKey = Address + "226fe0dc53026203416c348f675ce0c5ea35d87d959e41aaf6a3ca7829741710"

	// consumerType contents a value of type of Consumer's node.
	consumerType = "consumer"

	// consumerRegister represents name for Consumer's registration MagmaSmartContract function.
	consumerRegister = "consumer_register"

	// consumerSessionStart represents the name of MagmaSmartContract function.
	// When function is called it means that Consumer starts a new session.
	consumerSessionStart = "consumer_session_start"

	// consumerSessionStop represents the name of MagmaSmartContract function.
	// When function is called it means that Consumer stops an active session.
	consumerSessionStop = "consumer_session_stop"

	// consumerUpdate represents name for
	// consumer data update MagmaSmartContract function.
	consumerUpdate = "consumer_update"
)

// These constants used to identify smart contract functions by Provider.
const (
	// AllProvidersKey is a concatenated Address
	// and SHA3-256 hex encoded hash of "all_providers" string.
	AllProvidersKey = Address + "7e306c02ea1719b598aaf9dc7516eb930cd47c5360d974e22ab01e21d66a93d8"

	// providerType contents a value of type of Provider's node.
	providerType = "provider"

	// providerDataUsage represents name for
	// Provider's data usage billing MagmaSmartContract function.
	providerDataUsage = "provider_data_usage"

	// providerRegister represents name for
	// Provider's registration MagmaSmartContract function.
	providerRegister = "provider_register"

	// providerSessionInit represents name for
	// Provider's session init MagmaSmartContract function.
	providerSessionInit = "provider_session_init"

	// providerUpdate represents name for
	// provider data update MagmaSmartContract function.
	providerUpdate = "provider_update"

	// providerStakeTokenPool contents a value of provider's stake token pool string type.
	providerStakeTokenPool = "provider_stake_token_pool"
)

const (
	// accessPointRegister represents name for
	// Access Point's registration MagmaSmartContract function.
	accessPointRegister = "access_point_register"

	// accessPointUpdate represents name for
	// access point data update MagmaSmartContract function.
	accessPointUpdate = "access_point_update"

	// accessPointMinStake represents the key of a access point min stake config.
	accessPointMinStake = "access_point.min_stake"

	// accessPointStakeTokenPool contents a value of access point's stake token pool string type.
	accessPointStakeTokenPool = "access_point_stake_token_pool"
)

const (
	// AllAccessPointsKey is a concatenated Address
	// and SHA3-256 hex encoded hash of "all_access_points" string.
	AllAccessPointsKey = Address + "b0473d07c62a69f3d03165d3afc670045b8471309102e169fc2e990bd065e74c"

	// accessPointType contents a value of type of Access Point's node.
	accessPointType = "access-point"
)

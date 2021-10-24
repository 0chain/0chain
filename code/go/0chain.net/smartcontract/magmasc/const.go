package magmasc

import (
	zmc "github.com/0chain/gosdk/zmagmacore/magmasc"
)

const (
	// Name contents the smart contract name.
	Name = "magma"

	// colon represents values separator.
	colon = ":"

	// providerMinStake represents the key of a provider min stake config.
	providerMinStake = "provider.min_stake"

	// accessPointMinStake represents the key of a access point min stake config.
	accessPointMinStake = "access_point.min_stake"

	// serviceCharge represents the key of a service charge config.
	serviceCharge = "service_charge"

	// billingRatio represents the key of a billing ratio config.
	billingRatio = "billing.ratio"

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
	allRewardPoolsKey = zmc.Address + "59864241d642b4b6b5e5998b70bd201ca4d48926de8934e02e300950c778c7c2"

	// rewardTokenPool contents a value of reward token pool string type.
	rewardTokenPool = "reward_token_pool"
)

const (
	// allConsumersKey is a concatenated Address
	// and SHA3-256 hex encoded hash of "all_consumers" string.
	allConsumersKey = zmc.Address + "226fe0dc53026203416c348f675ce0c5ea35d87d959e41aaf6a3ca7829741710"

	// consumerType contents a value of type of Consumer's node.
	consumerType = "consumer"
)

const (
	// allProvidersKey is a concatenated Address
	// and SHA3-256 hex encoded hash of "all_providers" string.
	allProvidersKey = zmc.Address + "7e306c02ea1719b598aaf9dc7516eb930cd47c5360d974e22ab01e21d66a93d8"

	// providerStake contents a value of provider's stake token pool string type.
	providerStake = "provider_stake"

	// providerType contents a value of type of Provider's node.
	providerType = "provider"
)

const (
	// allAccessPointsKey is a concatenated Address
	// and SHA3-256 hex encoded hash of "all_access_points" string.
	allAccessPointsKey = zmc.Address + "b0473d07c62a69f3d03165d3afc670045b8471309102e169fc2e990bd065e74c"

	// accessPointType contents a value of type of Access Point's node.
	accessPointType = "access_point"

	// accessPointStake contents a value of access point's stake token pool string type.
	accessPointStake = "access_point_stake"
)

const (
	// allUsersKey is a concatenated Address
	// and SHA3-256 hex encoded hash of "all_users" string.
	allUsersKey = zmc.Address + "c076883a6a9d262d0f3405b07fb2f02a57a35f22679db452f1bc6fc509068c90"

	// userType contents a value of type of User's.
	userType = "user"
)

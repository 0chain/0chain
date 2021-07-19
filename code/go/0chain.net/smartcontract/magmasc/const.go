package magmasc

const (
	// Address is a SHA3-256 hex encoded hash of "magma" string.
	// Represents address of MagmaSmartContract.
	Address = "11f8411db41e34cea7c100f19faff32da8f3cd5a80635731cec06f32d08089be"

	// Name contents the smart contract name.
	Name = "magma"

	// colon represents values separator.
	colon = ":"
)

const (
	// acknowledgment contents a value of acknowledgment string type.
	acknowledgment = "acknowledgment"

	// datausage contents a value of datausage string type.
	datausage = "datausage"
)

// These constants used to identify smart contract functions by Consumer.
const (
	// AllConsumersKey is a concatenated Address
	// and SHA3-256 hex encoded hash of "all_consumers" string.
	AllConsumersKey = Address + "226fe0dc53026203416c348f675ce0c5ea35d87d959e41aaf6a3ca7829741710"

	// consumerType contents a value of type of Consumer's node.
	consumerType = "consumer"

	// consumerAcceptTerms represents the name of MagmaSmartContract function.
	// When function is called it means that Consumer accepted Provider terms.
	consumerAcceptTerms = "consumer_accept_terms"

	// consumerRegister represents name for Consumer's registration MagmaSmartContract function.
	consumerRegister = "consumer_register"

	// consumerSessionStop represents the name of MagmaSmartContract function.
	// When function is called it means that Consumer stops the session.
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

	// providerUpdate represents name for
	// provider data update MagmaSmartContract function.
	providerUpdate = "provider_update"
)

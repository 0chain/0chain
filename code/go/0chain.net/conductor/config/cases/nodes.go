package cases

type (
	// Nodes represents struct for containing miners and sharders info.
	Nodes struct {
		Miners   Miners   `json:"miners" yaml:"miners" mapstructure:"miners"`
		Sharders Sharders `json:"sharders" yaml:"sharders" mapstructure:"sharders"`
	}

	// Miners represents list of Miner.
	Miners []*Miner

	// Miner represents struct for containing miner's info.
	//
	// 	Explaining definitions example:
	//		Generators num = 2
	// 		len(miners) = 4
	//
	// 		Generator0:	rank = 0; TypeRank = 0; Generator = true.
	// 		Generator1:	rank = 1; TypeRank = 1; Generator = true.
	// 		Replica0:	rank = 2; TypeRank = 0; Generator = false.
	// 		Replica0:	rank = 3; TypeRank = 1; Generator = false.
	Miner struct {
		Generator bool `json:"generator" yaml:"generator" mapstructure:"generator"`

		TypeRank int `json:"type_rank" yaml:"type_rank" mapstructure:"type_rank"`
	}

	// Sharders represents list of Sharder.
	Sharders []Sharder

	// Sharder represents string that explains sharder type.
	//
	// Example: "sharder-1".
	Sharder string
)

// Num returns number of all nodes contained by Nodes.
func (n *Nodes) Num() int {
	return len(n.Miners) + len(n.Sharders)
}

// Get looks for Miner with provided Miner.Generator and Miner.TypeRank and returns it if founds.
func (m Miners) Get(generator bool, typeRank int) *Miner {
	for _, miner := range m {
		if miner.Generator == generator && miner.TypeRank == typeRank {
			return miner
		}
	}
	return nil
}

// Contains looks for Sharder with provided name.
func (s Sharders) Contains(name string) bool {
	for _, sharder := range s {
		if string(sharder) == name {
			return true
		}
	}
	return false
}

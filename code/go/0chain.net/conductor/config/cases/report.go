package cases

type (
	TestReport struct {
		ByGenerator        bool  `json:"by_generator" yaml:"by_generator" mapstructure:"by_generator"`
		ByNodeWithTypeRank int   `json:"by_node_with_type_rank" yaml:"by_node_with_type_rank" mapstructure:"by_node_with_type_rank"`
		OnRound            int64 `json:"round" yaml:"round" mapstructure:"round"`
	}
)

// IsTesting implements TestCaseConfigurator interface.
func (b *TestReport) IsTesting(round int64, generator bool, nodeTypeRank int) bool {
	return b.OnRound == round && b.ByGenerator == generator && nodeTypeRank == b.ByNodeWithTypeRank
}

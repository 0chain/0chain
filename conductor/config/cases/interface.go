package cases

import (
	"0chain.net/conductor/cases"
)

type (
	// TestCaseConfigurator represents interface for configuring test cases.
	TestCaseConfigurator interface {
		TestCase() cases.TestCase
		Name() string

		MapDecoder
	}

	MapDecoder interface {
		Decode(val interface{}) error
	}

	// TestReporter represents interface for test case configuration.
	TestReporter interface {
		IsTesting(round int64, generator bool, nodeTypeRank int) bool
		IsOnRound(round int64) bool
	}
)

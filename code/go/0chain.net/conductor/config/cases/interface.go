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
)

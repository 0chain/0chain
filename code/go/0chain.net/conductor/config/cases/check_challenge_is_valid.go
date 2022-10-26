package cases

import (
	"github.com/mitchellh/mapstructure"

	"0chain.net/conductor/cases"
)

type (
	// CheckChallengeIsValid represents TestCaseConfigurator implementation.
	CheckChallengeIsValid struct {
	}
)

const (
	CheckChallengeIsValidName = "check challenge is valid"
)

var (
	// Ensure CheckChallengeIsValid implements TestCaseConfigurator.
	_ TestCaseConfigurator = (*CheckChallengeIsValid)(nil)
)

// NewCheckChallengeIsValid creates initialised CheckChallengeIsValid.
func NewCheckChallengeIsValid() *CheckChallengeIsValid {
	return &CheckChallengeIsValid{}
}

// TestCase implements TestCaseConfigurator interface.
func (n *CheckChallengeIsValid) TestCase() cases.TestCase {
	return cases.NewCheckChallengeIsValid()
}

// Name implements TestCaseConfigurator interface.
func (n *CheckChallengeIsValid) Name() string {
	return CheckChallengeIsValidName
}

// Decode implements MapDecoder interface.
func (n *CheckChallengeIsValid) Decode(val interface{}) error {
	return mapstructure.Decode(val, n)
}

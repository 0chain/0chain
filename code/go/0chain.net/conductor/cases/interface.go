package cases

import (
	"context"
)

type (
	// TestCase represents interfaces to perform checks.
	TestCase interface {
		// Configure takes encoded input that can be any type, and sets any needed preconditions for the TestCase.
		Configure([]byte) error

		// AddResult adds result that is needed to be checked.
		AddResult([]byte) error

		// Check runs main test case.
		Check(ctx context.Context) (success bool, err error)
	}
)

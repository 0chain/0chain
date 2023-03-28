//go:build !integration_tests
// +build !integration_tests

// todo: it's a legacy ugly approach; refactor later

package storagesc

func afterInsertBlobber(id string)                                 {}
func afterAddChallenge(challengeID string, validatorsIDs []string) {}
func beforeEmitAddChallenge(challenge *StorageChallengeResponse)   {}

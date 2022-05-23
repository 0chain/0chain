//go:build !integration_tests
// +build !integration_tests

package main

// stubs that does nothing
func initIntegrationsTests(id string)  {}
func shutdownIntegrationTests()        {}
func configureIntegrationsTestsFlags() {}

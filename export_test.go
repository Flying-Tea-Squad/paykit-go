package paykit

// ResetRegistryForTesting clears the global registry for test isolation.
// It is only compiled during tests and exposed to external test packages.
func ResetRegistryForTesting() {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry = make(map[string]Factory)
}

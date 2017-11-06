package maintenance

// Start starts the maintenance process.
func Start(globalConfigurationExisted bool) {
	CheckForUpdates(!globalConfigurationExisted)
}

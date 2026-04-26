package providers

var defaultMetricRepo = newMetricRepo()

func canonicalObservabilityMetricName(storageName string) string {
	return defaultMetricRepo.SemanticName(storageName)
}

func storageObservabilityMetricName(semanticOrStorageName string) string {
	return defaultMetricRepo.StorageName(semanticOrStorageName)
}

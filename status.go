package menmos

func isStatusSuccess(statusCode int) bool {
	// Not mega-robust, but good enough for our use-case.
	return statusCode >= 200 && statusCode < 300
}

func isTemporaryRedirect(statusCode int) bool {
	return statusCode == 307
}

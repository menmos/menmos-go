package menmos

func isStatusSuccess(statusCode int) bool {
	// Not mega-robust, but good enough for our use-case.
	return statusCode < 300 && statusCode >= 200
}

func isTemporaryRedirect(statusCode int) bool {
	return statusCode == 307
}

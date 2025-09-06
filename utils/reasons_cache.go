package utils

// Global cache for rejection reasons
var (
	rejectionReasonCache           = make(map[string][]string)
	availableRejectionReasonsCache = make(map[string][]string) // To store all reasons before selection
	banReasonCache                 = make(map[string][]string)
	availableBanReasonsCache       = make(map[string][]string)
)

// SetRejectionReasons caches the selected rejection reasons for a submission.
func SetRejectionReasons(submissionID string, reasons []string) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	rejectionReasonCache[submissionID] = reasons
}

// GetRejectionReasons retrieves the cached rejection reasons for a submission.
func GetRejectionReasons(submissionID string) ([]string, bool) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	reasons, ok := rejectionReasonCache[submissionID]
	return reasons, ok
}

// DeleteRejectionReasons removes the cached rejection reasons for a submission.
func DeleteRejectionReasons(submissionID string) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	delete(rejectionReasonCache, submissionID)
}

// SetAvailableRejectionReasons caches all available rejection reasons for a submission.
func SetAvailableRejectionReasons(submissionID string, reasons []string) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	availableRejectionReasonsCache[submissionID] = reasons
}

// GetAvailableRejectionReasons retrieves all cached available rejection reasons for a submission.
func GetAvailableRejectionReasons(submissionID string) ([]string, bool) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	reasons, ok := availableRejectionReasonsCache[submissionID]
	return reasons, ok
}

// DeleteAvailableRejectionReasons removes the cached available rejection reasons for a submission.
func DeleteAvailableRejectionReasons(submissionID string) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	delete(availableRejectionReasonsCache, submissionID)
}

// SetBanReasons caches the selected ban reasons for a submission.
func SetBanReasons(submissionID string, reasons []string) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	banReasonCache[submissionID] = reasons
}

// GetBanReasons retrieves the cached ban reasons for a submission.
func GetBanReasons(submissionID string) ([]string, bool) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	reasons, ok := banReasonCache[submissionID]
	return reasons, ok
}

// DeleteBanReasons removes the cached ban reasons for a submission.
func DeleteBanReasons(submissionID string) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	delete(banReasonCache, submissionID)
}

// SetAvailableBanReasons caches all available ban reasons for a submission.
func SetAvailableBanReasons(submissionID string, reasons []string) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	availableBanReasonsCache[submissionID] = reasons
}

// GetAvailableBanReasons retrieves all cached available ban reasons for a submission.
func GetAvailableBanReasons(submissionID string) ([]string, bool) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	reasons, ok := availableBanReasonsCache[submissionID]
	return reasons, ok
}

// DeleteAvailableBanReasons removes the cached available ban reasons for a submission.
func DeleteAvailableBanReasons(submissionID string) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	delete(availableBanReasonsCache, submissionID)
}

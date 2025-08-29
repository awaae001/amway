package model

import (
	"sync"
	"time"
)

// SubmissionData holds the temporary data for a submission.
type SubmissionData struct {
	ChannelID        string
	MessageID        string
	OriginalAuthor   string
	RecommendTitle   string
	RecommendContent string
	ReplyToOriginal  bool
	SubmissionID     string // Added to track the database submission ID
	CreatedAt        time.Time
}

// Global cache for rejection reasons
var (
	rejectionReasonCache           = make(map[string][]string)
	availableRejectionReasonsCache = make(map[string][]string) // To store all reasons before selection
	cacheMutex                     = &sync.Mutex{}
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

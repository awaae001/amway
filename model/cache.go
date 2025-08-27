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
	EphChannelID     string
	EphMessageID     string
	ReplyToOriginal  bool
	CreatedAt        time.Time
}

// Global cache for rejection reasons
var (
	rejectionReasonCache = make(map[string][]string)
	cacheMutex           = &sync.Mutex{}
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

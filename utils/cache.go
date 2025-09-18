package utils

import (
	"amway/db"
	"amway/model"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	submissionCache     = make(map[string]model.SubmissionData)
	submissionRateLimit = make(map[string]time.Time) // userID -> last submission time
	cacheMutex          = &sync.RWMutex{}
	rateLimitMutex      = &sync.RWMutex{}
	cacheTTL            = 24 * time.Hour // Cache entries expire after 24 hours
	rateLimitTTL        = 3 * time.Hour  // Rate limit entries expire after 3 hours
)

func init() {
	go startCacheJanitor()
	go startRateLimitJanitor()
}

// AddToCache adds submission data to the cache and returns a unique ID.
func AddToCache(data model.SubmissionData) string {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	id := uuid.New().String()
	data.CreatedAt = time.Now()
	submissionCache[id] = data
	return id
}

// GetFromCache retrieves submission data from the cache by ID.
func GetFromCache(id string) (model.SubmissionData, bool) {
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()

	data, found := submissionCache[id]
	return data, found
}

// UpdateCache updates an existing cache entry.
func UpdateCache(id string, data model.SubmissionData) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	// It's important to preserve the original creation time
	if oldData, ok := submissionCache[id]; ok {
		data.CreatedAt = oldData.CreatedAt
		submissionCache[id] = data
	}
}

// RemoveFromCache removes submission data from the cache by ID.
func RemoveFromCache(id string) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	delete(submissionCache, id)
}

// startCacheJanitor runs a background process to clean up expired cache entries
// and automatically reject submissions that haven't been reviewed within 24 hours.
func startCacheJanitor() {
	ticker := time.NewTicker(1 * time.Hour) // Check every hour
	defer ticker.Stop()

	for range ticker.C {
		processExpiredSubmissions()
	}
}

// processExpiredSubmissions handles expired cache entries and auto-rejection
func processExpiredSubmissions() {
	cacheMutex.Lock()
	var expiredEntries []struct {
		cacheID string
		data    model.SubmissionData
	}

	// First, collect all expired entries
	for id, data := range submissionCache {
		if time.Since(data.CreatedAt) > cacheTTL {
			expiredEntries = append(expiredEntries, struct {
				cacheID string
				data    model.SubmissionData
			}{cacheID: id, data: data})
		}
	}
	cacheMutex.Unlock()

	// Process each expired entry
	for _, entry := range expiredEntries {
		handleExpiredSubmission(entry.cacheID, entry.data)
	}
}

// handleExpiredSubmission processes a single expired submission
func handleExpiredSubmission(cacheID string, data model.SubmissionData) {
	log.Printf("Processing expired submission cache: %s, submission ID: %s", cacheID, data.SubmissionID)
	
	if data.SubmissionID == "" {
		// No submission ID stored, just remove from cache
		log.Printf("No submission ID in expired cache entry %s, skipping auto-rejection", cacheID)
		removeFromCacheSafely(cacheID)
		return
	}

	// Get the submission from database to check its current status
	submission, err := db.GetSubmission(data.SubmissionID)
	if err != nil {
		log.Printf("Error getting submission %s for auto-rejection: %v", data.SubmissionID, err)
		removeFromCacheSafely(cacheID)
		return
	}

	if submission == nil {
		// Submission not found, already processed or deleted
		log.Printf("Submission %s not found, removing from cache", data.SubmissionID)
		removeFromCacheSafely(cacheID)
		return
	}

	// Only auto-reject if still pending
	if submission.Status == "pending" {
		log.Printf("Auto-rejecting expired submission %s after 24 hours", data.SubmissionID)
		autoRejectSubmission(submission)
	} else {
		log.Printf("Submission %s already processed (status: %s), removing from cache", data.SubmissionID, submission.Status)
	}
	
	// Remove from cache after processing
	removeFromCacheSafely(cacheID)
}

// autoRejectSubmission automatically rejects a submission with default reason
func autoRejectSubmission(submission *model.Submission) {
	const autoRejectReason = "你的安利存在一些问题，请修改后再次投稿吧"
	
	// Update submission status to rejected
	err := db.UpdateSubmissionReviewer(submission.ID, "rejected", "system")
	if err != nil {
		log.Printf("Error updating submission %s status to rejected: %v", submission.ID, err)
		return
	}

	// Update user stats - increment rejected count
	db.IncrementRejectedCount(submission.UserID)
	
	log.Printf("Successfully auto-rejected submission %s for user %s", submission.ID, submission.UserID)
	
	// Send rejection notification to user
	SendAutoRejectionDM(submission, autoRejectReason)
}

// removeFromCacheSafely removes an entry from cache with proper locking
func removeFromCacheSafely(cacheID string) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	delete(submissionCache, cacheID)
}

// CheckSubmissionRateLimit checks if a user can submit based on rate limiting.
// Returns (canSubmit bool, remainingTime time.Duration)
func CheckSubmissionRateLimit(userID string) (bool, time.Duration) {
	rateLimitMutex.RLock()
	defer rateLimitMutex.RUnlock()

	lastSubmission, exists := submissionRateLimit[userID]
	if !exists {
		return true, 0
	}

	elapsed := time.Since(lastSubmission)
	if elapsed >= rateLimitTTL {
		return true, 0
	}

	remainingTime := rateLimitTTL - elapsed
	return false, remainingTime
}

// RecordSubmissionTime records the submission time for rate limiting
func RecordSubmissionTime(userID string) {
	rateLimitMutex.Lock()
	defer rateLimitMutex.Unlock()

	submissionRateLimit[userID] = time.Now()
}

// startRateLimitJanitor runs a background process to clean up expired rate limit entries
func startRateLimitJanitor() {
	ticker := time.NewTicker(30 * time.Minute) // Check every 30 minutes
	defer ticker.Stop()

	for range ticker.C {
		cleanExpiredRateLimit()
	}
}

// cleanExpiredRateLimit removes expired rate limit entries
func cleanExpiredRateLimit() {
	rateLimitMutex.Lock()
	defer rateLimitMutex.Unlock()

	now := time.Now()
	for userID, lastSubmission := range submissionRateLimit {
		if now.Sub(lastSubmission) > rateLimitTTL {
			delete(submissionRateLimit, userID)
		}
	}
}
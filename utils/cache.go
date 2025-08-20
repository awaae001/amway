package utils

import (
	"amway/model"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	submissionCache = make(map[string]model.SubmissionData)
	cacheMutex      = &sync.RWMutex{}
	cacheTTL        = 5 * time.Minute // Cache entries expire after 5 minutes
)

func init() {
	go startCacheJanitor()
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

// RemoveFromCache removes submission data from the cache by ID.
func RemoveFromCache(id string) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	delete(submissionCache, id)
}

// startCacheJanitor runs a background process to clean up expired cache entries.
func startCacheJanitor() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		cacheMutex.Lock()
		for id, data := range submissionCache {
			if time.Since(data.CreatedAt) > cacheTTL {
				delete(submissionCache, id)
			}
		}
		cacheMutex.Unlock()
	}
}

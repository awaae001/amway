package vote

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// VoteType represents the type of vote.
type VoteType string

const (
	// Pass represents a vote to approve a submission.
	Pass VoteType = "pass"
	// Reject represents a vote to reject a submission.
	Reject VoteType = "reject"
	// Ban represents a vote to ban the author of the submission.
	Ban VoteType = "ban"
	// Feature represents a vote to feature the submission.
	Feature VoteType = "feature"
)

// Vote represents a single vote cast by an admin.
type Vote struct {
	VoterID   string    `json:"voter_id"`
	Type      VoteType  `json:"type"`
	Reason    string    `json:"reason,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// Session represents a voting session for a single submission.
type Session struct {
	VoteFileID   string `json:"vote_file_id"`
	SubmissionID string `json:"submission_id"`
	Votes        []Vote `json:"votes"`
}

// AddVote adds a new vote to a session.
func (s *Session) AddVote(vote Vote) {
	// Check if the user has already voted
	for i, v := range s.Votes {
		if v.VoterID == vote.VoterID {
			s.Votes[i] = vote // Overwrite the existing vote
			return
		}
	}
	s.Votes = append(s.Votes, vote)
}

// RemoveVote removes a vote from a session by voterID. Returns true if a vote was removed.
func (s *Session) RemoveVote(voterID string) bool {
	originalVoteCount := len(s.Votes)
	var newVotes []Vote
	for _, v := range s.Votes {
		if v.VoterID != voterID {
			newVotes = append(newVotes, v)
		}
	}
	s.Votes = newVotes
	return len(s.Votes) < originalVoteCount
}

const voteDir = "data/votes"

// Manager handles all vote-related operations.
type Manager struct {
	mu   sync.Mutex
	path string
}

// NewManager creates a new vote manager.
func NewManager() (*Manager, error) {
	err := os.MkdirAll(voteDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("could not create vote directory: %w", err)
	}
	return &Manager{path: voteDir}, nil
}

// getVoteFilePath returns the path to the JSON file for a given submission ID.
func (m *Manager) getVoteFilePath(voteFileID string) string {
	return filepath.Join(m.path, fmt.Sprintf("vote-%s.json", voteFileID))
}

// LoadSession loads a voting session from a JSON file.
func (m *Manager) LoadSession(voteFileID string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	filePath := m.getVoteFilePath(voteFileID)
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// If the file doesn't exist, create a new session
			return &Session{VoteFileID: voteFileID, Votes: []Vote{}}, nil
		}
		return nil, err
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}
	return &session, nil
}

// SaveSession saves a voting session to a JSON file.
func (m *Manager) SaveSession(session *Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}

	filePath := m.getVoteFilePath(session.VoteFileID)
	return ioutil.WriteFile(filePath, data, 0644)
}

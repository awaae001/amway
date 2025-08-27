package tools

import "amway/vote"

// hasConsensus checks if there is a consensus in the vote counts.
func HasConsensus(voteCounts map[vote.VoteType]int) bool {
	for _, count := range voteCounts {
		if count >= 2 {
			return true
		}
	}
	return false
}

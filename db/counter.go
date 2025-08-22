package db

import "database/sql"

// getNextSubmissionID retrieves the current submission ID and increments it by one within a transaction.
func getNextSubmissionID(tx *sql.Tx) (int, error) {
	var currentID int
	err := tx.QueryRow("SELECT current_value FROM id_counter WHERE counter_name = 'submission_id'").Scan(&currentID)
	if err != nil {
		return 0, err
	}

	newID := currentID + 1
	_, err = tx.Exec("UPDATE id_counter SET current_value = ? WHERE counter_name = 'submission_id'", newID)
	if err != nil {
		return 0, err
	}

	return newID, nil
}

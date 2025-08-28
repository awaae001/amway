package db

import "database/sql"

// getNextSubmissionID 在事务中检索当前投稿 ID 并将其加一。
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

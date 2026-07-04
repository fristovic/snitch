package record

import (
	"encoding/json"
	"time"
)

// SaveTurnSnapshot persists the full run payload JSON and snapshot metadata.
func (s *Store) SaveTurnSnapshot(runID string, payloadJSON []byte, startHEAD, endHEAD string, fileManifest map[string]string) error {
	manifestJSON, _ := json.Marshal(fileManifest)
	_, err := s.db.Exec(`
		UPDATE runs SET payload_json=?, start_head=?, end_head=?, file_manifest_json=?
		WHERE id=?`,
		string(payloadJSON), startHEAD, endHEAD, string(manifestJSON), runID)
	return err
}

// GetRunPayloadJSON returns the stored capture payload JSON for a run.
func (s *Store) GetRunPayloadJSON(runID string) ([]byte, error) {
	var raw string
	err := s.db.QueryRow(`SELECT payload_json FROM runs WHERE id=?`, runID).Scan(&raw)
	if err != nil {
		return nil, err
	}
	if raw == "" {
		return nil, nil
	}
	return []byte(raw), nil
}

// GetPriorRunPayloadJSON returns up to limit prior run payloads in the same session before the given time.
// Results are ordered oldest-first.
func (s *Store) GetPriorRunPayloadJSON(sessionID string, before time.Time, limit int) ([][]byte, error) {
	if sessionID == "" || limit <= 0 {
		return nil, nil
	}
	beforeStr := before.UTC().Format(time.RFC3339)
	rows, err := s.db.Query(`
		SELECT payload_json FROM runs
		WHERE session_id=? AND created_at < ? AND payload_json IS NOT NULL AND payload_json != ''
		ORDER BY created_at DESC LIMIT ?`,
		sessionID, beforeStr, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out [][]byte
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		out = append(out, []byte(raw))
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, rows.Err()
}

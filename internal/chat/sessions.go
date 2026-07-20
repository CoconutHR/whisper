package chat

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"time"
)

func hashSessionToken(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}

// CreateSession stores only a digest of the browser token. The raw token stays in the cookie.
func (s *Store) CreateSession(token, userID string, expiresAt time.Time) error {
	_, err := s.db.Exec(`INSERT INTO sessions(token_hash, user_id, expires_at, created_at)
		VALUES (?, ?, ?, ?)`, hashSessionToken(token), userID,
		expiresAt.Format(time.RFC3339Nano), time.Now().Format(time.RFC3339Nano))
	return err
}

func (s *Store) SessionUser(token string, now time.Time) (string, bool, error) {
	var userID, expiresAt string
	err := s.db.QueryRow(`SELECT user_id, expires_at FROM sessions WHERE token_hash = ?`,
		hashSessionToken(token)).Scan(&userID, &expiresAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", false, nil
		}
		return "", false, err
	}
	expires, err := time.Parse(time.RFC3339Nano, expiresAt)
	if err != nil {
		return "", false, err
	}
	if !now.Before(expires) {
		if _, deleteErr := s.db.Exec(`DELETE FROM sessions WHERE token_hash = ?`, hashSessionToken(token)); deleteErr != nil {
			return "", false, deleteErr
		}
		return "", false, nil
	}
	return userID, true, nil
}

func (s *Store) DeleteSession(token string) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE token_hash = ?`, hashSessionToken(token))
	return err
}

func (s *Store) DeleteExpiredSessions(now time.Time) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE expires_at <= ?`, now.Format(time.RFC3339Nano))
	return err
}

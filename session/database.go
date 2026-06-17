package session

import (
	"database/sql"
	"encoding/json"
	"time"
)

func NewDatabaseStore(db *sql.DB) *DatabaseStore {
	return &DatabaseStore{db: db}
}

func (d *DatabaseStore) read(id string) (*Session, error) {
	var session Session
	var payload string
	var createdAt int64
	var lastActivityAt int64

	err := d.db.QueryRow("SELECT id, user_id, ip_address, user_agent, payload, created_at, last_activity FROM sessions WHERE id = $1", id).
		Scan(&session.Id, &session.UserId, &session.IpAddress, &session.UserAgent, &payload, &createdAt, &lastActivityAt)
	if err != nil {
		return nil, err
	}

	session.createdAt = time.UnixMilli(createdAt)
	session.lastActivityAt = time.UnixMilli(lastActivityAt)
	json.Unmarshal([]byte(payload), &session.payload)

	return &session, nil
}

func (d *DatabaseStore) write(session *Session) error {
	payload, err := json.Marshal(session.payload)
	if err != nil {
		return err
	}

	_, err = d.db.Exec(
		"INSERT INTO sessions (id, user_id, ip_address, user_agent, payload, created_at, last_activity) "+
			"VALUES ($1, $2, $3, $4, $5, $6, $7)"+
			"ON CONFLICT (id) DO UPDATE SET payload = $5, last_activity = $7",
		session.Id, session.UserId, session.IpAddress, session.UserAgent, payload, session.createdAt.UnixMilli(), session.lastActivityAt.UnixMilli(),
	)
	if err != nil {
		return err
	}

	return nil
}

func (d *DatabaseStore) destroy(id string) error {
	_, err := d.db.Exec("DELETE FROM sessions WHERE id = $1", id)
	if err != nil {
		return err
	}

	return nil
}

func (d *DatabaseStore) gc(idleExpiration, absoluteExpiration time.Duration) error {

	_, err := d.db.Exec("DELETE FROM sessions WHERE created_at < $1 OR last_activity_at < $2", time.Now().Add(-absoluteExpiration), time.Now().Add(-idleExpiration))
	if err != nil {
		return err
	}

	return nil
}

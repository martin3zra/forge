package session

import (
	"context"
	"encoding/json"
	"time"

	"github.com/martin3zra/playsql"
)

// sessionModel is the on-disk shape of a row in the sessions table. Kept
// separate from Session (the domain type) so playsql's struct-tag-driven
// metadata never has to know about Session's unexported fields.
//
// The creation-time column is named started_at, not created_at: playsql's
// Upsert auto-stamps any column named exactly "created_at"/"updated_at" with
// time.Now() on every call (see write_map.go), which would both clobber the
// Unix-millis int64 this code writes and reset the value on every touch of an
// existing session — created_at is meant to be set once, at creation.
type sessionModel struct {
	ID           string `db:"id"`
	UserID       *int64 `db:"user_id"`
	IPAddress    string `db:"ip_address"`
	UserAgent    string `db:"user_agent"`
	Payload      string `db:"payload"`
	StartedAt    int64  `db:"started_at"`
	LastActivity int64  `db:"last_activity"`
}

func (sessionModel) TableName() string { return "sessions" }

func NewDatabaseStore(db *playsql.DB) *DatabaseStore {
	return &DatabaseStore{db: db}
}

func (d *DatabaseStore) read(id string) (*Session, error) {
	var row sessionModel
	err := d.db.Model(&sessionModel{}).WhereEq("id", id).First(context.Background(), &row)
	if err != nil {
		return nil, err
	}

	session := &Session{
		Id:             row.ID,
		UserId:         row.UserID,
		IpAddress:      row.IPAddress,
		UserAgent:      row.UserAgent,
		createdAt:      time.UnixMilli(row.StartedAt),
		lastActivityAt: time.UnixMilli(row.LastActivity),
	}
	json.Unmarshal([]byte(row.Payload), &session.payload)

	return session, nil
}

func (d *DatabaseStore) write(session *Session) error {
	payload, err := json.Marshal(session.payload)
	if err != nil {
		return err
	}

	row := map[string]any{
		"id":            session.Id,
		"user_id":       session.UserId,
		"ip_address":    session.IpAddress,
		"user_agent":    session.UserAgent,
		"payload":       string(payload),
		"started_at":    session.createdAt.UnixMilli(),
		"last_activity": session.lastActivityAt.UnixMilli(),
	}

	_, err = d.db.Model(&sessionModel{}).Upsert(
		context.Background(),
		[]map[string]any{row},
		[]string{"id"},
		[]string{"payload", "last_activity"},
	)
	return err
}

func (d *DatabaseStore) destroy(id string) error {
	_, err := d.db.Model(&sessionModel{}).WhereEq("id", id).Delete(context.Background())
	return err
}

func (d *DatabaseStore) gc(idleExpiration, absoluteExpiration time.Duration) error {
	now := time.Now()
	_, err := d.db.Model(&sessionModel{}).
		Where("started_at", "<", now.Add(-absoluteExpiration).UnixMilli()).
		OrWhere("last_activity", "<", now.Add(-idleExpiration).UnixMilli()).
		Delete(context.Background())
	return err
}

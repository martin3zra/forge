package session

import (
	"database/sql"
	"net/http"
	"time"
)

type SessionContextKey struct{}

type Session struct {
	createdAt      time.Time
	lastActivityAt time.Time
	Id             string
	UserId         *int64
	IpAddress      string
	UserAgent      string
	payload        map[string]any
}

type SessionStore interface {
	// reads the session that has the given unique ID.
	read(id string) (*Session, error)
	// writes a session to the storage engine.
	write(session *Session) error
	// deletes a session with the given ID.
	destroy(id string) error
	// performs garbage collection. It queries all expired sessions and deletes them from storage.
	gc(idleExpiration, absoluteExpiration time.Duration) error
}

// type that we'll use to coordinate all session operations: reading, writing, and deleting sessions.
type SessionManager struct {
	store              SessionStore
	idleExpiration     time.Duration
	absoluteExpiration time.Duration
	cookieName         string
	domain             string
	secure             bool
	httpOnly           bool
}

type DatabaseStore struct {
	db *sql.DB
}

type sessionResponseWriter struct {
	http.ResponseWriter
	sessionManager *SessionManager
	request        *http.Request
	done           bool
}

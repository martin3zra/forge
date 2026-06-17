package session

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/martin3zra/forge/foundation"
	"github.com/romsar/gonertia/v2"
)

func NewSessionManager(
	store SessionStore,
	gcInterval,
	idleExpiration,
	absoluteExpiration time.Duration,
	cookieName string,
	domain string,
	secure bool,
	httpOnly bool,
) *SessionManager {
	manager := &SessionManager{
		store:              store,
		idleExpiration:     idleExpiration,
		absoluteExpiration: absoluteExpiration,
		cookieName:         cookieName,
		domain:             domain,
		secure:             secure,
		httpOnly:           httpOnly,
	}

	go manager.gc(gcInterval)

	return manager
}

// The gc method uses a ticker to run garbage collection at fixed intervals.
// On each tick, it calls the store's gc method, which is responsible
// for removing expired sessions:
func (m *SessionManager) gc(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		m.store.gc(m.idleExpiration, m.absoluteExpiration)
	}
}

// validate method that ensures a given session is valid for use:
func (m *SessionManager) validate(session *Session) bool {

	if time.Since(session.createdAt) > m.absoluteExpiration ||
		time.Since(session.lastActivityAt) > m.idleExpiration {

		// Delete the session from the store
		err := m.store.destroy(session.Id)
		if err != nil {
			panic(err)
		}

		return false
	}
	return true
}

func (m *SessionManager) Start(r *http.Request) (*Session, *http.Request) {
	var session *Session
	// Read from cookie
	cookie, err := r.Cookie(m.cookieName)
	if err == nil {
		session, err = m.store.read(cookie.Value)
		if err != nil {
			log.Printf("Failed to read session from store: %v", err)
		}
	}

	// Generate a new session if the cookie is not present or the session is invalid
	if session == nil || !m.validate(session) {
		session = newSession()
	}

	// Attach the session to the request context
	ctx := context.WithValue(r.Context(), SessionContextKey{}, session)
	r = r.WithContext(ctx)

	return session, r
}

// save a session to the store after updating its lastActivityAt field:
func (m *SessionManager) save(session *Session) error {
	session.lastActivityAt = time.Now()

	err := m.store.write(session)
	if err != nil {
		log.Printf("Failed to write session to store: %v", err)
		return err
	}

	return nil
}

func (m *SessionManager) Migrate(session *Session) error {

	err := m.store.destroy(session.Id)
	if err != nil {
		return err
	}

	session.Id = generateSessionID()
	session.Put("csrf_token", generateCSRFToken())

	return nil
}

func (m *SessionManager) ReGenerate(r *http.Request, user foundation.Authenticatable, attrs map[string]any) error {
	sess := GetSession(r)
	err := m.Migrate(sess)
	if err != nil {
		log.Printf("error migrating session: %s\n", err)
		return err
	}

	sess.Put("user_id", user.GetAuthIdentifier())
	sess.Put("user", user)
	sess.Put("attrs", attrs)
	sess.ClearErrors()

	gonertia.SetProp(r.Context(), "csrf_token", sess.Get("csrf_token"))

	return nil
}

func (m *SessionManager) Invalidate(r *http.Request) error {
	sess := GetSession(r)
	err := m.Migrate(sess)
	if err != nil {
		log.Printf("error migrating session: %s\n", err)
		return err
	}

	sess.Put("user_id", float64(0))
	sess.Put("user", nil)

	return nil
}

func (m *SessionManager) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, rws := m.Start(r)

		sw := &sessionResponseWriter{
			ResponseWriter: w,
			sessionManager: m,
			request:        rws,
		}

		// Add essential headers
		w.Header().Add("Vary", "Cookie")
		w.Header().Add("Cache-Control", `no-cache="Set-Cookie"`)

		if r.Method == http.MethodPost ||
			r.Method == http.MethodPut ||
			r.Method == http.MethodPatch ||
			r.Method == http.MethodDelete {
			if !m.verifyCSRFToken(rws, session) {
				// Can we trigger inertia from here?
				http.Error(sw, "CSRF token mismatch", http.StatusForbidden)
				return
			}
		}

		// Call the next handler and the new response writer and new request
		next.ServeHTTP(sw, rws)

		ip, _ := foundation.GetIpAddress(r)
		session.IpAddress = ip
		session.UserAgent = r.UserAgent()

		m.save(session)

		// Write the session cookie to the response if not already written
		writeCookieIfNecessary(sw)
	})
}

func (m *SessionManager) verifyCSRFToken(r *http.Request, session *Session) bool {
	sToken, ok := session.Get("csrf_token").(string)
	if !ok {
		return false
	}

	token := r.FormValue("csrf_token")
	if token == "" {
		token = r.Header.Get("X-CSRF-Token")
	}

	return sToken == token
}

func (m *SessionManager) AgeFlash(session *Session) {
	session.ClearErrors()
	m.save(session)
}

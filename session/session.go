package session

import (
	"crypto/rand"
	"encoding/base64"
	"io"
	"net/http"
	"time"

	"github.com/martin3zra/forge/foundation"
)

func generateCSRFToken() string {
	token := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, token)
	if err != nil {
		panic("failed to generate CSRF token")
	}

	return base64.RawURLEncoding.EncodeToString(token)
}

func generateSessionID() string {
	id := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, id)
	if err != nil {
		panic("failed to generate session ID")
	}

	return base64.RawURLEncoding.EncodeToString(id)
}

func newSession() *Session {
	return &Session{
		Id:             generateSessionID(),
		createdAt:      time.Now(),
		lastActivityAt: time.Now(),
		payload:        map[string]any{"csrf_token": generateCSRFToken()},
	}
}

func (s *Session) All() map[string]any {
	return s.payload
}

func (s *Session) Get(key string) any {
	return s.payload[key]
}

func (s *Session) Put(key string, value any) {
	s.payload[key] = value
}

func (s *Session) Delete(key string) {
	delete(s.payload, key)
}

func (s *Session) Flash(name string, value any) {
	// Ensure flash map exists
	flash, ok := s.payload["flash"].(map[string]any)
	if !ok || flash == nil {
		flash = make(map[string]any)
		s.payload["flash"] = flash
	}

	// Add or overwrite a single key
	flash[name] = value
}

func (s *Session) Errors(name string, value string) {

	errors, ok := s.payload["errors"]
	if !ok {
		item := map[string][]string{}
		item[name] = []string{value}
		s.payload["errors"] = item
		return
	}

	data := errors.(map[string][]string)
	newMap := map[string][]string{}
	newMap[name] = []string{value}
	s.payload["errors"] = mergeMaps(data, newMap)
}

func (s *Session) FormErrors(values foundation.ErrorBag) {
	errors, ok := s.payload["errors"]
	if !ok {
		s.payload["errors"] = values
		return
	}
	data := errors.(map[string][]string)

	s.payload["errors"] = mergeMaps(data, values)
}

func (s *Session) ClearErrors() {
	s.payload["errors"] = map[string][]string{}
	s.payload["flash"] = map[string]any{}
}

func GetSession(r *http.Request) *Session {
	session, ok := r.Context().Value(SessionContextKey{}).(*Session)
	if !ok {
		panic("session not found in request context")
	}

	return session
}

func mergeMaps(maps ...map[string][]string) map[string][]string {
	result := make(map[string][]string)
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}

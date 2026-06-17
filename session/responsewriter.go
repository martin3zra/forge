package session

import (
	"net/http"
	"time"
)

func (w *sessionResponseWriter) Write(b []byte) (int, error) {
	writeCookieIfNecessary(w)

	return w.ResponseWriter.Write(b)
}

func (w *sessionResponseWriter) WriteHeader(statusCode int) {
	writeCookieIfNecessary(w)

	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *sessionResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func writeCookieIfNecessary(w *sessionResponseWriter) {
	if w.done {
		return
	}

	session, ok := w.request.Context().Value(SessionContextKey{}).(*Session)
	if !ok {
		panic("session not found in request context")
	}

	cookie := &http.Cookie{
		Name:     w.sessionManager.cookieName,
		Value:    session.Id,
		HttpOnly: w.sessionManager.httpOnly,
		Path:     "/",
		Secure:   secureOnlyWithHttps(w),
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(w.sessionManager.idleExpiration),
		MaxAge:   int(w.sessionManager.idleExpiration / time.Second),
	}

	// as we are using localhost, we don't need to set the domain
	// otherwise the cookie will not be set
	// cookie.Domain = w.domain

	http.SetCookie(w.ResponseWriter, cookie)

	w.done = true
}

func secureOnlyWithHttps(w *sessionResponseWriter) bool {
	// Flag this cookie as secure only if we build for production
	// and the request was made it using https scheme
	return w.sessionManager.secure && w.request.TLS != nil
}

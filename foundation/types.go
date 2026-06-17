package foundation

import (
	"net/http"
	"time"
)

type ErrorBag map[string][]string

// Set sets the key to value. It replaces any existing
// values.
func (v ErrorBag) Set(key, value string) {
	if v == nil {
		v = make(ErrorBag)
	}
	v[key] = append(v[key], value)
}

type Status string

type Timestamps struct {
	CreatedAt *time.Time `json:"created_at,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// BodyContract interface that ensure his implementation have
// a validation methods, use to validate the request body
type BodyContract interface {
	Validate() ErrorBag
}

type ErrorFormatter interface {
	Status() int
	Error() string
}

type Unauthorized struct{}

func (Unauthorized) Status() int {
	return http.StatusForbidden
}

func (Unauthorized) Error() string {
	return "Unauthorized"
}

type UnprocessableEntity struct{}

func (UnprocessableEntity) Status() int {
	return http.StatusUnprocessableEntity
}

func (UnprocessableEntity) Error() string {
	return "Unprocessable Entity"
}

type BadRequest struct{}

func (BadRequest) Status() int {
	return http.StatusBadRequest
}

func (BadRequest) Error() string {
	return "Bad Request"
}

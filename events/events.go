// Package events is a minimal synchronous, in-transaction domain-event
// dispatcher. Listeners registered for an event run against the same *sql.Tx as
// the code that raised it, in registration order; the first listener error stops
// dispatch and is returned so the caller can roll the transaction back. This
// keeps a write path focused on its own job while side effects react to the
// event, all within one atomic unit.
//
// The package has no dependencies beyond the standard library, so it stays
// reusable and cycle-free; concrete events and listeners live in the application.
package events

import (
	"context"
	"database/sql"
)

// Event is a domain event. Name routes it to its listeners.
type Event interface {
	Name() string
}

// Listener handles an event within the dispatching transaction.
type Listener interface {
	Handle(ctx context.Context, tx *sql.Tx, e Event) error
}

// Dispatcher routes events to the listeners registered for them.
type Dispatcher struct {
	listeners map[string][]Listener
}

// NewDispatcher returns an empty dispatcher.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{listeners: make(map[string][]Listener)}
}

// On registers one or more listeners for an event type (matched by Name).
// Registration order is preserved and honored at dispatch time.
func (d *Dispatcher) On(e Event, ls ...Listener) {
	d.listeners[e.Name()] = append(d.listeners[e.Name()], ls...)
}

// Dispatch runs every listener registered for the event, in order, against tx.
// The first error stops dispatch and is returned; the caller owns the tx and is
// responsible for rolling it back.
func (d *Dispatcher) Dispatch(ctx context.Context, tx *sql.Tx, e Event) error {
	for _, l := range d.listeners[e.Name()] {
		if err := l.Handle(ctx, tx, e); err != nil {
			return err
		}
	}
	return nil
}

// Package events is a minimal synchronous, in-transaction domain-event
// dispatcher. A listener registered for an event type runs — against the
// transaction the event was raised in — whenever a value of that type is
// dispatched, in registration order; the first listener error stops the
// rest and is returned so the caller can roll the transaction back. This
// lets a write path stay focused on its own job while side effects react
// within the same atomic unit.
//
// The transaction handle is a type parameter on Dispatcher, so an app that
// writes through *sql.Tx and one that writes through an ORM's own tx type
// both use this package without it depending on either. Events are plain
// values: an event's type is its routing key, so there's no Name method or
// registration constant to keep in sync.
//
// Depends only on the standard library.
package events

import (
	"context"
	"reflect"
)

// Dispatcher routes a dispatched event to the listeners registered for its
// type. Tx is the transaction handle threaded through to every listener.
// The zero value is not usable — build one with NewDispatcher.
type Dispatcher[Tx any] struct {
	listeners map[reflect.Type][]func(context.Context, Tx, any) error
}

// NewDispatcher returns an empty dispatcher whose listeners receive a Tx.
func NewDispatcher[Tx any]() *Dispatcher[Tx] {
	return &Dispatcher[Tx]{listeners: make(map[reflect.Type][]func(context.Context, Tx, any) error)}
}

// On registers fn as a listener for event type E. Several listeners for the
// same event keep their registration order at dispatch time. Register the
// event's value type, not a pointer to it — On panics on a pointer type,
// since Dispatch keys on the value type and the listener would never fire.
func On[Tx any, E any](d *Dispatcher[Tx], fn func(ctx context.Context, tx Tx, event E) error) {
	t := reflect.TypeFor[E]()
	if t == nil || t.Kind() == reflect.Pointer {
		panic("events: On must be given a concrete event value type, got " + typeName(t))
	}
	d.listeners[t] = append(d.listeners[t], func(ctx context.Context, tx Tx, e any) error {
		return fn(ctx, tx, e.(E))
	})
}

// Dispatch runs every listener registered for e's dynamic type, in
// registration order, against tx. The first error stops dispatch and is
// returned; the caller owns tx and is responsible for rolling it back.
// Dispatch on a nil *Dispatcher is a no-op, so a dispatcher can be wired as
// an optional dependency.
func (d *Dispatcher[Tx]) Dispatch(ctx context.Context, tx Tx, e any) error {
	if d == nil {
		return nil
	}
	for _, l := range d.listeners[reflect.TypeOf(e)] {
		if err := l(ctx, tx, e); err != nil {
			return err
		}
	}
	return nil
}

func typeName(t reflect.Type) string {
	if t == nil {
		return "nil"
	}
	return t.String()
}

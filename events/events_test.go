package events_test

import (
	"context"
	"errors"
	"testing"

	"github.com/martin3zra/forge/events"
)

type invoiceCreated struct{ id int }
type invoiceVoided struct{ id int }

// tx stands in for a real transaction handle — the package doesn't care
// what Tx is, only that it's threaded through to every listener unchanged.
type tx struct{ name string }

func TestDispatchRunsListenersInOrder(t *testing.T) {
	var log []string
	d := events.NewDispatcher[*tx]()
	events.On(d, func(_ context.Context, _ *tx, _ invoiceCreated) error { log = append(log, "a"); return nil })
	events.On(d, func(_ context.Context, _ *tx, _ invoiceCreated) error { log = append(log, "b"); return nil })

	if err := d.Dispatch(context.Background(), &tx{}, invoiceCreated{1}); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if len(log) != 2 || log[0] != "a" || log[1] != "b" {
		t.Fatalf("want [a b], got %v", log)
	}
}

func TestDispatchStopsOnFirstError(t *testing.T) {
	var log []string
	d := events.NewDispatcher[*tx]()
	events.On(d, func(context.Context, *tx, invoiceCreated) error { log = append(log, "a"); return errors.New("boom") })
	events.On(d, func(context.Context, *tx, invoiceCreated) error { log = append(log, "b"); return nil })

	if err := d.Dispatch(context.Background(), nil, invoiceCreated{1}); err == nil {
		t.Fatal("expected the failing listener's error")
	}
	if len(log) != 1 || log[0] != "a" {
		t.Fatalf("second listener must not run after an error; got %v", log)
	}
}

// Events route by type: a listener for one event is not called for another.
func TestDispatchRoutesByType(t *testing.T) {
	var got []string
	d := events.NewDispatcher[*tx]()
	events.On(d, func(context.Context, *tx, invoiceCreated) error { got = append(got, "created"); return nil })
	events.On(d, func(context.Context, *tx, invoiceVoided) error { got = append(got, "voided"); return nil })

	if err := d.Dispatch(context.Background(), nil, invoiceVoided{1}); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if len(got) != 1 || got[0] != "voided" {
		t.Fatalf("listener for invoiceCreated must not run for invoiceVoided; got %v", got)
	}
}

func TestDispatchNoListenersIsNoOp(t *testing.T) {
	d := events.NewDispatcher[*tx]()
	if err := d.Dispatch(context.Background(), nil, invoiceCreated{1}); err != nil {
		t.Fatalf("dispatch with no listeners should be a no-op: %v", err)
	}
}

func TestDispatchNilDispatcherIsNoOp(t *testing.T) {
	var d *events.Dispatcher[*tx]
	if err := d.Dispatch(context.Background(), nil, invoiceCreated{1}); err != nil {
		t.Fatalf("dispatch on a nil dispatcher should be a no-op: %v", err)
	}
}

func TestOnRejectsPointerEventType(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected On to panic when given a pointer event type")
		}
	}()
	d := events.NewDispatcher[*tx]()
	events.On(d, func(context.Context, *tx, *invoiceCreated) error { return nil })
}

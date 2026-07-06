package events_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/martin3zra/forge/events"
)

type ev struct{ name string }

func (e ev) Name() string { return e.name }

// recorder appends its tag to a shared log when handled, optionally failing.
type recorder struct {
	tag  string
	log  *[]string
	fail bool
}

func (r recorder) Handle(_ context.Context, _ *sql.Tx, _ events.Event) error {
	*r.log = append(*r.log, r.tag)
	if r.fail {
		return errors.New("boom")
	}
	return nil
}

func TestDispatchRunsListenersInOrder(t *testing.T) {
	var log []string
	d := events.NewDispatcher()
	d.On(ev{"invoice.created"}, recorder{"a", &log, false}, recorder{"b", &log, false})

	if err := d.Dispatch(context.Background(), nil, ev{"invoice.created"}); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if len(log) != 2 || log[0] != "a" || log[1] != "b" {
		t.Fatalf("want [a b], got %v", log)
	}
}

func TestDispatchStopsOnFirstError(t *testing.T) {
	var log []string
	d := events.NewDispatcher()
	d.On(ev{"x"}, recorder{"a", &log, true}, recorder{"b", &log, false})

	if err := d.Dispatch(context.Background(), nil, ev{"x"}); err == nil {
		t.Fatal("expected the failing listener's error")
	}
	if len(log) != 1 || log[0] != "a" {
		t.Fatalf("second listener must not run after an error; got %v", log)
	}
}

func TestDispatchNoListenersIsNoOp(t *testing.T) {
	d := events.NewDispatcher()
	if err := d.Dispatch(context.Background(), nil, ev{"none"}); err != nil {
		t.Fatalf("dispatch with no listeners should be a no-op: %v", err)
	}
}

// Events route by Name: a listener for one event is not called for another.
func TestDispatchRoutesByName(t *testing.T) {
	var log []string
	d := events.NewDispatcher()
	d.On(ev{"a"}, recorder{"a-listener", &log, false})

	if err := d.Dispatch(context.Background(), nil, ev{"b"}); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if len(log) != 0 {
		t.Fatalf("listener for a must not run for b; got %v", log)
	}
}

package goat

import (
	"context"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestEnvironmentClone(t *testing.T) {
	sm1 := newTestStateMachine(newTestState("state1"))
	sm2 := newTestStateMachine(newTestState("state2"))

	event1 := &testEvent{Value: 1}
	event2 := &testEvent{Value: 2}
	event3 := &testEvent{Value: 3}

	original := newTestEnvironment(sm1, sm2)
	original.queue[sm1.id()] = []AbstractEvent{event1, event2}
	original.queue[sm2.id()] = []AbstractEvent{event3}

	cloned := original.clone()

	opts := cmp.Options{
		cmp.AllowUnexported(environment{}, testStateMachine{}, StateMachine{}, testState{}, State{}, testEvent{}, Event{}),
		cmpopts.IgnoreFields(StateMachine{}, "EventHandlers", "HandlerBuilders"),
	}
	if diff := cmp.Diff(original, cloned, opts); diff != "" {
		t.Errorf("environment mismatch (-original +cloned):\n%s", diff)
	}

	for id, sm := range original.machines {
		clonedSm := cloned.machines[id]
		if reflect.ValueOf(sm).Pointer() == reflect.ValueOf(clonedSm).Pointer() {
			t.Errorf("Expected different pointer addresses for machine %s", id)
		}
	}

	for smID, events := range original.queue {
		clonedEvents := cloned.queue[smID]
		for i, event := range events {
			if reflect.ValueOf(event).Pointer() == reflect.ValueOf(clonedEvents[i]).Pointer() {
				t.Errorf("Expected different pointer addresses for event %d of machine %s", i, smID)
			}
		}
	}

}

func TestEnvironmentEnqueueEvent(t *testing.T) {
	sm := newTestStateMachine(newTestState("initial"))
	env := newTestEnvironment(sm)
	event1 := &testEvent{Value: 1}
	event2 := &testEvent{Value: 2}
	expectedQueue := map[string][]AbstractEvent{
		sm.id(): {event1, event2},
	}

	env.enqueueEvent(sm, event1)
	env.enqueueEvent(sm, event2)

	if diff := cmp.Diff(expectedQueue, env.queue, cmp.AllowUnexported(testEvent{}, Event{})); diff != "" {
		t.Errorf("Queue mismatch (-expected +actual):\n%s", diff)
	}
}

func TestEnvironmentDequeueEvent(t *testing.T) {
	tests := []struct {
		name          string
		smID          string
		initialQueue  map[string][]AbstractEvent
		expectedEvent *testEvent
		expectedOK    bool
		expectedQueue map[string][]AbstractEvent
	}{
		{
			name: "empty queue",
			smID: "test-sm-id",
			initialQueue: map[string][]AbstractEvent{
				"test-sm-id": {},
			},
			expectedOK: false,
		},
		{
			name:          "non-existent state machine ID",
			smID:          "non-existent-id",
			initialQueue:  make(map[string][]AbstractEvent),
			expectedEvent: nil,
			expectedOK:    false,
		},
		{
			name: "dequeue from queue with events",
			smID: "test-sm-id",
			initialQueue: map[string][]AbstractEvent{
				"test-sm-id": {&testEvent{Value: 1}, &testEvent{Value: 2}, &testEvent{Value: 3}},
			},
			expectedEvent: &testEvent{Value: 1},
			expectedOK:    true,
			expectedQueue: map[string][]AbstractEvent{
				"test-sm-id": {&testEvent{Value: 2}, &testEvent{Value: 3}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := environment{
				queue: tt.initialQueue,
			}

			event, ok := env.dequeueEvent(tt.smID)
			if ok != tt.expectedOK {
				t.Errorf("Expected ok to be %v, but got %v", tt.expectedOK, ok)
			}
			if !tt.expectedOK {
				return
			}

			if diff := cmp.Diff(tt.expectedEvent, event.(*testEvent), cmp.AllowUnexported(testEvent{}, Event{})); diff != "" {
				t.Errorf("Event mismatch (-expected +actual):\n%s", diff)
			}
			if diff := cmp.Diff(tt.expectedQueue, env.queue, cmp.AllowUnexported(testEvent{}, Event{})); diff != "" {
				t.Errorf("Queue mismatch (-expected +actual):\n%s", diff)
			}

		})
	}

}

func TestSendTo(t *testing.T) {
	tests := []struct {
		name     string
		setupSMs func() ([]*testStateMachine, *testStateMachine)
		event    AbstractEvent
	}{
		{
			name: "send event to single state machine",
			setupSMs: func() ([]*testStateMachine, *testStateMachine) {
				return []*testStateMachine{newTestStateMachine(newTestState("initial"))}, newTestStateMachine(newTestState("initial"))
			},
			event: &testEvent{Value: 42},
		},
		{
			name: "send event to specific state machine in multi-SM environment",
			setupSMs: func() ([]*testStateMachine, *testStateMachine) {
				return []*testStateMachine{
					newTestStateMachine(newTestState("sm1")),
					newTestStateMachine(newTestState("sm2")),
					newTestStateMachine(newTestState("sm3")),
				}, newTestStateMachine(newTestState("sm2"))
			},
			event: &testEvent{Value: 123},
		},
		{
			name: "send multiple events to same state machine",
			setupSMs: func() ([]*testStateMachine, *testStateMachine) {
				return []*testStateMachine{newTestStateMachine(newTestState("initial"))}, newTestStateMachine(newTestState("initial"))
			},
			event: &haltEvent{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sms, target := tt.setupSMs()
			env := newTestEnvironment(sms...)

			ctx := context.WithValue(t.Context(), envKey{}, &env)

			SendTo(ctx, target, tt.event)

			queue := env.queue[target.id()]
			if len(queue) != 1 {
				t.Errorf("Expected 1 event in queue, got %d", len(queue))
			}
			if !cmp.Equal(queue[len(queue)-1], tt.event) {
				t.Errorf("Last queued event mismatch: %v", cmp.Diff(tt.event, queue[len(queue)-1]))
			}
		})
	}
}

func TestGoto(t *testing.T) {
	tests := []struct {
		name         string
		initialState AbstractState
		targetState  AbstractState
		wantEvents   []AbstractEvent
	}{
		{
			name:         "transition from initial to target state",
			initialState: newTestState("initial"),
			targetState:  newTestState("target"),
			wantEvents: []AbstractEvent{
				&exitEvent{},
				&transitionEvent{To: newTestState("target")},
				&entryEvent{},
			},
		},
		{
			name:         "transition to same state",
			initialState: newTestState("same"),
			targetState:  newTestState("same"),
			wantEvents: []AbstractEvent{
				&exitEvent{},
				&transitionEvent{To: newTestState("same")},
				&entryEvent{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := newTestStateMachine(tt.initialState, tt.targetState)
			env := newTestEnvironment(sm)

			ctx := context.WithValue(t.Context(), envKey{}, &env)
			ctx = context.WithValue(ctx, smKey{}, sm)

			Goto(ctx, tt.targetState)

			queue := env.queue[sm.id()]
			if !cmp.Equal(queue, tt.wantEvents) {
				t.Errorf("Queue mismatch: %v", cmp.Diff(tt.wantEvents, queue))
			}
		})
	}
}

func TestHalt(t *testing.T) {
	t.Run("enqueues haltEvent to target state machine", func(t *testing.T) {
		sm := newTestStateMachine(newTestState("initial"))
		env := newTestEnvironment(sm)

		ctx := context.WithValue(context.Background(), envKey{}, &env)

		Halt(ctx, sm)

		queue := env.queue[sm.id()]
		wantEvents := []AbstractEvent{&haltEvent{}}

		if !cmp.Equal(queue, wantEvents) {
			t.Errorf("Queue mismatch: %v", cmp.Diff(wantEvents, queue))
		}
	})
}

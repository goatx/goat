package goat

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestInvariantFor(t *testing.T) {
	tests := []struct {
		name       string
		setup      func() (*testStateMachine, world)
		checkFunc  func(*testStateMachine) bool
		wantResult bool
	}{
		{
			name: "always true invariant",
			setup: func() (*testStateMachine, world) {
				sm := newTestStateMachine(newTestState("initial"))
				return sm, newTestWorld(newTestEnvironment(sm))
			},
			checkFunc: func(sm *testStateMachine) bool {
				return true
			},
			wantResult: true,
		},
		{
			name: "always false invariant",
			setup: func() (*testStateMachine, world) {
				sm := newTestStateMachine(newTestState("initial"))
				return sm, newTestWorld(newTestEnvironment(sm))
			},
			checkFunc: func(sm *testStateMachine) bool {
				return false
			},
			wantResult: false,
		},
		{
			name: "check specific state",
			setup: func() (*testStateMachine, world) {
				sm := newTestStateMachine(newTestState("initial"))
				return sm, newTestWorld(newTestEnvironment(sm))
			},
			checkFunc: func(sm *testStateMachine) bool {
				currentState := sm.currentState().(*testState)
				return currentState.Name == "initial"
			},
			wantResult: true,
		},
		{
			name: "non-existent state machine",
			setup: func() (*testStateMachine, world) {
				sm := newTestStateMachine(newTestState("initial"))
				emptyEnv := environment{
					machines: make(map[string]AbstractStateMachine),
					queue:    make(map[string][]AbstractEvent),
				}
				return sm, newTestWorld(emptyEnv)
			},
			checkFunc: func(sm *testStateMachine) bool {
				return true
			},
			wantResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm, w := tt.setup()
			invariant := NewInvariant(sm, tt.checkFunc)

			got := invariant.Evaluate(w)
			if diff := cmp.Diff(tt.wantResult, got); diff != "" {
				t.Errorf("Evaluate() result mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestBoolInvariant(t *testing.T) {
	tests := []struct {
		name       string
		value      bool
		wantResult bool
	}{
		{
			name:       "true invariant",
			value:      true,
			wantResult: true,
		},
		{
			name:       "false invariant",
			value:      false,
			wantResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := newTestWorld(newTestEnvironment())
			invariant := BoolInvariant(tt.value)

			got := invariant.Evaluate(w)
			if diff := cmp.Diff(tt.wantResult, got); diff != "" {
				t.Errorf("Evaluate() result mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

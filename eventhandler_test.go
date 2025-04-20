package goat

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestOnEvent(t *testing.T) {
	t.Run("registers handler builder for specified state and event", func(t *testing.T) {
		spec := &StateMachineSpec[*testStateMachine]{
			prototype:       &testStateMachine{},
			handlerBuilders: make(map[AbstractState][]handlerBuilderInfo),
		}
		state := &testState{Name: "test"}
		event := &testEvent{Value: 1}

		OnEvent(spec, state, event, func(ctx context.Context, e *testEvent, sm *testStateMachine) {})

		builders, exists := spec.handlerBuilders[state]
		if !exists {
			t.Error("Handler builder should be registered for the specified state")
		}
		if len(builders) != 1 {
			t.Error("Exactly one handler builder should be registered")
		}

		builderInfo := builders[0]
		if !sameEvent(builderInfo.event, event) {
			t.Error("Handler builder should be registered for the specified event")
		}

	})
}

func TestOnEntry(t *testing.T) {
	t.Run("registers entry handler builder for specified state", func(t *testing.T) {
		spec := &StateMachineSpec[*testStateMachine]{
			prototype:       &testStateMachine{},
			handlerBuilders: make(map[AbstractState][]handlerBuilderInfo),
		}
		state := &testState{Name: "entry_test"}

		OnEntry(spec, state, func(ctx context.Context, sm *testStateMachine) {})

		builders, exists := spec.handlerBuilders[state]
		if !exists {
			t.Error("Handler builder should be registered for the specified state")
		}
		if len(builders) != 1 {
			t.Error("Exactly one handler builder should be registered")
		}

		builderInfo := builders[0]
		if !sameEvent(builderInfo.event, &entryEvent{}) {
			t.Error("Handler builder should be registered for entryEvent")
		}
	})
}

func TestOnExit(t *testing.T) {
	t.Run("registers exit handler builder for specified state", func(t *testing.T) {
		spec := &StateMachineSpec[*testStateMachine]{
			prototype:       &testStateMachine{},
			handlerBuilders: make(map[AbstractState][]handlerBuilderInfo),
		}
		state := &testState{Name: "exit_test"}

		OnExit(spec, state, func(ctx context.Context, sm *testStateMachine) {})

		builders, exists := spec.handlerBuilders[state]
		if !exists {
			t.Error("Handler builder should be registered for the specified state")
		}
		if len(builders) != 1 {
			t.Error("Exactly one handler builder should be registered")
		}

		builderInfo := builders[0]
		if !sameEvent(builderInfo.event, &exitEvent{}) {
			t.Error("Handler builder should be registered for exitEvent")
		}
	})
}

func TestOnTransition(t *testing.T) {
	t.Run("registers transition handler builder for specified state", func(t *testing.T) {
		spec := &StateMachineSpec[*testStateMachine]{
			prototype:       &testStateMachine{},
			handlerBuilders: make(map[AbstractState][]handlerBuilderInfo),
		}
		state := &testState{Name: "transition_test"}

		OnTransition(spec, state, func(ctx context.Context, toState AbstractState, sm *testStateMachine) {})

		builders, exists := spec.handlerBuilders[state]
		if !exists {
			t.Error("Handler builder should be registered for the specified state")
		}
		if len(builders) != 1 {
			t.Error("Exactly one handler builder should be registered")
		}

		builderInfo := builders[0]
		if !sameEvent(builderInfo.event, &transitionEvent{}) {
			t.Error("Handler builder should be registered for transitionEvent")
		}
	})
}

func TestOnHalt(t *testing.T) {
	t.Run("registers halt handler builder for specified state", func(t *testing.T) {
		spec := &StateMachineSpec[*testStateMachine]{
			prototype:       &testStateMachine{},
			handlerBuilders: make(map[AbstractState][]handlerBuilderInfo),
		}
		state := &testState{Name: "halt_test"}

		OnHalt(spec, state, func(ctx context.Context, sm *testStateMachine) {})

		builders, exists := spec.handlerBuilders[state]
		if !exists {
			t.Error("Handler builder should be registered for the specified state")
		}
		if len(builders) != 1 {
			t.Error("Exactly one handler builder should be registered")
		}

		builderInfo := builders[0]
		if !sameEvent(builderInfo.event, &haltEvent{}) {
			t.Error("Handler builder should be registered for haltEvent")
		}
	})
}

func TestDefaultOnTransitionHandler_handle(t *testing.T) {
	t.Run("returns single local state with updated state machine", func(t *testing.T) {
		handler := &defaultOnTransitionHandler{}
		initialState := newTestState("initial")
		targetState := newTestState("target")
		sm := newTestStateMachine(initialState, targetState)
		env := newTestEnvironment(sm)
		event := &transitionEvent{To: targetState}

		states, err := handler.handle(env, sm.id(), event)
		if err != nil {
			t.Errorf("handle() error = %v", err)
		}
		if len(states) != 1 {
			t.Errorf("Expected 1 state, got %d", len(states))
		}
		newSM := states[0].env.machines[sm.id()]
		if !sameState(newSM.currentState(), targetState) {
			t.Error("State should be changed to target state")
		}
	})
}

func TestDefaultOnHaltHandler_handle(t *testing.T) {
	t.Run("returns single local state with halted state machine", func(t *testing.T) {
		handler := &defaultOnHaltHandler{}
		initialState := newTestState("initial")
		sm := newTestStateMachine(initialState)
		env := newTestEnvironment(sm)
		event := &haltEvent{}

		states, err := handler.handle(env, sm.id(), event)

		if err != nil {
			t.Errorf("handle() error = %v", err)
		}
		if len(states) != 1 {
			t.Errorf("Expected 1 state, got %d", len(states))
		}
		newSM := states[0].env.machines[sm.id()]
		innerSM := getInnerStateMachine(newSM)
		if !innerSM.halted {
			t.Error("State machine should be halted")
		}
	})
}

func TestEventHandlers_handle(t *testing.T) {
	tests := []struct {
		name       string
		handlers   *eventHandlers
		setupEnv   func() environment
		inputEvent AbstractEvent
		wantStates []localState
	}{
		{
			name: "returns nil for non-matching event types",
			handlers: &eventHandlers{
				fs:    []eventHandler{},
				event: &testEvent{Value: 1},
			},
			setupEnv: func() environment {
				return environment{
					machines: make(map[string]AbstractStateMachine),
					queue:    make(map[string][]AbstractEvent),
				}
			},
			inputEvent: &entryEvent{},
			wantStates: nil,
		},
		{
			name: "processes matching event types and returns local states",
			handlers: &eventHandlers{
				fs: []eventHandler{
					func(event AbstractEvent, env *environment) {
						// Handler that modifies environment
					},
				},
				event: &testEvent{Value: 1},
			},
			setupEnv: func() environment {
				return environment{
					machines: make(map[string]AbstractStateMachine),
					queue:    make(map[string][]AbstractEvent),
				}
			},
			inputEvent: &testEvent{Value: 1},
			wantStates: func() []localState {
				env := environment{
					machines: make(map[string]AbstractStateMachine),
					queue:    make(map[string][]AbstractEvent),
				}
				return []localState{{env: env}}
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := tt.setupEnv()

			states, err := tt.handlers.handle(env, "test", tt.inputEvent)

			if err != nil {
				t.Fatalf("handle() error = %v", err)
			}

			if diff := cmp.Diff(tt.wantStates, states, cmp.AllowUnexported(localState{}, environment{})); diff != "" {
				t.Errorf("States mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestTransitionHandlers_handle(t *testing.T) {
	tests := []struct {
		name       string
		handlers   *transitionHandlers
		setupEnv   func() (environment, string, AbstractState)
		event      AbstractEvent
		wantStates []localState
	}{
		{
			name: "single handler transitions state",
			handlers: &transitionHandlers{
				fs: []transitionHandler{
					func(to AbstractState, env *environment) {},
				},
			},
			setupEnv: func() (environment, string, AbstractState) {
				target := newTestState("target")
				sm := newTestStateMachine(newTestState("initial"), target)
				env := newTestEnvironment(sm)
				return env, sm.id(), target
			},
			event: nil,
			wantStates: func() []localState {
				sm := newTestStateMachine(newTestState("target"))
				env := newTestEnvironment(sm)
				sm.setCurrentState(newTestState("target"))
				return []localState{{env: env}}
			}(),
		},
		{
			name: "multiple handlers create multiple states",
			handlers: &transitionHandlers{
				fs: []transitionHandler{
					func(to AbstractState, env *environment) {},
					func(to AbstractState, env *environment) {},
				},
			},
			setupEnv: func() (environment, string, AbstractState) {
				target := newTestState("target")
				sm := newTestStateMachine(newTestState("initial"), target)
				env := newTestEnvironment(sm)
				return env, sm.id(), target
			},
			event: nil,
			wantStates: func() []localState {
				target := newTestState("target")
				sm1 := newTestStateMachine(newTestState("initial"), target)
				env1 := newTestEnvironment(sm1)
				sm1.setCurrentState(newTestState("target"))

				sm2 := newTestStateMachine(newTestState("initial"), target)
				env2 := newTestEnvironment(sm2)
				sm2.setCurrentState(newTestState("target"))

				return []localState{{env: env1}, {env: env2}}
			}(),
		},
		{
			name:     "wrong event type returns nil",
			handlers: &transitionHandlers{fs: []transitionHandler{}},
			setupEnv: func() (environment, string, AbstractState) {
				sm := newTestStateMachine(newTestState("initial"))
				env := newTestEnvironment(sm)
				return env, sm.id(), nil
			},
			event:      &haltEvent{},
			wantStates: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env, smID, target := tt.setupEnv()

			event := tt.event
			if event == nil && target != nil {
				event = &transitionEvent{To: target}
			}

			states, err := tt.handlers.handle(env, smID, event)

			if err != nil {
				t.Fatalf("handle() error = %v", err)
			}

			opts := cmp.Options{
				cmp.AllowUnexported(localState{}, environment{}, testStateMachine{}, StateMachine{}, testState{}, State{}, testEvent{}, Event{}),
				cmpopts.IgnoreFields(StateMachine{}, "EventHandlers", "HandlerBuilders"),
			}
			if diff := cmp.Diff(tt.wantStates, states, opts); diff != "" {
				t.Errorf("States mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestHaltHandlers_handle(t *testing.T) {
	tests := []struct {
		name       string
		handlers   *haltHandlers
		setupEnv   func() (environment, string)
		event      AbstractEvent
		wantStates []localState
	}{
		{
			name: "single handler halts state machine",
			handlers: &haltHandlers{
				fs: []haltHandler{
					func(env *environment) {},
				},
			},
			setupEnv: func() (environment, string) {
				sm := newTestStateMachine(newTestState("initial"))
				env := newTestEnvironment(sm)
				return env, sm.id()
			},
			event: &haltEvent{},
			wantStates: func() []localState {
				sm := newTestStateMachine(newTestState("initial"))
				env := newTestEnvironment(sm)
				innerSM := getInnerStateMachine(sm)
				innerSM.halted = true
				return []localState{{env: env}}
			}(),
		},
		{
			name: "multiple handlers create multiple states",
			handlers: &haltHandlers{
				fs: []haltHandler{
					func(env *environment) {},
					func(env *environment) {},
				},
			},
			setupEnv: func() (environment, string) {
				sm := newTestStateMachine(newTestState("initial"))
				env := newTestEnvironment(sm)
				return env, sm.id()
			},
			event: &haltEvent{},
			wantStates: func() []localState {
				// Each handler creates a separate halted state
				sm1 := newTestStateMachine(newTestState("initial"))
				env1 := newTestEnvironment(sm1)
				innerSM1 := getInnerStateMachine(sm1)
				innerSM1.halted = true

				sm2 := newTestStateMachine(newTestState("initial"))
				env2 := newTestEnvironment(sm2)
				innerSM2 := getInnerStateMachine(sm2)
				innerSM2.halted = true

				return []localState{{env: env1}, {env: env2}}
			}(),
		},
		{
			name:     "wrong event type returns nil",
			handlers: &haltHandlers{fs: []haltHandler{}},
			setupEnv: func() (environment, string) {
				sm := newTestStateMachine(newTestState("initial"))
				env := newTestEnvironment(sm)
				return env, sm.id()
			},
			event:      &entryEvent{},
			wantStates: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env, smID := tt.setupEnv()

			states, err := tt.handlers.handle(env, smID, tt.event)

			if err != nil {
				t.Fatalf("handle() error = %v", err)
			}

			opts := cmp.Options{
				cmp.AllowUnexported(localState{}, environment{}, testStateMachine{}, StateMachine{}, testState{}, State{}, testEvent{}, Event{}),
				cmpopts.IgnoreFields(StateMachine{}, "EventHandlers", "HandlerBuilders"),
			}
			if diff := cmp.Diff(tt.wantStates, states, opts); diff != "" {
				t.Errorf("States mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestEntryHandlers_handle(t *testing.T) {
	tests := []struct {
		name       string
		handlers   *entryHandlers
		setupEnv   func() (environment, string)
		event      AbstractEvent
		wantStates []localState
	}{
		{
			name: "single entry handler",
			handlers: &entryHandlers{
				fs: []entryHandler{
					func(env *environment) {},
				},
			},
			setupEnv: func() (environment, string) {
				sm := newTestStateMachine(newTestState("initial"))
				env := newTestEnvironment(sm)
				return env, sm.id()
			},
			event: &entryEvent{},
			wantStates: func() []localState {
				sm := newTestStateMachine(newTestState("initial"))
				env := newTestEnvironment(sm)
				return []localState{{env: env}}
			}(),
		},
		{
			name: "multiple entry handlers",
			handlers: &entryHandlers{
				fs: []entryHandler{
					func(env *environment) {},
					func(env *environment) {},
					func(env *environment) {},
				},
			},
			setupEnv: func() (environment, string) {
				sm := newTestStateMachine(newTestState("initial"))
				env := newTestEnvironment(sm)
				return env, sm.id()
			},
			event: &entryEvent{},
			wantStates: func() []localState {
				sm1 := newTestStateMachine(newTestState("initial"))
				env1 := newTestEnvironment(sm1)

				sm2 := newTestStateMachine(newTestState("initial"))
				env2 := newTestEnvironment(sm2)

				sm3 := newTestStateMachine(newTestState("initial"))
				env3 := newTestEnvironment(sm3)

				return []localState{{env: env1}, {env: env2}, {env: env3}}
			}(),
		},
		{
			name:     "no handlers",
			handlers: &entryHandlers{fs: []entryHandler{}},
			setupEnv: func() (environment, string) {
				sm := newTestStateMachine(newTestState("initial"))
				env := newTestEnvironment(sm)
				return env, sm.id()
			},
			event:      &entryEvent{},
			wantStates: []localState{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env, smID := tt.setupEnv()

			states, err := tt.handlers.handle(env, smID, tt.event)

			if err != nil {
				t.Fatalf("handle() error = %v", err)
			}

			opts := cmp.Options{
				cmp.AllowUnexported(localState{}, environment{}, testStateMachine{}, StateMachine{}, testState{}, State{}, testEvent{}, Event{}),
				cmpopts.IgnoreFields(StateMachine{}, "EventHandlers", "HandlerBuilders"),
			}
			if diff := cmp.Diff(tt.wantStates, states, opts); diff != "" {
				t.Errorf("States mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestExitHandlers_handle(t *testing.T) {
	tests := []struct {
		name       string
		handlers   *exitHandlers
		setupEnv   func() (environment, string)
		event      AbstractEvent
		wantStates []localState
	}{
		{
			name: "single exit handler",
			handlers: &exitHandlers{
				fs: []exitHandler{
					func(env *environment) {},
				},
			},
			setupEnv: func() (environment, string) {
				sm := newTestStateMachine(newTestState("initial"))
				env := newTestEnvironment(sm)
				return env, sm.id()
			},
			event: &exitEvent{},
			wantStates: func() []localState {
				sm := newTestStateMachine(newTestState("initial"))
				env := newTestEnvironment(sm)
				return []localState{{env: env}}
			}(),
		},
		{
			name: "multiple exit handlers",
			handlers: &exitHandlers{
				fs: []exitHandler{
					func(env *environment) {},
					func(env *environment) {},
				},
			},
			setupEnv: func() (environment, string) {
				sm := newTestStateMachine(newTestState("initial"))
				env := newTestEnvironment(sm)
				return env, sm.id()
			},
			event: &exitEvent{},
			wantStates: func() []localState {
				sm1 := newTestStateMachine(newTestState("initial"))
				env1 := newTestEnvironment(sm1)

				sm2 := newTestStateMachine(newTestState("initial"))
				env2 := newTestEnvironment(sm2)

				return []localState{{env: env1}, {env: env2}}
			}(),
		},
		{
			name:     "no handlers",
			handlers: &exitHandlers{fs: []exitHandler{}},
			setupEnv: func() (environment, string) {
				sm := newTestStateMachine(newTestState("initial"))
				env := newTestEnvironment(sm)
				return env, sm.id()
			},
			event:      &exitEvent{},
			wantStates: []localState{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env, smID := tt.setupEnv()

			states, err := tt.handlers.handle(env, smID, tt.event)

			if err != nil {
				t.Fatalf("handle() error = %v", err)
			}

			opts := cmp.Options{
				cmp.AllowUnexported(localState{}, environment{}, testStateMachine{}, StateMachine{}, testState{}, State{}, testEvent{}, Event{}),
				cmpopts.IgnoreFields(StateMachine{}, "EventHandlers", "HandlerBuilders"),
			}
			if diff := cmp.Diff(tt.wantStates, states, opts); diff != "" {
				t.Errorf("States mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

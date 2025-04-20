package goat

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestModel_Solve(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() model
		want    func() model
		wantErr bool
	}{
		{
			name: "simple state machine with no transitions",
			setup: func() model {
				sm := newTestStateMachine(newTestState("initial"))
				m, _ := newModel(WithStateMachines(sm))
				return m
			},
			want: func() model {
				sm := newTestStateMachine(newTestState("initial"))
				getInnerStateMachine(sm).smID = testStateMachineID

				initialWorld := newWorld(environment{
					machines: map[string]AbstractStateMachine{
						testStateMachineID: sm,
					},
					queue: map[string][]AbstractEvent{
						testStateMachineID: {&entryEvent{}},
					},
				})

				processedWorld := newWorld(environment{
					machines: map[string]AbstractStateMachine{
						testStateMachineID: sm,
					},
					queue: map[string][]AbstractEvent{
						testStateMachineID: {},
					},
				})

				return model{
					worlds: worlds{
						initialWorld.id:   initialWorld,
						processedWorld.id: processedWorld,
					},
					initial: initialWorld,
					accessible: map[worldID][]worldID{
						initialWorld.id:   {processedWorld.id},
						processedWorld.id: {},
					},
					invariants: nil,
				}
			},
			wantErr: false,
		},
		{
			name: "state machine with invariant violation",
			setup: func() model {
				sm := newTestStateMachine(newTestState("initial"))
				inv := BoolInvariant(false) // Always false invariant
				m, _ := newModel(
					WithStateMachines(sm),
					WithInvariants(inv),
				)
				return m
			},
			want: func() model {
				sm := newTestStateMachine(newTestState("initial"))
				getInnerStateMachine(sm).smID = testStateMachineID

				initialWorld := newWorld(environment{
					machines: map[string]AbstractStateMachine{
						testStateMachineID: sm,
					},
					queue: map[string][]AbstractEvent{
						testStateMachineID: {&entryEvent{}},
					},
				})
				initialWorld.invariantViolation = false

				processedWorld := newWorld(environment{
					machines: map[string]AbstractStateMachine{
						testStateMachineID: sm,
					},
					queue: map[string][]AbstractEvent{
						testStateMachineID: {},
					},
				})
				processedWorld.invariantViolation = true

				initialWorldInMap := initialWorld
				initialWorldInMap.invariantViolation = true

				return model{
					worlds: worlds{
						initialWorld.id:   initialWorldInMap,
						processedWorld.id: processedWorld,
					},
					initial: initialWorld,
					accessible: map[worldID][]worldID{
						initialWorld.id:   {processedWorld.id},
						processedWorld.id: {},
					},
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.setup()
			err := m.Solve()
			if (err != nil) != tt.wantErr {
				t.Errorf("Solve() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.want != nil {
				expected := tt.want()

				opts := cmp.Options{
					cmpopts.IgnoreFields(StateMachine{}, "EventHandlers", "HandlerBuilders"),
					cmpopts.IgnoreFields(model{}, "invariants"), // Ignore function pointers
					cmp.AllowUnexported(model{}, world{}, environment{}, StateMachine{}),
				}

				if diff := cmp.Diff(expected, m, opts); diff != "" {
					t.Errorf("Solve() result mismatch (-want +got):\n%s", diff)
				}
			}

			if !m.worlds.member(m.initial) {
				t.Error("Initial world should be in explored worlds")
			}
		})
	}
}

func TestModel_evaluateInvariants(t *testing.T) {
	tests := []struct {
		name          string
		setup         func() (model, world)
		wantViolation bool
	}{
		{
			name: "no invariants",
			setup: func() (model, world) {
				sm := newTestStateMachine(newTestState("initial"), newTestState("target"))
				m, _ := newModel(WithStateMachines(sm))
				w := initialWorld(sm)
				return m, w
			},
			wantViolation: false,
		},
		{
			name: "passing invariant",
			setup: func() (model, world) {
				sm := newTestStateMachine(newTestState("initial"), newTestState("target"))
				inv := BoolInvariant(true)
				m, _ := newModel(
					WithStateMachines(sm),
					WithInvariants(inv),
				)
				w := initialWorld(sm)
				return m, w
			},
			wantViolation: false,
		},
		{
			name: "failing invariant",
			setup: func() (model, world) {
				sm := newTestStateMachine(newTestState("initial"), newTestState("target"))
				inv := BoolInvariant(false)
				m, _ := newModel(
					WithStateMachines(sm),
					WithInvariants(inv),
				)
				w := initialWorld(sm)
				return m, w
			},
			wantViolation: true,
		},
		{
			name: "multiple invariants with one failing",
			setup: func() (model, world) {
				sm := newTestStateMachine(newTestState("initial"), newTestState("target"))
				inv1 := BoolInvariant(true)
				inv2 := BoolInvariant(false)
				inv3 := BoolInvariant(true)
				m, _ := newModel(
					WithStateMachines(sm),
					WithInvariants(inv1, inv2, inv3),
				)
				w := initialWorld(sm)
				return m, w
			},
			wantViolation: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, w := tt.setup()
			result := m.evaluateInvariants(w)
			if result == tt.wantViolation {
				t.Errorf("evaluateInvariants() returned %v, but expected %v", result, !tt.wantViolation)
			}
		})
	}
}

func TestInitialWorld(t *testing.T) {
	tests := []struct {
		name string
		sms  []AbstractStateMachine
		want world
	}{
		{
			name: "single state machine",
			sms:  []AbstractStateMachine{newTestStateMachine(newTestState("initial"))},
			want: world{
				env: environment{
					machines: map[string]AbstractStateMachine{
						"testStateMachine": func() AbstractStateMachine {
							sm := newTestStateMachine(newTestState("initial"))
							getInnerStateMachine(sm).smID = testStateMachineID
							return sm
						}(),
					},
					queue: map[string][]AbstractEvent{
						testStateMachineID: {&entryEvent{}},
					},
				},
				invariantViolation: false,
			},
		},
		{
			name: "multiple state machines",
			sms: []AbstractStateMachine{
				newTestStateMachine(newTestState("state1")),
				newTestStateMachine(newTestState("state2")),
			},
			want: world{
				env: environment{
					machines: map[string]AbstractStateMachine{
						"testStateMachine": func() AbstractStateMachine {
							sm := newTestStateMachine(newTestState("state1"))
							getInnerStateMachine(sm).smID = testStateMachineID
							return sm
						}(),
						"testStateMachine_1": func() AbstractStateMachine {
							sm := newTestStateMachine(newTestState("state2"))
							getInnerStateMachine(sm).smID = "testStateMachine_1"
							return sm
						}(),
					},
					queue: map[string][]AbstractEvent{
						"testStateMachine":   {&entryEvent{}},
						"testStateMachine_1": {&entryEvent{}},
					},
				},
				invariantViolation: false,
			},
		},
		{
			name: "no state machines",
			sms:  []AbstractStateMachine{},
			want: world{
				env: environment{
					machines: map[string]AbstractStateMachine{},
					queue:    map[string][]AbstractEvent{},
				},
				invariantViolation: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := initialWorld(tt.sms...)

			opts := cmp.Options{
				cmpopts.IgnoreFields(world{}, "id"),
				cmpopts.IgnoreFields(StateMachine{}, "EventHandlers", "HandlerBuilders"),
				cmp.AllowUnexported(world{}, environment{}, StateMachine{}),
			}

			if diff := cmp.Diff(tt.want, got, opts); diff != "" {
				t.Errorf("initialWorld() mismatch (-want +got):\n%s", diff)
			}

			for smID, sm := range got.env.machines {
				innerSM := getInnerStateMachine(sm)

				if innerSM.HandlerBuilders != nil {
					t.Errorf("StateMachine %q: HandlerBuilders should be nil after initialization, got %v", smID, innerSM.HandlerBuilders)
				}

				if innerSM.EventHandlers == nil {
					t.Errorf("StateMachine %q: EventHandlers should be initialized, got nil", smID)
				}
			}
		})
	}
}

type errorHandler struct{}

func (errorHandler) handle(_ environment, _ string, _ AbstractEvent) ([]localState, error) {
	return nil, errors.New("test error")
}

func TestStepGlobal(t *testing.T) {

	tests := []struct {
		name    string
		setup   func() world
		want    func() []world
		wantErr bool
	}{
		{
			name: "returns error when handler fails",
			setup: func() world {
				sm := newTestStateMachine(newTestState("initial"))
				innerSM := getInnerStateMachine(sm)
				innerSM.smID = testStateMachineID

				innerSM.EventHandlers = make(map[AbstractState][]handlerInfo)
				innerSM.EventHandlers[sm.currentState()] = []handlerInfo{
					{
						event:   &entryEvent{},
						handler: errorHandler{},
					},
				}

				env := environment{
					machines: map[string]AbstractStateMachine{
						testStateMachineID: sm,
					},
					queue: map[string][]AbstractEvent{
						testStateMachineID: {&entryEvent{}},
					},
				}
				return newWorld(env)
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "returns empty slice when no events to process",
			setup: func() world {
				sm := newTestStateMachine(newTestState("initial"))
				innerSM := getInnerStateMachine(sm)
				innerSM.smID = testStateMachineID
				innerSM.EventHandlers = make(map[AbstractState][]handlerInfo)

				env := environment{
					machines: map[string]AbstractStateMachine{
						testStateMachineID: sm,
					},
					queue: map[string][]AbstractEvent{
						testStateMachineID: {},
					},
				}
				return newWorld(env)
			},
			want: func() []world {
				return []world{}
			},
			wantErr: false,
		},
		{
			name: "processes entryEvent and returns new world",
			setup: func() world {
				sm := newTestStateMachine(newTestState("initial"))
				innerSM := getInnerStateMachine(sm)
				innerSM.smID = testStateMachineID
				innerSM.EventHandlers = make(map[AbstractState][]handlerInfo)

				env := environment{
					machines: map[string]AbstractStateMachine{
						testStateMachineID: sm,
					},
					queue: map[string][]AbstractEvent{
						testStateMachineID: {&entryEvent{}},
					},
				}
				return newWorld(env)
			},
			want: func() []world {
				sm := newTestStateMachine(newTestState("initial"))
				innerSM := getInnerStateMachine(sm)
				innerSM.smID = testStateMachineID
				innerSM.EventHandlers = make(map[AbstractState][]handlerInfo)

				expectedEnv := environment{
					machines: map[string]AbstractStateMachine{
						testStateMachineID: sm,
					},
					queue: map[string][]AbstractEvent{
						testStateMachineID: {},
					},
				}
				return []world{newWorld(expectedEnv)}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := tt.setup()
			got, err := stepGlobal(w)

			if (err != nil) != tt.wantErr {
				t.Errorf("stepGlobal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && got != nil {
				t.Error("stepGlobal() should return nil slice on error")
				return
			}

			if tt.want != nil {
				expected := tt.want()

				opts := cmp.Options{
					cmpopts.IgnoreFields(world{}, "id"),
					cmpopts.IgnoreFields(StateMachine{}, "EventHandlers", "HandlerBuilders"),
					cmp.AllowUnexported(world{}, environment{}, StateMachine{}),
				}

				if diff := cmp.Diff(expected, got, opts); diff != "" {
					t.Errorf("stepGlobal() result mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

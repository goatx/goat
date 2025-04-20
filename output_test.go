package goat

import (
	"bytes"
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestModel_writeDot(t *testing.T) {
	tests := []struct {
		name  string
		setup func() model
		want  string
	}{
		{
			name: "simple state machine",
			setup: func() model {
				sm := newTestStateMachine(newTestState("initial"))
				m, _ := newModel(WithStateMachines(sm))
				_ = m.Solve()
				return m
			},
			want: `digraph {
  5438153399123815847 [ label="StateMachines:
testStateMachine = no fields; State: {Name:Name,Type:string,Value:initial}

QueuedEvents:" ];
  8682599965454615616 [ label="StateMachines:
testStateMachine = no fields; State: {Name:Name,Type:string,Value:initial}

QueuedEvents:
testStateMachine << entryEvent;" ];
  8682599965454615616 [ penwidth=5 ];
  8682599965454615616 -> 5438153399123815847;
}
`,
		},
		{
			name: "state machine with invariant violation",
			setup: func() model {
				sm := newTestStateMachine(newTestState("initial"))
				inv := BoolInvariant(false)
				m, _ := newModel(
					WithStateMachines(sm),
					WithInvariants(inv),
				)
				_ = m.Solve()
				return m
			},
			want: `digraph {
  5438153399123815847 [ label="StateMachines:
testStateMachine = no fields; State: {Name:Name,Type:string,Value:initial}

QueuedEvents:" ];
  5438153399123815847 [ color=red, penwidth=3 ];
  8682599965454615616 [ label="StateMachines:
testStateMachine = no fields; State: {Name:Name,Type:string,Value:initial}

QueuedEvents:
testStateMachine << entryEvent;" ];
  8682599965454615616 [ penwidth=5 ];
  8682599965454615616 [ color=red, penwidth=3 ];
  8682599965454615616 -> 5438153399123815847;
}
`,
		},
		{
			name: "multiple state machines",
			setup: func() model {
				sm1 := newTestStateMachine(newTestState("state1"))
				sm2 := newTestStateMachine(newTestState("state2"))
				m, _ := newModel(WithStateMachines(sm1, sm2))
				_ = m.Solve()
				return m
			},
			want: `digraph {
  1352120299877738753 [ label="StateMachines:
testStateMachine = no fields; State: {Name:Name,Type:string,Value:state1}
testStateMachine = no fields; State: {Name:Name,Type:string,Value:state2}

QueuedEvents:
testStateMachine << entryEvent;" ];
  8000304505176841628 [ label="StateMachines:
testStateMachine = no fields; State: {Name:Name,Type:string,Value:state1}
testStateMachine = no fields; State: {Name:Name,Type:string,Value:state2}

QueuedEvents:" ];
  10115204962392696257 [ label="StateMachines:
testStateMachine = no fields; State: {Name:Name,Type:string,Value:state1}
testStateMachine = no fields; State: {Name:Name,Type:string,Value:state2}

QueuedEvents:
testStateMachine << entryEvent;" ];
  18043829544564786018 [ label="StateMachines:
testStateMachine = no fields; State: {Name:Name,Type:string,Value:state1}
testStateMachine = no fields; State: {Name:Name,Type:string,Value:state2}

QueuedEvents:
testStateMachine << entryEvent;
testStateMachine << entryEvent;" ];
  18043829544564786018 [ penwidth=5 ];
  1352120299877738753 -> 8000304505176841628;
  10115204962392696257 -> 8000304505176841628;
  18043829544564786018 -> 1352120299877738753;
  18043829544564786018 -> 10115204962392696257;
}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.setup()
			var buf bytes.Buffer
			m.writeDot(&buf)
			got := buf.String()

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("writeDot() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestModel_writeLog(t *testing.T) {
	tests := []struct {
		name        string
		setup       func() model
		description string
		want        string
	}{
		{
			name: "no invariant violations",
			setup: func() model {
				sm := newTestStateMachine(newTestState("initial"))
				inv := BoolInvariant(true)
				m, _ := newModel(
					WithStateMachines(sm),
					WithInvariants(inv),
				)
				_ = m.Solve()
				return m
			},
			description: "test invariant",
			want:        "No invariant violations found.\n",
		},
		{
			name: "with invariant violation",
			setup: func() model {
				sm := newTestStateMachine(newTestState("initial"))
				inv := BoolInvariant(false)
				m, _ := newModel(
					WithStateMachines(sm),
					WithInvariants(inv),
				)
				_ = m.Solve()
				return m
			},
			description: "failing test invariant",
			want: `InvariantError:  failing test invariant   ✘
Path (length = 1):
  [0] <-- violation here
  StateMachines:
    Name: testStateMachine, Detail: no fields, State: {Name:Name,Type:string,Value:initial}
  QueuedEvents:
    StateMachine: testStateMachine, Event: entryEvent, Detail: no fields
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.setup()
			var buf bytes.Buffer
			m.writeLog(&buf, tt.description)
			got := buf.String()

			if got != tt.want {
				t.Errorf("writeLog() output mismatch\ngot:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestModel_findPathsToViolations(t *testing.T) {
	tests := []struct {
		name          string
		setup         func() model
		expectedPaths [][]worldID
	}{
		{
			name: "no violations",
			setup: func() model {
				sm := newTestStateMachine(newTestState("initial"))
				inv := BoolInvariant(true)
				m, err := newModel(
					WithStateMachines(sm),
					WithInvariants(inv),
				)
				if err != nil {
					panic(err)
				}
				_ = m.Solve()
				return m
			},
			expectedPaths: nil,
		},
		{
			name: "single violation",
			setup: func() model {
				sm := newTestStateMachine(newTestState("initial"))
				inv := BoolInvariant(false)
				m, err := newModel(
					WithStateMachines(sm),
					WithInvariants(inv),
				)
				if err != nil {
					panic(err)
				}
				_ = m.Solve()
				return m
			},
			expectedPaths: [][]worldID{{8682599965454615616}},
		},
		{
			name: "violation after transition",
			setup: func() model {
				type testCounter struct {
					testStateMachine
					count int
				}

				spec := NewStateMachineSpec(&testCounter{})
				stateA := newTestState("A")
				stateB := newTestState("B")
				spec.DefineStates(stateA, stateB).SetInitialState(stateA)

				OnEntry(spec, stateA, func(ctx context.Context, sm *testCounter) {
					sm.count = 1
					Goto(ctx, stateB)
				})

				OnEntry(spec, stateB, func(ctx context.Context, sm *testCounter) {
					sm.count = 2
				})

				sm, err := spec.NewInstance()
				if err != nil {
					panic(err)
				}

				inv := NewInvariant(sm, func(sm *testCounter) bool {
					return sm.count <= 1
				})

				m, err := newModel(
					WithStateMachines(sm),
					WithInvariants(inv),
				)
				if err != nil {
					panic(err)
				}
				_ = m.Solve()
				return m
			},
			expectedPaths: [][]worldID{
				{5790322525083387874, 15591947093441390666, 10703074720578030081, 15159594575768829045, 8395799135532667686},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.setup()
			actualPaths := m.findPathsToViolations()

			if diff := cmp.Diff(tt.expectedPaths, actualPaths); diff != "" {
				t.Errorf("Violation paths mismatch (-expected +actual):\n%s", diff)
			}
		})
	}
}

func TestWorld_label(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() world
		expected string
	}{
		{
			name: "single state machine with test state",
			setup: func() world {
				sm := newTestStateMachine(newTestState("test"))
				return initialWorld(sm)
			},
			expected: "StateMachines:\ntestStateMachine = no fields; State: {Name:Name,Type:string,Value:test}\n\nQueuedEvents:\ntestStateMachine << entryEvent;",
		},
		{
			name: "single state machine with initial state",
			setup: func() world {
				sm := newTestStateMachine(newTestState("initial"))
				return initialWorld(sm)
			},
			expected: "StateMachines:\ntestStateMachine = no fields; State: {Name:Name,Type:string,Value:initial}\n\nQueuedEvents:\ntestStateMachine << entryEvent;",
		},
		{
			name: "multiple state machines",
			setup: func() world {
				sm1 := newTestStateMachine(newTestState("state1"))
				sm2 := newTestStateMachine(newTestState("state2"))
				return initialWorld(sm1, sm2)
			},
			expected: "StateMachines:\ntestStateMachine = no fields; State: {Name:Name,Type:string,Value:state1}\ntestStateMachine = no fields; State: {Name:Name,Type:string,Value:state2}\n\nQueuedEvents:\ntestStateMachine << entryEvent;\ntestStateMachine << entryEvent;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := tt.setup()
			got := w.label()

			if got != tt.expected {
				t.Errorf("World.label() output mismatch\ngot:\n%q\nwant:\n%q", got, tt.expected)
			}
		})
	}
}

func TestModel_worldsToJSON(t *testing.T) {
	tests := []struct {
		name           string
		setupModel     func() model
		expectedWorlds []worldJSON
	}{
		{
			name: "single state machine creates multiple worlds",
			setupModel: func() model {
				sm := newTestStateMachine(newTestState("initial"))
				m, _ := newModel(WithStateMachines(sm))
				_ = m.Solve()
				return m
			},
			expectedWorlds: []worldJSON{
				{
					InvariantViolation: false,
					StateMachines: []stateMachineJSON{
						{
							ID:      "testStateMachine",
							Name:    "testStateMachine",
							State:   "{Name:Name,Type:string,Value:initial}",
							Details: "no fields",
						},
					},
					QueuedEvents: []eventJSON{},
				},
				{
					InvariantViolation: false,
					StateMachines: []stateMachineJSON{
						{
							ID:      "testStateMachine",
							Name:    "testStateMachine",
							State:   "{Name:Name,Type:string,Value:initial}",
							Details: "no fields",
						},
					},
					QueuedEvents: []eventJSON{
						{
							TargetMachine: "testStateMachine",
							EventName:     "entryEvent",
							Details:       "no fields",
						},
					},
				},
			},
		},
		{
			name: "multiple state machines creating multiple worlds",
			setupModel: func() model {
				sm1 := newTestStateMachine(newTestState("state1"))
				sm2 := newTestStateMachine(newTestState("state2"))
				m, _ := newModel(WithStateMachines(sm1, sm2))
				_ = m.Solve()
				return m
			},
			expectedWorlds: []worldJSON{
				{
					InvariantViolation: false,
					StateMachines: []stateMachineJSON{
						{
							ID:      "testStateMachine",
							Name:    "testStateMachine",
							State:   "{Name:Name,Type:string,Value:state1}",
							Details: "no fields",
						},
						{
							ID:      "testStateMachine_1",
							Name:    "testStateMachine",
							State:   "{Name:Name,Type:string,Value:state2}",
							Details: "no fields",
						},
					},
					QueuedEvents: []eventJSON{},
				},
				{
					InvariantViolation: false,
					StateMachines: []stateMachineJSON{
						{
							ID:      "testStateMachine",
							Name:    "testStateMachine",
							State:   "{Name:Name,Type:string,Value:state1}",
							Details: "no fields",
						},
						{
							ID:      "testStateMachine_1",
							Name:    "testStateMachine",
							State:   "{Name:Name,Type:string,Value:state2}",
							Details: "no fields",
						},
					},
					QueuedEvents: []eventJSON{
						{
							TargetMachine: "testStateMachine",
							EventName:     "entryEvent",
							Details:       "no fields",
						},
					},
				},
				{
					InvariantViolation: false,
					StateMachines: []stateMachineJSON{
						{
							ID:      "testStateMachine",
							Name:    "testStateMachine",
							State:   "{Name:Name,Type:string,Value:state1}",
							Details: "no fields",
						},
						{
							ID:      "testStateMachine_1",
							Name:    "testStateMachine",
							State:   "{Name:Name,Type:string,Value:state2}",
							Details: "no fields",
						},
					},
					QueuedEvents: []eventJSON{
						{
							TargetMachine: "testStateMachine",
							EventName:     "entryEvent",
							Details:       "no fields",
						},
					},
				},
				{
					InvariantViolation: false,
					StateMachines: []stateMachineJSON{
						{
							ID:      "testStateMachine",
							Name:    "testStateMachine",
							State:   "{Name:Name,Type:string,Value:state1}",
							Details: "no fields",
						},
						{
							ID:      "testStateMachine_1",
							Name:    "testStateMachine",
							State:   "{Name:Name,Type:string,Value:state2}",
							Details: "no fields",
						},
					},
					QueuedEvents: []eventJSON{
						{
							TargetMachine: "testStateMachine",
							EventName:     "entryEvent",
							Details:       "no fields",
						},
						{
							TargetMachine: "testStateMachine",
							EventName:     "entryEvent",
							Details:       "no fields",
						},
					},
				},
			},
		},
		{
			name: "empty kripke structure",
			setupModel: func() model {
				m, _ := newModel()
				return m
			},
			expectedWorlds: []worldJSON{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.setupModel()
			actualWorlds := m.worldsToJSON()

			if diff := cmp.Diff(tt.expectedWorlds, actualWorlds); diff != "" {
				t.Errorf("Worlds data mismatch (-expected +actual):\n%s", diff)
			}
		})
	}
}

func TestModel_summarize(t *testing.T) {
	tests := []struct {
		name            string
		setupModel      func() model
		executionTimeMs int64
		wantSummary     *modelSummary
	}{
		{
			name: "kripke with no invariant violations",
			setupModel: func() model {
				sm := newTestStateMachine(newTestState("initial"))
				inv := BoolInvariant(true)
				m, _ := newModel(
					WithStateMachines(sm),
					WithInvariants(inv),
				)
				_ = m.Solve()
				return m
			},
			executionTimeMs: 150,
			wantSummary: &modelSummary{
				TotalWorlds:     2,
				ExecutionTimeMs: 150,
				InvariantViolations: struct {
					Found bool `json:"found"`
					Count int  `json:"count"`
				}{
					Found: false,
					Count: 0,
				},
			},
		},
		{
			name: "kripke with invariant violations",
			setupModel: func() model {
				sm := newTestStateMachine(newTestState("initial"))
				inv := BoolInvariant(false)
				m, _ := newModel(
					WithStateMachines(sm),
					WithInvariants(inv),
				)
				_ = m.Solve()
				return m
			},
			executionTimeMs: 250,
			wantSummary: &modelSummary{
				TotalWorlds:     2,
				ExecutionTimeMs: 250,
				InvariantViolations: struct {
					Found bool `json:"found"`
					Count int  `json:"count"`
				}{
					Found: true,
					Count: 2,
				},
			},
		},
		{
			name: "empty kripke structure",
			setupModel: func() model {
				m, _ := newModel()
				return m
			},
			executionTimeMs: 0,
			wantSummary: &modelSummary{
				TotalWorlds:     0,
				ExecutionTimeMs: 0,
				InvariantViolations: struct {
					Found bool `json:"found"`
					Count int  `json:"count"`
				}{
					Found: false,
					Count: 0,
				},
			},
		},
		{
			name: "multiple worlds without violations",
			setupModel: func() model {
				sm1 := newTestStateMachine(newTestState("state1"))
				sm2 := newTestStateMachine(newTestState("state2"))
				m, _ := newModel(WithStateMachines(sm1, sm2))
				_ = m.Solve()
				return m
			},
			executionTimeMs: 500,
			wantSummary: &modelSummary{
				TotalWorlds:     4,
				ExecutionTimeMs: 500,
				InvariantViolations: struct {
					Found bool `json:"found"`
					Count int  `json:"count"`
				}{
					Found: false,
					Count: 0,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.setupModel()
			summary := m.summarize(tt.executionTimeMs)

			if !cmp.Equal(summary, tt.wantSummary) {
				t.Errorf("summarize() mismatch: %v", cmp.Diff(tt.wantSummary, summary))
			}
		})
	}
}

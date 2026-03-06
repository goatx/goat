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
				inv := BoolCondition("inv", false)
				m, _ := newModel(
					WithStateMachines(sm),
					WithRules(Always(inv)),
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

func TestModel_writeInvariantViolations(t *testing.T) {
	tests := []struct {
		name  string
		setup func() model
		want  string
	}{
		{
			name: "no invariant violations",
			setup: func() model {
				sm := newTestStateMachine(newTestState("initial"))
				inv := BoolCondition("pass", true)
				m, _ := newModel(
					WithStateMachines(sm),
					WithRules(Always(inv)),
				)
				_ = m.Solve()
				return m
			},
			want: "",
		},
		{
			name: "with invariant violation",
			setup: func() model {
				sm := newTestStateMachine(newTestState("initial"))
				inv := BoolCondition("fail", false)
				m, _ := newModel(
					WithStateMachines(sm),
					WithRules(Always(inv)),
				)
				_ = m.Solve()
				return m
			},
			want: `Condition failed. Not Always fail.
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
			m.writeInvariantViolations(&buf)
			got := buf.String()

			if got != tt.want {
				t.Errorf("writeInvariantViolations() output mismatch\ngot:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestModel_writeTemporalViolations(t *testing.T) {
	sm := newTestStateMachine(newTestState("s"))
	cFalse := BoolCondition("c", false)
	m, err := newModel(
		WithStateMachines(sm),
		WithRules(EventuallyAlways(cFalse)),
	)
	if err != nil {
		t.Fatalf("newModel error: %v", err)
	}
	_ = m.Solve()

	results := m.checkLTL()
	var buf bytes.Buffer
	m.writeTemporalViolations(&buf, results)
	got := buf.String()

	want := `Condition failed. Not eventually always c.
Violation path (length = 2):
  [0]
  StateMachines:
    Name: testStateMachine, Detail: no fields, State: {Name:Name,Type:string,Value:s}
  QueuedEvents:
    StateMachine: testStateMachine, Event: entryEvent, Detail: no fields
  [1]
  StateMachines:
    Name: testStateMachine, Detail: no fields, State: {Name:Name,Type:string,Value:s}
  QueuedEvents:
`

	if got != want {
		t.Errorf("writeTemporalViolations() output mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestModel_collectInvariantViolations(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() model
		expected []invariantViolationWitness
	}{
		{
			name: "deduplicates paths per invariant",
			setup: func() model {
				initial := world{id: 1}
				violationA := world{id: 2, failedInvariants: []ConditionName{"dup"}}
				violationB := world{id: 3, failedInvariants: []ConditionName{"dup"}}

				return model{
					initial: initial,
					worlds: worlds{
						1: initial,
						2: violationA,
						3: violationB,
					},
					accessible: map[worldID][]worldID{
						1: {2, 3},
						2: nil,
						3: nil,
					},
				}
			},
			expected: []invariantViolationWitness{
				{
					path:      []worldID{1, 2},
					condition: "dup",
				},
			},
		},
		{
			name: "finds deeper violations after early failure",
			setup: func() model {
				initial := world{id: 1}
				first := world{id: 2, failedInvariants: []ConditionName{"first"}}
				second := world{id: 3, failedInvariants: []ConditionName{"second"}}

				return model{
					initial: initial,
					worlds: worlds{
						1: initial,
						2: first,
						3: second,
					},
					accessible: map[worldID][]worldID{
						1: {2},
						2: {3},
						3: nil,
					},
				}
			},
			expected: []invariantViolationWitness{
				{
					path:      []worldID{1, 2},
					condition: "first",
				},
				{
					path:      []worldID{1, 2, 3},
					condition: "second",
				},
			},
		},
		{
			name: "no violations",
			setup: func() model {
				sm := newTestStateMachine(newTestState("initial"))
				inv := BoolCondition("pass", true)
				m, err := newModel(
					WithStateMachines(sm),
					WithRules(Always(inv)),
				)
				if err != nil {
					panic(err)
				}
				_ = m.Solve()
				return m
			},
			expected: nil,
		},
		{
			name: "single violation",
			setup: func() model {
				sm := newTestStateMachine(newTestState("initial"))
				inv := BoolCondition("fail", false)
				m, err := newModel(
					WithStateMachines(sm),
					WithRules(Always(inv)),
				)
				if err != nil {
					panic(err)
				}
				_ = m.Solve()
				return m
			},
			expected: []invariantViolationWitness{
				{
					path:      []worldID{8682599965454615616},
					condition: "fail",
				},
			},
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

				inv := NewCondition("count<=1", sm, func(sm *testCounter) bool {
					return sm.count <= 1
				})

				m, err := newModel(
					WithStateMachines(sm),
					WithRules(Always(inv)),
				)
				if err != nil {
					panic(err)
				}
				_ = m.Solve()
				return m
			},
			expected: []invariantViolationWitness{
				{
					path: []worldID{
						5790322525083387874,
						15591947093441390666,
						10703074720578030081,
						15159594575768829045,
						8395799135532667686,
					},
					condition: "count<=1",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.setup()
			actual := m.collectInvariantViolations()

			if diff := cmp.Diff(tt.expected, actual, cmp.AllowUnexported(invariantViolationWitness{})); diff != "" {
				t.Errorf("Invariant violations mismatch (-expected +actual):\n%s", diff)
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
				inv := BoolCondition("pass", true)
				m, _ := newModel(
					WithStateMachines(sm),
					WithRules(Always(inv)),
				)
				_ = m.Solve()
				return m
			},
			executionTimeMs: 150,
			wantSummary: &modelSummary{
				TotalWorlds:     2,
				ExecutionTimeMs: 150,
			},
		},
		{
			name: "kripke with invariant violations",
			setupModel: func() model {
				sm := newTestStateMachine(newTestState("initial"))
				inv := BoolCondition("fail", false)
				m, _ := newModel(
					WithStateMachines(sm),
					WithRules(Always(inv)),
				)
				_ = m.Solve()
				return m
			},
			executionTimeMs: 250,
			wantSummary: &modelSummary{
				TotalWorlds:     2,
				ExecutionTimeMs: 250,
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
			},
		},
		{
			name: "multiple worlds",
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

type smWithGoatPointer struct {
	StateMachine
	Other *testStateMachine
}

type smWithDomainPointer struct {
	StateMachine
	Data *int
}

type smWithUnexportedPointer struct {
	StateMachine
	data *int //nolint:unused // field existence is the test target
}

type stateWithPointer struct {
	State
	Data *int
}

type eventWithPointer struct {
	Event[*testStateMachine, *testStateMachine]
	Data *int
}

type myDomainStruct struct {
	Value int
}

type smWithDomainStructPointer struct {
	StateMachine
	Info *myDomainStruct
}

type innerWithPointer struct {
	Ptr *int
}

type smWithNestedPointer struct {
	StateMachine
	Inner innerWithPointer
}

func TestWarnShallowPointerFields(t *testing.T) {
	tests := []struct {
		name  string
		setup func() []AbstractStateMachine
		want  string
	}{
		{
			name: "AbstractStateMachine pointer field is allowed",
			setup: func() []AbstractStateMachine {
				spec := NewStateMachineSpec(&smWithGoatPointer{})
				spec.DefineStates(newTestState("s")).SetInitialState(newTestState("s"))
				sm, err := spec.NewInstance()
				if err != nil {
					panic(err)
				}
				return []AbstractStateMachine{sm}
			},
			want: "",
		},
		{
			name: "domain pointer field warns",
			setup: func() []AbstractStateMachine {
				spec := NewStateMachineSpec(&smWithDomainPointer{})
				spec.DefineStates(newTestState("s")).SetInitialState(newTestState("s"))
				sm, err := spec.NewInstance()
				if err != nil {
					panic(err)
				}
				return []AbstractStateMachine{sm}
			},
			want: "WARNING: type \"smWithDomainPointer\" has pointer field \"Data\" (*int) which will be shared between states during model checking, potentially causing incorrect results. Consider using a value type instead.\n",
		},
		{
			name: "unexported pointer field does not warn",
			setup: func() []AbstractStateMachine {
				spec := NewStateMachineSpec(&smWithUnexportedPointer{})
				spec.DefineStates(newTestState("s")).SetInitialState(newTestState("s"))
				sm, err := spec.NewInstance()
				if err != nil {
					panic(err)
				}
				return []AbstractStateMachine{sm}
			},
			want: "",
		},
		{
			name: "same type checked only once",
			setup: func() []AbstractStateMachine {
				spec := NewStateMachineSpec(&smWithDomainPointer{})
				spec.DefineStates(newTestState("s")).SetInitialState(newTestState("s"))
				sm1, err := spec.NewInstance()
				if err != nil {
					panic(err)
				}
				sm2, err := spec.NewInstance()
				if err != nil {
					panic(err)
				}
				return []AbstractStateMachine{sm1, sm2}
			},
			want: "WARNING: type \"smWithDomainPointer\" has pointer field \"Data\" (*int) which will be shared between states during model checking, potentially causing incorrect results. Consider using a value type instead.\n",
		},
		{
			name: "state pointer field warns",
			setup: func() []AbstractStateMachine {
				spec := NewStateMachineSpec(&testStateMachine{})
				s := &stateWithPointer{}
				spec.DefineStates(s).SetInitialState(s)
				sm, err := spec.NewInstance()
				if err != nil {
					panic(err)
				}
				return []AbstractStateMachine{sm}
			},
			want: "WARNING: type \"stateWithPointer\" has pointer field \"Data\" (*int) which will be shared between states during model checking, potentially causing incorrect results. Consider using a value type instead.\n",
		},
		{
			name: "event pointer field warns",
			setup: func() []AbstractStateMachine {
				spec := NewStateMachineSpec(&testStateMachine{})
				s := newTestState("s")
				spec.DefineStates(s).SetInitialState(s)
				OnEvent[*eventWithPointer](spec, s, func(ctx context.Context, event *eventWithPointer, sm *testStateMachine) {})
				sm, err := spec.NewInstance()
				if err != nil {
					panic(err)
				}
				return []AbstractStateMachine{sm}
			},
			want: "WARNING: type \"eventWithPointer\" has pointer field \"Data\" (*int) which will be shared between states during model checking, potentially causing incorrect results. Consider using a value type instead.\n",
		},
		{
			name: "domain struct pointer field warns",
			setup: func() []AbstractStateMachine {
				spec := NewStateMachineSpec(&smWithDomainStructPointer{})
				spec.DefineStates(newTestState("s")).SetInitialState(newTestState("s"))
				sm, err := spec.NewInstance()
				if err != nil {
					panic(err)
				}
				return []AbstractStateMachine{sm}
			},
			want: "WARNING: type \"smWithDomainStructPointer\" has pointer field \"Info\" (*goat.myDomainStruct) which will be shared between states during model checking, potentially causing incorrect results. Consider using a value type instead.\n",
		},
		{
			name: "nested struct with pointer field warns",
			setup: func() []AbstractStateMachine {
				spec := NewStateMachineSpec(&smWithNestedPointer{})
				spec.DefineStates(newTestState("s")).SetInitialState(newTestState("s"))
				sm, err := spec.NewInstance()
				if err != nil {
					panic(err)
				}
				return []AbstractStateMachine{sm}
			},
			want: "WARNING: type \"innerWithPointer\" has pointer field \"Ptr\" (*int) which will be shared between states during model checking, potentially causing incorrect results. Consider using a value type instead.\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			warnShallowPointerFields(&buf, tt.setup())
			got := buf.String()

			if got != tt.want {
				t.Errorf("warnShallowPointerFields() output mismatch\ngot:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

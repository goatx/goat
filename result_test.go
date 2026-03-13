package goat

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestTest(t *testing.T) {
	sm := StateMachineSnapshot{Name: "testStateMachine", State: "{Name:Name,Type:string,Value:s}", Details: "no fields"}
	entry := EventSnapshot{TargetMachine: "testStateMachine", EventName: "entryEvent", Details: "no fields"}

	tests := []struct {
		name    string
		opts    []Option
		wantErr bool
		want    *Result
	}{
		{
			name: "no violation",
			opts: func() []Option {
				sm := newTestStateMachine(newTestState("s"))
				return []Option{WithStateMachines(sm), WithRules(Always(BoolCondition("ok", true)))}
			}(),
			want: &Result{
				Summary: Summary{TotalWorlds: 2},
			},
		},
		{
			name: "invariant violation",
			opts: func() []Option {
				sm := newTestStateMachine(newTestState("s"))
				return []Option{WithStateMachines(sm), WithRules(Always(BoolCondition("bad", false)))}
			}(),
			want: &Result{
				Violations: []Violation{
					{
						Rule: "Always bad",
						Path: []WorldSnapshot{
							{
								StateMachines: []StateMachineSnapshot{sm},
								QueuedEvents:  []EventSnapshot{entry},
							},
						},
					},
				},
				Summary: Summary{TotalWorlds: 2},
			},
		},
		{
			name: "temporal violation",
			opts: func() []Option {
				sm := newTestStateMachine(newTestState("s"))
				return []Option{WithStateMachines(sm), WithRules(EventuallyAlways(BoolCondition("cF", false)))}
			}(),
			want: &Result{
				Violations: []Violation{
					{
						Rule: "eventually always cF",
						Path: []WorldSnapshot{
							{
								StateMachines: []StateMachineSnapshot{sm},
								QueuedEvents:  []EventSnapshot{entry},
							},
							{
								StateMachines: []StateMachineSnapshot{sm},
								QueuedEvents:  []EventSnapshot{},
							},
						},
						Loop: []WorldSnapshot{
							{
								StateMachines: []StateMachineSnapshot{sm},
								QueuedEvents:  []EventSnapshot{},
							},
						},
					},
				},
				Summary: Summary{TotalWorlds: 2},
			},
		},
		{
			name:    "error on empty options",
			wantErr: true,
		},
	}

	cmpOpts := cmp.Options{
		cmpopts.IgnoreFields(Summary{}, "ExecutionTimeMs"),
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Test(tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Test() error = %v, wantErr %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(tt.want, result, cmpOpts...); diff != "" {
				t.Errorf("result mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

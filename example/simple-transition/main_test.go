package main

import (
	"testing"

	"github.com/goatx/goat"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestSimpleTransition(t *testing.T) {
	opts := createSimpleTransitionModel()

	result, err := goat.Test(opts...)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	expected := &goat.Result{
		Violations: []goat.Violation{
			{
				Rule: "Always mut<=1",
				Path: []goat.WorldSnapshot{
					{
						StateMachines: []goat.StateMachineSnapshot{
							{Name: "StateMachine", State: "{Name:StateType,Type:main.StateType,Value:A}", Details: "{Name:Mut,Type:int,Value:0}"},
						},
						QueuedEvents: []goat.EventSnapshot{
							{TargetMachine: "StateMachine", EventName: "entryEvent", Details: "no fields"},
						},
					},
					{
						StateMachines: []goat.StateMachineSnapshot{
							{Name: "StateMachine", State: "{Name:StateType,Type:main.StateType,Value:A}", Details: "{Name:Mut,Type:int,Value:1}"},
						},
						QueuedEvents: []goat.EventSnapshot{
							{TargetMachine: "StateMachine", EventName: "exitEvent", Details: "no fields"},
							{TargetMachine: "StateMachine", EventName: "transitionEvent", Details: "{Name:To,Type:goat.AbstractState,Value:&{{0} B}}"},
							{TargetMachine: "StateMachine", EventName: "entryEvent", Details: "no fields"},
						},
					},
					{
						StateMachines: []goat.StateMachineSnapshot{
							{Name: "StateMachine", State: "{Name:StateType,Type:main.StateType,Value:A}", Details: "{Name:Mut,Type:int,Value:1}"},
						},
						QueuedEvents: []goat.EventSnapshot{
							{TargetMachine: "StateMachine", EventName: "transitionEvent", Details: "{Name:To,Type:goat.AbstractState,Value:&{{0} B}}"},
							{TargetMachine: "StateMachine", EventName: "entryEvent", Details: "no fields"},
						},
					},
					{
						StateMachines: []goat.StateMachineSnapshot{
							{Name: "StateMachine", State: "{Name:StateType,Type:main.StateType,Value:B}", Details: "{Name:Mut,Type:int,Value:1}"},
						},
						QueuedEvents: []goat.EventSnapshot{
							{TargetMachine: "StateMachine", EventName: "entryEvent", Details: "no fields"},
						},
					},
					{
						StateMachines: []goat.StateMachineSnapshot{
							{Name: "StateMachine", State: "{Name:StateType,Type:main.StateType,Value:B}", Details: "{Name:Mut,Type:int,Value:2}"},
						},
						QueuedEvents: []goat.EventSnapshot{
							{TargetMachine: "StateMachine", EventName: "exitEvent", Details: "no fields"},
							{TargetMachine: "StateMachine", EventName: "transitionEvent", Details: "{Name:To,Type:goat.AbstractState,Value:&{{0} C}}"},
							{TargetMachine: "StateMachine", EventName: "entryEvent", Details: "no fields"},
						},
					},
				},
			},
		},
		Summary: goat.Summary{TotalWorlds: 8},
	}

	cmpOpts := cmp.Options{
		cmpopts.IgnoreFields(goat.Summary{}, "ExecutionTimeMs"),
	}
	if diff := cmp.Diff(expected, result, cmpOpts...); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}

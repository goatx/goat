package main

import (
	"testing"

	"github.com/goatx/goat"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestTemporalRuleViolationExample(t *testing.T) {
	opts := createTemporalRuleViolationModel()

	result, err := goat.Test(opts...)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	expected := &goat.Result{
		Violations: []goat.Violation{
			{
				Rule: "whenever inPaid eventually inShipped",
				Path: []goat.WorldSnapshot{
					{
						StateMachines: []goat.StateMachineSnapshot{
							{Name: "FailingShipper", State: "no fields", Details: "no fields"},
							{Name: "Order", State: "{Name:StateType,Type:main.StateType,Value:Pending}", Details: "no fields"},
						},
						QueuedEvents: []goat.EventSnapshot{
							{TargetMachine: "FailingShipper", EventName: "entryEvent", Details: "no fields"},
							{TargetMachine: "Order", EventName: "entryEvent", Details: "no fields"},
						},
					},
					{
						StateMachines: []goat.StateMachineSnapshot{
							{Name: "FailingShipper", State: "no fields", Details: "no fields"},
							{Name: "Order", State: "{Name:StateType,Type:main.StateType,Value:Pending}", Details: "no fields"},
						},
						QueuedEvents: []goat.EventSnapshot{
							{TargetMachine: "Order", EventName: "entryEvent", Details: "no fields"},
						},
					},
					{
						StateMachines: []goat.StateMachineSnapshot{
							{Name: "FailingShipper", State: "no fields", Details: "no fields"},
							{Name: "Order", State: "{Name:StateType,Type:main.StateType,Value:Pending}", Details: "no fields"},
						},
						QueuedEvents: []goat.EventSnapshot{
							{TargetMachine: "Order", EventName: "exitEvent", Details: "no fields"},
							{TargetMachine: "Order", EventName: "transitionEvent", Details: "{Name:To,Type:goat.AbstractState,Value:&{{0} Paid}}"},
							{TargetMachine: "Order", EventName: "entryEvent", Details: "no fields"},
						},
					},
					{
						StateMachines: []goat.StateMachineSnapshot{
							{Name: "FailingShipper", State: "no fields", Details: "no fields"},
							{Name: "Order", State: "{Name:StateType,Type:main.StateType,Value:Pending}", Details: "no fields"},
						},
						QueuedEvents: []goat.EventSnapshot{
							{TargetMachine: "Order", EventName: "transitionEvent", Details: "{Name:To,Type:goat.AbstractState,Value:&{{0} Paid}}"},
							{TargetMachine: "Order", EventName: "entryEvent", Details: "no fields"},
						},
					},
					{
						StateMachines: []goat.StateMachineSnapshot{
							{Name: "FailingShipper", State: "no fields", Details: "no fields"},
							{Name: "Order", State: "{Name:StateType,Type:main.StateType,Value:Paid}", Details: "no fields"},
						},
						QueuedEvents: []goat.EventSnapshot{
							{TargetMachine: "Order", EventName: "entryEvent", Details: "no fields"},
						},
					},
					{
						StateMachines: []goat.StateMachineSnapshot{
							{Name: "FailingShipper", State: "no fields", Details: "no fields"},
							{Name: "Order", State: "{Name:StateType,Type:main.StateType,Value:Paid}", Details: "no fields"},
						},
						QueuedEvents: []goat.EventSnapshot{
							{TargetMachine: "FailingShipper", EventName: "eShipRequest", Details: "no fields"},
						},
					},
					{
						StateMachines: []goat.StateMachineSnapshot{
							{Name: "FailingShipper", State: "no fields", Details: "no fields"},
							{Name: "Order", State: "{Name:StateType,Type:main.StateType,Value:Paid}", Details: "no fields"},
						},
						QueuedEvents: []goat.EventSnapshot{},
					},
				},
				Loop: []goat.WorldSnapshot{
					{
						StateMachines: []goat.StateMachineSnapshot{
							{Name: "FailingShipper", State: "no fields", Details: "no fields"},
							{Name: "Order", State: "{Name:StateType,Type:main.StateType,Value:Paid}", Details: "no fields"},
						},
						QueuedEvents: []goat.EventSnapshot{},
					},
				},
			},
		},
		Summary: goat.Summary{TotalWorlds: 11},
	}

	cmpOpts := cmp.Options{
		cmpopts.IgnoreFields(goat.Summary{}, "ExecutionTimeMs"),
	}
	if diff := cmp.Diff(expected, result, cmpOpts...); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}

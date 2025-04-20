package main

import (
	"context"
	"log"

	"github.com/goatx/goat"
)

type (
	StateType string
)

const (
	StateA StateType = "A"
	StateB StateType = "B"
)

type (
	State struct {
		goat.State
		StateType StateType
	}
)

type StateMachine struct {
	goat.StateMachine
}

func createSimpleHaltModel() []goat.Option {
	// === StateMachine Spec ===
	spec := goat.NewStateMachineSpec(&StateMachine{})
	stateA := &State{StateType: StateA}
	stateB := &State{StateType: StateB}

	spec.DefineStates(stateA, stateB).SetInitialState(stateA)

	goat.OnEntry(spec, stateA, func(ctx context.Context, machine *StateMachine) {
		goat.Halt(ctx, machine)
		// no longer reachable since the state machine is halted.
		goat.Goto(ctx, stateB)
	})

	goat.OnEntry(spec, stateB, func(ctx context.Context, machine *StateMachine) {
		goat.Goto(ctx, stateA)
	})

	// === Create Instance ===
	sm, err := spec.NewInstance()
	if err != nil {
		log.Fatal(err)
	}

	opts := []goat.Option{
		goat.WithStateMachines(sm),
	}

	return opts
}

func main() {
	opts := createSimpleHaltModel()

	err := goat.Test(opts...)
	if err != nil {
		panic(err)
	}
}

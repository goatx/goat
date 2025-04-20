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
	StateC StateType = "C"
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

func createSimpleNonDeterministicModel() []goat.Option {
	// === StateMachine Spec ===
	spec := goat.NewStateMachineSpec(&StateMachine{})

	stateA := &State{StateType: StateA}
	stateB := &State{StateType: StateB}
	stateC := &State{StateType: StateC}

	spec.DefineStates(stateA, stateB, stateC).SetInitialState(stateA)

	// Processing in state A
	// When transitioning to state A, execute the following non-deterministically:
	// 1. Transition to state B
	// 2. Transition to state C
	goat.OnEntry(spec, stateA, func(ctx context.Context, machine *StateMachine) {
		goat.Goto(ctx, stateB)
	})
	goat.OnEntry(spec, stateA, func(ctx context.Context, machine *StateMachine) {
		goat.Goto(ctx, stateC)
	})

	// Processing in state B
	// When transitioning to state B, execute the following non-deterministically:
	// 1. Transition to state C
	// 2. Transition to state A
	goat.OnEntry(spec, stateB, func(ctx context.Context, machine *StateMachine) {
		goat.Goto(ctx, stateC)
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
	opts := createSimpleNonDeterministicModel()

	err := goat.Test(opts...)
	if err != nil {
		panic(err)
	}
}

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
	Mut int
}

func createSimpleTransitionModel() []goat.Option {
	// === StateMachine Spec ===
	spec := goat.NewStateMachineSpec(&StateMachine{})
	stateA := &State{StateType: StateA}
	stateB := &State{StateType: StateB}
	stateC := &State{StateType: StateC}

	spec.DefineStates(stateA, stateB, stateC).SetInitialState(stateA)

	goat.OnEntry(spec, stateA, func(ctx context.Context, machine *StateMachine) {
		machine.Mut = 1
		goat.Goto(ctx, stateB)
	})

	goat.OnEntry(spec, stateB, func(ctx context.Context, machine *StateMachine) {
		machine.Mut = 2
		goat.Goto(ctx, stateC)
	})

	goat.OnEntry(spec, stateC, func(ctx context.Context, machine *StateMachine) {
		machine.Mut = 3
	})

	// === Create Instance ===
	sm, err := spec.NewInstance()
	if err != nil {
		log.Fatal(err)
	}

	cond := goat.NewCondition("mut<=1", sm, func(sm *StateMachine) bool {
		return sm.Mut <= 1
	})

	opts := []goat.Option{
		goat.WithStateMachines(sm),
		goat.WithRules(goat.Always(cond)),
	}

	return opts
}

func main() {
	opts := createSimpleTransitionModel()

	_, err := goat.Test(opts...)
	if err != nil {
		panic(err)
	}
}

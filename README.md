# goat

A Go library for model checking concurrent systems using state machines. goat helps you verify the correctness of distributed systems by exhaustively exploring all possible states and checking invariants.

## Installation

```bash
go get github.com/goatx/goat
```

Requires Go 1.24.5 or later.

## Quick Example

```go
package main

import (
    "context"
    "github.com/goatx/goat"
)

type State struct {
    goat.State
    Name string
}

type MyStateMachine struct {
    goat.StateMachine
    Counter int
}

func main() {
    // Create state machine specification
    spec := goat.NewStateMachineSpec(&MyStateMachine{})
    stateA := &State{Name: "A"}
    stateB := &State{Name: "B"}

    // Define the states that the StateMachine can take and set initial state
    spec.
        DefineStates(stateA, stateB).
        SetInitialState(stateA)

    // Define behavior
    goat.OnEntry(spec, stateA, func(ctx context.Context, sm *MyStateMachine) {
        sm.Counter = 1
        goat.Goto(ctx, stateB)
    })

    goat.OnEntry(spec, stateB, func(ctx context.Context, sm *MyStateMachine) {
        sm.Counter = 2
    })

    // Create instance and run model checking
    sm, err := spec.NewInstance()
    if err != nil {
        log.Fatal(err)
    }
    err = goat.Test(
        goat.WithStateMachines(sm),
        goat.WithInvariants(
            goat.NewInvariant(sm, func(sm *MyStateMachine) bool {
                return sm.Counter <= 2
            }),
        ),
    )
    if err != nil {
        panic(err)
    }
}
```

## Key Functions

- **`NewStateMachineSpec()`** - Create a state machine specification
- **`OnEntry()`, `OnEvent()`, `OnExit()`** - Register event handlers for each lifecycle events
- **`Goto()`** - Trigger state transitions
- **`SendTo()`** - Send events between state machines
- **`Test()`** - Run model checking with invariant verification
- **`Debug()`** - Output detailed JSON results for debugging
- **`WithStateMachines()`** - Configure which state machines to test
- **`WithInvariants()`** - Configure invariants to check

## Examples

The `example/` directory contains several complete examples:

- **`simple-transition/`** - Basic state transitions and invariants
- **`client-server/`** - Distributed communication patterns
- **`meeting-room-reservation/`** - Resource contention scenarios
- **`simple-halt/`** - State machine termination
- **`simple-non-deterministic/`** - Non-deterministic behavior modeling

Run any example:

```bash
go run ./example/simple-transition
```

package goat

import (
	"context"
	"fmt"
)

// EventHandler is a function type for handling generic events in a state machine.
// It receives the event, the state machine instance, and a context for
// interacting with other state machines via SendTo, Goto, or Halt functions.
type EventHandler[T AbstractEvent, SM AbstractStateMachine] func(ctx context.Context, event T, sm SM)

// EntryHandler is a function type for handling state entry events.
// It is called when a state machine enters a new state.
type EntryHandler[SM AbstractStateMachine] func(ctx context.Context, sm SM)

// ExitHandler is a function type for handling state exit events.
// It is called when a state machine exits its current state.
type ExitHandler[SM AbstractStateMachine] func(ctx context.Context, sm SM)

// TransitionHandler is a function type for handling state transitions.
// It receives information about the target state and the state machine.
type TransitionHandler[SM AbstractStateMachine] func(ctx context.Context, toState AbstractState, sm SM)

// HaltHandler is a function type for handling halt events.
// It is called when a state machine is about to stop permanently.
type HaltHandler[SM AbstractStateMachine] func(ctx context.Context, sm SM)

type handler interface {
	handle(env environment, smID string, event AbstractEvent) ([]localState, error)
}

// OnEvent registers an event handler that defines how a state machine responds
// to a specific event when in a particular state. This is the primary way to
// specify the behavior and reactions of your state machine to events.
//
// Parameters:
//   - spec: The state machine specification to register the handler with
//   - state: The state in which this handler should be active
//   - event: The specific event instance to handle
//   - fn: The function to call when the event occurs
//
// Example:
//
//	goat.OnEvent(spec, IdleState{}, Event{Name: "START"}, func(ctx context.Context, event Event, sm *MyStateMachine) {
//	    // Handle start event
//	    goat.Goto(ctx, &ActiveState{})
//	})
func OnEvent[T AbstractEvent, SM AbstractStateMachine](
	spec *StateMachineSpec[SM],
	state AbstractState,
	event T,
	fn EventHandler[T, SM],
) {
	builder := func(smID string) handler {
		return &eventHandlers{
			fs:    []eventHandler{handleEvent[T, SM](smID, fn)},
			event: event,
		}
	}

	spec.handlerBuilders[state] = append(spec.handlerBuilders[state], handlerBuilderInfo{
		event:   event,
		builder: builder,
	})
}

// OnEntry registers an entry handler that defines what actions the state machine
// should perform when entering a specific state. This allows you to specify
// initialization logic, setup operations, or state-specific preparations.
//
// Parameters:
//   - spec: The state machine specification to register the handler with
//   - state: The state for which to register the entry handler
//   - fn: The function to call when entering the state
//
// Example:
//
//	goat.OnEntry(spec, ActiveState{}, func(ctx context.Context, sm *MyStateMachine) {
//	    // Perform initialization when entering active state
//	    sm.StartTime = time.Now()
//	})
func OnEntry[SM AbstractStateMachine](
	spec *StateMachineSpec[SM],
	state AbstractState,
	fn EntryHandler[SM],
) {
	builder := func(smID string) handler {
		return &entryHandlers{
			fs: []entryHandler{handleEntry[SM](smID, fn)},
		}
	}

	spec.handlerBuilders[state] = append(spec.handlerBuilders[state], handlerBuilderInfo{
		event:   &entryEvent{},
		builder: builder,
	})
}

// OnExit registers an exit handler that defines what cleanup or finalization
// actions the state machine should perform when leaving a specific state.
// This is essential for proper resource management and state transitions.
//
// Parameters:
//   - spec: The state machine specification to register the handler with
//   - state: The state for which to register the exit handler
//   - fn: The function to call when exiting the state
//
// Example:
//
//	goat.OnExit(spec, ActiveState{}, func(ctx context.Context, sm *MyStateMachine) {
//	    // Perform cleanup when exiting active state
//	    sm.cleanup()
//	})
func OnExit[SM AbstractStateMachine](
	spec *StateMachineSpec[SM],
	state AbstractState,
	fn ExitHandler[SM],
) {
	builder := func(smID string) handler {
		return &exitHandlers{
			fs: []exitHandler{handleExit[SM](smID, fn)},
		}
	}

	spec.handlerBuilders[state] = append(spec.handlerBuilders[state], handlerBuilderInfo{
		event:   &exitEvent{},
		builder: builder,
	})
}

// OnTransition registers a transition handler that defines what actions should
// occur during state transitions from a specific state. This allows you to
// specify logic that runs between exiting one state and entering another.
//
// Parameters:
//   - spec: The state machine specification to register the handler with
//   - state: The source state for which to register the transition handler
//   - fn: The function to call during transition from this state
//
// Example:
//
//	goat.OnTransition(spec, IdleState{}, func(ctx context.Context, toState AbstractState, sm *MyStateMachine) {
//	    // Log transition from idle to any other state
//	    log.Printf("Transitioning from Idle to %T", toState)
//	})
func OnTransition[SM AbstractStateMachine](
	spec *StateMachineSpec[SM],
	state AbstractState,
	fn TransitionHandler[SM],
) {
	builder := func(smID string) handler {
		return &transitionHandlers{
			fs: []transitionHandler{handleTransition[SM](smID, fn)},
		}
	}

	spec.handlerBuilders[state] = append(spec.handlerBuilders[state], handlerBuilderInfo{
		event:   &transitionEvent{},
		builder: builder,
	})
}

// OnHalt registers a halt handler that defines what final cleanup or shutdown
// actions the state machine should perform when stopping execution while in
// a specific state. This ensures proper termination and resource cleanup.
//
// Parameters:
//   - spec: The state machine specification to register the handler with
//   - state: The state for which to register the halt handler
//   - fn: The function to call when the state machine halts
//
// Example:
//
//	goat.OnHalt(spec, ActiveState{}, func(ctx context.Context, sm *MyStateMachine) {
//	    // Perform final cleanup before halting
//	    sm.saveState()
//	})
func OnHalt[SM AbstractStateMachine](
	spec *StateMachineSpec[SM],
	state AbstractState,
	fn HaltHandler[SM],
) {
	builder := func(smID string) handler {
		return &haltHandlers{
			fs: []haltHandler{handleHalt[SM](smID, fn)},
		}
	}

	spec.handlerBuilders[state] = append(spec.handlerBuilders[state], handlerBuilderInfo{
		event:   &haltEvent{},
		builder: builder,
	})
}

func handleEvent[T AbstractEvent, SM AbstractStateMachine](smID string, fn EventHandler[T, SM]) eventHandler {
	return func(event AbstractEvent, env *environment) {
		typedEvent := event.(T)

		machine, exists := env.machines[smID]
		if !exists {
			panic(fmt.Sprintf("StateMachine with ID %s not found in environment", smID))
		}

		sm := machine.(SM)
		ctx := withEnvAndSM(env, sm)

		fn(ctx, typedEvent, sm)
	}
}

func handleEntry[SM AbstractStateMachine](smID string, fn EntryHandler[SM]) entryHandler {
	return func(env *environment) {
		machine, exists := env.machines[smID]
		if !exists {
			panic(fmt.Sprintf("StateMachine with ID %s not found in environment", smID))
		}

		sm := machine.(SM)
		ctx := withEnvAndSM(env, sm)

		fn(ctx, sm)
	}
}

func handleExit[SM AbstractStateMachine](smID string, fn ExitHandler[SM]) exitHandler {
	return func(env *environment) {
		machine, exists := env.machines[smID]
		if !exists {
			panic(fmt.Sprintf("StateMachine with ID %s not found in environment", smID))
		}

		sm := machine.(SM)
		ctx := withEnvAndSM(env, sm)

		fn(ctx, sm)
	}
}

func handleTransition[SM AbstractStateMachine](smID string, fn TransitionHandler[SM]) transitionHandler {
	return func(toState AbstractState, env *environment) {
		machine, exists := env.machines[smID]
		if !exists {
			panic(fmt.Sprintf("StateMachine with ID %s not found in environment", smID))
		}

		sm := machine.(SM)
		ctx := withEnvAndSM(env, sm)

		fn(ctx, toState, sm)
	}
}

func handleHalt[SM AbstractStateMachine](smID string, fn HaltHandler[SM]) haltHandler {
	return func(env *environment) {
		machine, exists := env.machines[smID]
		if !exists {
			panic(fmt.Sprintf("StateMachine with ID %s not found in environment", smID))
		}

		sm := machine.(SM)
		ctx := withEnvAndSM(env, sm)

		fn(ctx, sm)
	}
}

type handlerInfo struct {
	event   AbstractEvent
	handler handler
}

type entryHandler func(env *environment)
type exitHandler func(env *environment)
type eventHandler func(event AbstractEvent, env *environment)
type transitionHandler func(toState AbstractState, env *environment)
type haltHandler func(env *environment)

type entryHandlers struct {
	fs []entryHandler
}

func (h *entryHandlers) handle(env environment, _ string, _ AbstractEvent) ([]localState, error) {
	lss := make([]localState, 0)
	for _, f := range h.fs {
		ec := env.clone()
		f(&ec)
		lss = append(lss, localState{env: ec})
	}
	return lss, nil
}

type exitHandlers struct {
	fs []exitHandler
}

func (h *exitHandlers) handle(env environment, _ string, _ AbstractEvent) ([]localState, error) {
	lss := make([]localState, 0)
	for _, f := range h.fs {
		ec := env.clone()
		f(&ec)
		lss = append(lss, localState{env: ec})
	}
	return lss, nil
}

type eventHandlers struct {
	fs    []eventHandler
	event AbstractEvent
}

func (h *eventHandlers) handle(env environment, _ string, event AbstractEvent) ([]localState, error) {
	if !sameEvent(h.event, event) {
		return nil, nil
	}
	lss := make([]localState, 0)
	for _, f := range h.fs {
		ec := env.clone()
		f(event, &ec)
		lss = append(lss, localState{env: ec})
	}
	return lss, nil
}

type transitionHandlers struct {
	fs []transitionHandler
}

func (h *transitionHandlers) handle(env environment, smID string, event AbstractEvent) ([]localState, error) {
	if !sameEvent(&transitionEvent{}, event) {
		return nil, nil
	}
	lss := make([]localState, 0)
	for _, f := range h.fs {
		ec := env.clone()
		sm := ec.machines[smID]
		f(event.(*transitionEvent).To, &ec)
		sm.setCurrentState(event.(*transitionEvent).To)
		lss = append(lss, localState{env: ec})
	}
	return lss, nil
}

type haltHandlers struct {
	fs []haltHandler
}

func (h *haltHandlers) handle(env environment, smID string, event AbstractEvent) ([]localState, error) {
	if !sameEvent(&haltEvent{}, event) {
		return nil, nil
	}
	lss := make([]localState, 0)
	for _, f := range h.fs {
		ec := env.clone()
		sm := ec.machines[smID]
		f(&ec)
		innerSm := getInnerStateMachine(sm)
		innerSm.halted = true
		lss = append(lss, localState{env: ec})
	}
	return lss, nil
}

type defaultOnTransitionHandler struct{}

//nolint:unparam // error return required by interface
func (*defaultOnTransitionHandler) handle(env environment, smID string, event AbstractEvent) ([]localState, error) {
	if !sameEvent(&transitionEvent{}, event) {
		return nil, nil
	}
	ec := env.clone()
	sm := ec.machines[smID]
	sm.setCurrentState(event.(*transitionEvent).To)
	return []localState{{env: ec}}, nil
}

type defaultOnHaltHandler struct{}

//nolint:unparam // error return required by interface
func (*defaultOnHaltHandler) handle(env environment, smID string, event AbstractEvent) ([]localState, error) {
	if !sameEvent(&haltEvent{}, event) {
		return nil, nil
	}
	ec := env.clone()
	sm := ec.machines[smID]
	innerSm := getInnerStateMachine(sm)
	innerSm.halted = true
	return []localState{{env: ec}}, nil
}

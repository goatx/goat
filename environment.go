package goat

import "context"

type environment struct {
	machines map[string]AbstractStateMachine
	queue    map[string][]AbstractEvent
}

type (
	envKey struct{}
	smKey  struct{}
)

func withEnvAndSM(env *environment, sm AbstractStateMachine) context.Context {
	ctx := context.WithValue(context.Background(), envKey{}, env)
	ctx = context.WithValue(ctx, smKey{}, sm)
	return ctx
}

func getEnvFromContext(ctx context.Context) *environment {
	if env, ok := ctx.Value(envKey{}).(*environment); ok {
		return env
	}
	panic("environment not found in context")
}

func getSMFromContext(ctx context.Context) AbstractStateMachine {
	if sm, ok := ctx.Value(smKey{}).(AbstractStateMachine); ok {
		return sm
	}
	panic("StateMachine not found in context")
}

type localState struct {
	env environment
}

func (e *environment) clone() environment {
	machines := make(map[string]AbstractStateMachine)
	for _, sm := range e.machines {
		smc := cloneStateMachine(sm)
		machines[smc.id()] = smc
	}
	queue := make(map[string][]AbstractEvent)
	for smID, events := range e.queue {
		evsc := make([]AbstractEvent, len(events))
		for i, ev := range events {
			evc := cloneEvent(ev)
			evsc[i] = evc
		}
		queue[smID] = evsc
	}

	ec := environment{
		machines: machines,
		queue:    queue,
	}
	return ec
}

func (e *environment) enqueueEvent(target AbstractStateMachine, event AbstractEvent) {
	e.queue[target.id()] = append(e.queue[target.id()], event)
}

func (e *environment) dequeueEvent(smID string) (AbstractEvent, bool) {
	events, ok := e.queue[smID]
	if !ok {
		return nil, false
	}
	if len(events) == 0 {
		return nil, false
	}

	event := events[0]
	e.queue[smID] = events[1:]
	return event, true
}

// SendTo sends an event to a specific state machine.
// This function must be called from within event handlers registered with
// OnEvent, OnEntry, OnExit, OnTransition, or OnHalt functions.
//
// Parameters:
//   - ctx: Context passed to the event handler
//   - target: The state machine that should receive the event
//   - event: The event to send
//
// Example:
//
//	goat.OnEntry(spec, IdleState{}, func(ctx context.Context, sm *MyStateMachine) {
//	    goat.SendTo(ctx, otherSM, Event{Name: "NOTIFY"})
//	})
func SendTo(ctx context.Context, target AbstractStateMachine, event AbstractEvent) {
	env := getEnvFromContext(ctx)
	env.enqueueEvent(target, event)
}

// Goto triggers a state transition for the current state machine.
// This function must be called from within event handlers registered with
// OnEvent, OnEntry, OnExit, OnTransition, or OnHalt functions.
// It automatically generates the necessary sequence of events (Exit, Transition, Entry).
//
// Parameters:
//   - ctx: Context passed to the event handler
//   - state: The target state to transition to
//
// Example:
//
//	goat.OnEvent(spec, IdleState{}, startEvent, func(ctx context.Context, event Event, sm *MyStateMachine) {
//	    goat.Goto(ctx, &ActiveState{Ready: true})
//	})
func Goto(ctx context.Context, state AbstractState) {
	env := getEnvFromContext(ctx)
	sm := getSMFromContext(ctx)
	env.enqueueEvent(sm, &exitEvent{})
	env.enqueueEvent(sm, &transitionEvent{To: state})
	env.enqueueEvent(sm, &entryEvent{})
}

// Halt stops the execution of a specific state machine permanently.
// This function must be called from within event handlers registered with
// OnEvent, OnEntry, OnExit, OnTransition, or OnHalt functions.
// It triggers the haltEvent and any associated cleanup handlers.
//
// Parameters:
//   - ctx: Context passed to the event handler
//   - target: The state machine to halt
//
// Example:
//
//	goat.OnEvent(spec, ActiveState{}, errorEvent, func(ctx context.Context, event Event, sm *MyStateMachine) {
//	    goat.Halt(ctx, sm) // Stop this state machine
//	})
func Halt(ctx context.Context, target AbstractStateMachine) {
	env := getEnvFromContext(ctx)
	env.enqueueEvent(target, &haltEvent{})
}

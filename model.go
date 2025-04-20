package goat

import (
	"fmt"
	"hash/fnv"
	"sort"
	"strconv"
	"strings"
)

type model struct {
	worlds     worlds
	initial    world
	accessible map[worldID][]worldID
	invariants []Invariant
}

type worldID uint64
type worlds map[worldID]world

func (ws worlds) member(w world) bool {
	_, ok := ws[w.id]
	return ok
}

func (ws worlds) insert(w world) {
	ws[w.id] = w
}

type world struct {
	id                 worldID
	env                environment
	invariantViolation bool
}

func newWorld(env environment) world {
	return world{
		id:  id(env),
		env: env,
	}
}

func id(env environment) worldID {
	strs := make([]string, 0)
	smIDs := make([]string, 0)
	for smID := range env.machines {
		smIDs = append(smIDs, smID)
	}
	sort.Strings(smIDs)
	for _, smID := range smIDs {
		sm := env.machines[smID]
		strs = append(strs, fmt.Sprintf("%s=%s;%s", sm.id(), getStateMachineDetails(sm), getStateDetails(sm.currentState())))
	}

	qeNames := make([]string, 0)
	for smID, events := range env.queue {
		for _, event := range events {
			qeNames = append(qeNames, fmt.Sprintf("%s<<%s;%s", smID, getEventName(event), getEventDetails(event)))
		}
	}
	sort.Strings(qeNames)
	strs = append(strs, qeNames...)
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(strings.Join(strs, ",")))
	return worldID(hasher.Sum64())
}

func initialWorld(sms ...AbstractStateMachine) world {
	machines := make(map[string]AbstractStateMachine)
	queue := make(map[string][]AbstractEvent)
	nameCounts := make(map[string]int)

	for _, sm := range sms {
		baseName := sm.id()
		count := nameCounts[baseName]

		var finalID string
		if count == 0 {
			finalID = baseName
		} else {
			finalID = baseName + "_" + strconv.Itoa(count)
		}

		// Update the state machine's ID
		innerSM := getInnerStateMachine(sm)
		innerSM.smID = finalID

		innerSM.EventHandlers = make(map[AbstractState][]handlerInfo)
		for state, builders := range innerSM.HandlerBuilders {
			for _, builderInfo := range builders {
				handler := builderInfo.builder(finalID)
				innerSM.EventHandlers[state] = append(innerSM.EventHandlers[state], handlerInfo{
					event:   builderInfo.event,
					handler: handler,
				})
			}
		}
		innerSM.HandlerBuilders = nil

		machines[finalID] = sm
		queue[finalID] = []AbstractEvent{&entryEvent{}}
		nameCounts[baseName]++
	}

	env := environment{
		machines: machines,
		queue:    queue,
	}

	return newWorld(env)
}

func stepLocal(env environment, smID string) ([]localState, error) {
	ec := env.clone()
	event, ok := ec.dequeueEvent(smID)
	if !ok {
		return nil, nil
	}

	for _, sm := range ec.machines {
		if sm.id() == smID {
			innerSm := getInnerStateMachine(sm)
			if innerSm.halted {
				return []localState{{env: env.clone()}}, nil
			}
			for state, his := range innerSm.EventHandlers {
				if sameState(state, sm.currentState()) {
					lss := make([]localState, 0)
					for _, hi := range his {
						if sameEvent(hi.event, event) {
							states, err := hi.handler.handle(ec, smID, event)
							if err != nil {
								return nil, err
							}
							lss = append(lss, states...)
						}
					}
					if len(lss) > 0 {
						return lss, nil
					}
					return []localState{{env: ec}}, nil
				}
			}
		}
	}
	return []localState{{env: ec}}, nil
}

func stepGlobal(w world) ([]world, error) {
	ws := make([]world, 0)

	env := w.env

	smIDs := make([]string, 0)
	for smID := range env.machines {
		smIDs = append(smIDs, smID)
	}
	sort.Strings(smIDs)

	for _, smID := range smIDs {
		states, err := stepLocal(env, smID)
		if err != nil {
			return nil, err
		}

		for _, state := range states {
			w := newWorld(state.env)
			ws = append(ws, w)
		}
	}

	return ws, nil
}

func newModel(opts ...Option) (model, error) {
	os := newOptions(opts...)
	if len(os.sms) == 0 {
		return model{}, fmt.Errorf("no state machines provided")
	}

	initial := initialWorld(os.sms...)
	return model{
		initial:    initial,
		worlds:     make(worlds),
		accessible: make(map[worldID][]worldID),
		invariants: os.invariants,
	}, nil
}

func (m *model) Solve() error {
	m.worlds.insert(m.initial)
	stack := []world{m.initial}

	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if !m.evaluateInvariants(current) {
			current.invariantViolation = true
			m.worlds[current.id] = current
		}

		acc := make([]worldID, 0)
		nexts, err := stepGlobal(current)
		if err != nil {
			return err
		}
		for _, next := range nexts {
			acc = append(acc, next.id)
			if !m.worlds.member(next) {
				m.worlds.insert(next)
				stack = append(stack, next)
			}
		}
		m.accessible[current.id] = acc
	}

	return nil
}

func (m *model) evaluateInvariants(w world) bool {
	for _, invariant := range m.invariants {
		if !invariant.Evaluate(w) {
			return false
		}
	}
	return true
}

type options struct {
	sms        []AbstractStateMachine
	invariants []Invariant
}

// Option is a configuration option for model checking operations.
// Options are used with Test() and Debug() functions to configure
// state machines, invariants, and other testing parameters.
//
// Use the provided helper functions like WithStateMachines() and
// WithInvariants() to create options.
//
// Example:
//
//	goat.Test(
//	    goat.WithStateMachines(sm1, sm2),
//	    goat.WithInvariants(invariant1),
//	)
type Option interface {
	apply(*options)
}

func newOptions(opts ...Option) *options {
	os := &options{}
	for _, o := range opts {
		o.apply(os)
	}
	return os
}

type optionFunc func(*options)

func (f optionFunc) apply(o *options) {
	f(o)
}

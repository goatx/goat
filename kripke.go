package goat

import (
	"fmt"
	"hash/fnv"
	"io"
	"sort"
	"strings"
)

// Kripke represents a Kripke model for goat.
type Kripke struct {
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
	// id is identifier of the world calculated by the hash of the Environment and the counters.
	id worldID
	// env means "env". env is the Environment of the world in a certain state.
	env Environment
	// invariantViolation indicates if this world violates any invariants
	invariantViolation bool
}

func newWorld(env Environment) world {
	return world{
		id:  id(env),
		env: env,
	}
}

func id(env Environment) worldID {
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
	for _, qeName := range qeNames {
		strs = append(strs, qeName)
	}
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(strings.Join(strs, ",")))
	return worldID(hasher.Sum64())
}

func (w world) label() string {
	strs := make([]string, 0)
	strs = append(strs, "StateMachines:")
	smIDs := make([]string, 0)
	for _, sm := range w.env.machines {
		smIDs = append(smIDs, sm.id())
	}
	sort.Strings(smIDs)
	for _, name := range smIDs {
		sm := w.env.machines[name]
		strs = append(strs, fmt.Sprintf("* %s=%s;%s", getStateMachineName(sm), getStateMachineDetails(sm), getStateDetails(sm.currentState())))
	}

	strs = append(strs, "\nQueuedEvents:")
	smIDs = make([]string, 0)
	for smID, _ := range w.env.queue {
		smIDs = append(smIDs, smID)
	}
	sort.Strings(smIDs)
	for _, smID := range smIDs {
		for _, e := range w.env.queue[smID] {
			sm := w.env.machines[smID]
			strs = append(strs, fmt.Sprintf("* %s<<%s;%s", getStateMachineName(sm), getEventName(e), getEventDetails(e)))
		}
	}
	return strings.Join(strs, "\n")
}

func initialWorld(sms ...AbstractStateMachine) world {
	machines := make(map[string]AbstractStateMachine)
	queue := make(map[string][]AbstractEvent)
	for _, sm := range sms {
		machines[sm.id()] = sm
		queue[sm.id()] = []AbstractEvent{&EntryEvent{}}
	}
	env := Environment{
		machines: machines,
		queue:    queue,
	}

	return newWorld(env)
}

func stepLocal(env Environment, smID string) ([]localState, error) {
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
					} else {
						return []localState{{env: ec}}, nil
					}
				}
			}
		}
	}
	return []localState{{env: ec}}, nil
}

func stepGlobal(w world) ([]world, error) {
	ws := make([]world, 0)

	env := w.env

	for smID := range env.machines {
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

func KripkeModel(opts ...Option) (Kripke, error) {
	os := newOptions(opts...)
	if len(os.sms) == 0 {
		return Kripke{}, fmt.Errorf("no state machines provided")
	}

	initial := initialWorld(os.sms...)
	return Kripke{
		initial:    initial,
		worlds:     make(worlds),
		accessible: make(map[worldID][]worldID),
		invariants: os.invariants,
	}, nil
}

func (k *Kripke) Solve() error {
	k.worlds.insert(k.initial)
	stack := []world{k.initial}

	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if !k.evaluateInvariants(current) {
			current.invariantViolation = true
			k.worlds[current.id] = current
		}

		acc := make([]worldID, 0)
		nexts, err := stepGlobal(current)
		if err != nil {
			return err
		}
		for _, next := range nexts {
			acc = append(acc, next.id)
			if !k.worlds.member(next) {
				k.worlds.insert(next)
				stack = append(stack, next)
			}
		}
		k.accessible[current.id] = acc
	}

	return nil
}

func (k *Kripke) WriteAsDot(w io.Writer) {
	_, _ = fmt.Fprintln(w, "digraph {")
	for id, wld := range k.worlds {
		_, _ = fmt.Fprintf(w, "  %d [ label=\"%s\" ];\n", id, wld.label())
		if id == k.initial.id {
			_, _ = fmt.Fprintf(w, "  %d [ penwidth=5 ];\n", id)
		}
		if wld.invariantViolation {
			_, _ = fmt.Fprintf(w, "  %d [ color=red, penwidth=3 ];\n", id)
		}
	}
	for from, tos := range k.accessible {
		for _, to := range tos {
			_, _ = fmt.Fprintf(w, "  %d -> %d;\n", from, to)
		}
	}
	_, _ = fmt.Fprintln(w, "}")
}

func (k *Kripke) evaluateInvariants(w world) bool {
	for _, invariant := range k.invariants {
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

func WithStateMachines(sms ...AbstractStateMachine) Option {
	return optionFunc(func(o *options) {
		o.sms = sms
	})
}

func WithInvariants(is ...Invariant) Option {
	return optionFunc(func(o *options) {
		o.invariants = is
	})
}
